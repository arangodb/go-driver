//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package connection

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/pkg/errors"
)

type AuthenticationGetter func(ctx context.Context, conn Connection) (Authentication, error)

func WrapAuthentication(getter AuthenticationGetter) Wrapper {
	return func(c Connection) Connection {
		return &wrapAuthentication{
			getter:     getter,
			Connection: c,
		}
	}
}

type wrapAuthentication struct {
	getter AuthenticationGetter
	Connection

	lock sync.Mutex
}

func (w *wrapAuthentication) Do(ctx context.Context, request Request, output interface{}, allowedStatusCodes ...int) (Response, error) {
	r, err := w.Connection.Do(ctx, request, output, allowedStatusCodes...)
	if err != nil {
		return r, err
	}

	if r.Code() != http.StatusUnauthorized {
		return r, err
	}

	if err := w.reAuth(ctx); err != nil {
		return nil, err
	}

	return w.Connection.Do(ctx, request, output, allowedStatusCodes...)
}

// Stream performs HTTP request.
// It returns the response and body reader to read the data from there.
// The caller is responsible to free the response body.
func (w *wrapAuthentication) Stream(ctx context.Context, request Request) (Response, io.ReadCloser, error) {
	r, body, err := w.Connection.Stream(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	if r.Code() != http.StatusUnauthorized {
		return r, body, err
	}

	if body != nil {
		body.Close()
	}

	if err := w.reAuth(ctx); err != nil {
		return nil, nil, err
	}

	return w.Connection.Stream(ctx, request)
}

func (w *wrapAuthentication) reAuth(ctx context.Context) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if err := w.Connection.SetAuthentication(nil); err != nil {
		return err
	}

	if a, err := w.getter(ctx, w.Connection); err != nil {
		return err
	} else {
		if err := w.Connection.SetAuthentication(a); err != nil {
			return err
		}
	}

	return nil
}

func (w *wrapAuthentication) SetAuthentication(_ Authentication) error {
	return errors.Errorf("Unable to override authentication when it wrapped by Authentication wrapper")
}
