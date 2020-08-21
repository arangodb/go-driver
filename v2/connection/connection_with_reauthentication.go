//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package connection

import (
	"context"
	"net/http"
	"sync"

	"github.com/pkg/errors"
)

type AuthenticationGetter func(ctx context.Context, conn Connection) (Authentication, error)

func WrapAuthentication(getter AuthenticationGetter) Wrapper {
	return func(c Connection) Connection {
		return &wrapAuthentication{
			getter:     getter,
			connection: c,
		}
	}
}

type wrapAuthentication struct {
	getter     AuthenticationGetter
	connection Connection

	lock sync.Mutex
}

func (w wrapAuthentication) Decoder(contentType string) Decoder {
	return w.connection.Decoder(contentType)
}

func (w wrapAuthentication) Do(ctx context.Context, request Request, output interface{}) (Response, error) {
	r, err := w.connection.Do(ctx, request, output)

	if err != nil {
		return nil, err
	}

	if r.Code() != http.StatusUnauthorized {
		return r, err
	}

	if err := w.reAuth(ctx); err != nil {
		return nil, err
	}

	return w.connection.Do(ctx, request, output)
}

func (w *wrapAuthentication) reAuth(ctx context.Context) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if err := w.connection.SetAuthentication(nil); err != nil {
		return err
	}

	if a, err := w.getter(ctx, w.connection); err != nil {
		return err
	} else {
		if err := w.connection.SetAuthentication(a); err != nil {
			return err
		}
	}

	return nil
}

func (w wrapAuthentication) NewRequest(method string, urls ...string) (Request, error) {
	return w.connection.NewRequest(method, urls...)
}

func (w wrapAuthentication) NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error) {
	return w.connection.NewRequestWithEndpoint(endpoint, method, urls...)
}

func (w wrapAuthentication) GetEndpoint() Endpoint {
	return w.connection.GetEndpoint()
}

func (w wrapAuthentication) SetEndpoint(e Endpoint) error {
	return w.connection.SetEndpoint(e)
}

func (w wrapAuthentication) GetAuthentication() Authentication {
	return w.connection.GetAuthentication()
}

func (w wrapAuthentication) SetAuthentication(a Authentication) error {
	return errors.Errorf("Unable to override authentication when it rapped by Authentication wrapper")
}
