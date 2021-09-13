//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package connection

import (
	"context"
	"net/http"
)

func RetryOn503(conn Connection, retries int) Connection {
	return NewRetryWrapper(conn, retries, func(response Response, err error) bool {
		if err != nil {
			return false
		}

		return response.Code() == http.StatusServiceUnavailable
	})
}

func NewRetryWrapper(conn Connection, retries int, wrapper RetryWrapper) Connection {
	return &retryWrapper{
		wrapper:    wrapper,
		Connection: conn,
		retries:    retries,
	}
}

type RetryWrapper func(response Response, err error) bool

type retryWrapper struct {
	wrapper RetryWrapper
	Connection

	retries int
}

func (w retryWrapper) Do(ctx context.Context, request Request, output interface{}) (Response, error) {
	var r Response
	var err error
	for i := 0; i < w.retries; i++ {
		r, err = w.Connection.Do(ctx, request, output)

		if w.wrapper(r, err) {
			continue
		}

		if err == nil {
			return r, nil
		}

		return nil, err
	}

	return r, err
}
