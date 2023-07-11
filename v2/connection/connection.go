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
)

type Wrapper func(c Connection) Connection

type Factory func() (Connection, error)

type Connection interface {
	// NewRequest initializes Request object
	NewRequest(method string, urls ...string) (Request, error)
	// NewRequestWithEndpoint initializes Request object with specific endpoint
	NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error)
	// Do executes the given Request and parses the response into output
	// If allowed status codes are provided, they will be checked before decoding the response body.
	// In case of mismatch shared.ArangoError will be returned
	Do(ctx context.Context, request Request, output interface{}, allowedStatusCodes ...int) (Response, error)
	// Stream executes the given Request and returns a reader for Response body
	Stream(ctx context.Context, request Request) (Response, io.ReadCloser, error)
	// GetEndpoint returns Endpoint which is currently used to execute requests
	GetEndpoint() Endpoint
	// SetEndpoint changes Endpoint which is used to execute requests
	SetEndpoint(e Endpoint) error
	// GetAuthentication returns Authentication
	GetAuthentication() Authentication
	// SetAuthentication returns Authentication parameters used to execute requests
	SetAuthentication(a Authentication) error
	// Decoder returns Decoder to use for Response body decoding
	Decoder(contentType string) Decoder
}

type Request interface {
	Method() string
	URL() string

	Endpoint() string

	SetBody(i interface{}) error
	AddHeader(key, value string)
	AddQuery(key, value string)

	GetHeader(key string) (string, bool)
	GetQuery(key string) (string, bool)

	SetFragment(s string)
}

type Response interface {
	// Code returns an HTTP compatible status code of the response.
	Code() int
	// Response returns underlying response object
	Response() interface{}
	// Endpoint returns the endpoint that handled the request.
	Endpoint() string
	// Content returns Content-Type
	Content() string
}
