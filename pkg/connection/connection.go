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

package connection

import (
	"context"
	"io"
)

type Connection interface {
	NewRequest(method string, urls ...string) (Request, error)
	NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error)

	DoWithArray(ctx context.Context, request Request) (Response, Array, error)
	DoWithReader(ctx context.Context, request Request) (Response, io.ReadCloser, error)
	DoWithOutput(ctx context.Context, request Request, output interface{}) (Response, error)
	Do(ctx context.Context, request Request) (Response, error)

	Authentication(auth Authentication) error

	Endpoint() string
}

type Request interface {
	Method() string
	URL() string

	Endpoint() string

	SetBody(i interface{}) error
	AddHeader(key, value string)
	AddQuery(key, value string)
	SetFragment(s string)
}

type Response interface {
	Code() int
	Response() interface{}
	Endpoint() string
}
