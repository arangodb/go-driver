//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
)

type EncodingCodec interface {
}

type Wrapper func(c Connection) Connection

type Factory func() (Connection, error)

type ArangoDBConfiguration struct {
	// ArangoQueueTimeoutEnabled is used to enable Queue timeout on the server side.
	// If ArangoQueueTimeoutEnabled is used, then its value takes precedence.
	// In another case value of context.Deadline will be taken
	ArangoQueueTimeoutEnabled bool

	// ArangoQueueTimeout defines max queue timeout on the server side
	ArangoQueueTimeoutSec uint

	// DriverFlags configure additional flags for the `x-arango-driver` header
	DriverFlags []string

	// Compression is used to enable compression between client and server
	Compression *CompressionConfig
}

// CompressionConfig is used to enable compression for the connection
type CompressionConfig struct {
	// CompressionConfig is used to enable compression for the requests
	CompressionType CompressionType

	// ResponseCompressionEnabled is used to enable compression for the responses (requires server side adjustments)
	ResponseCompressionEnabled bool

	// RequestCompressionEnabled is used to enable compression for the requests
	RequestCompressionEnabled bool

	// RequestCompressionLevel - Sets the compression level between -1 and 9
	// Default: 0 (NoCompression). For Reference see: https://pkg.go.dev/compress/flate#pkg-constants
	RequestCompressionLevel int
}

type CompressionType string

const (

	// RequestCompressionTypeGzip is used to enable gzip compression
	RequestCompressionTypeGzip CompressionType = "gzip"

	// RequestCompressionTypeDeflate is used to enable deflate compression
	RequestCompressionTypeDeflate CompressionType = "deflate"
)

type Connection interface {
	// NewRequest initializes Request object
	NewRequest(method string, urls ...string) (Request, error)

	// NewRequestWithEndpoint initializes a Request object with a specific endpoint
	NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error)

	// Do execute the given Request and parses the response into output
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

	// GetConfiguration returns the configuration for the connection to database
	GetConfiguration() ArangoDBConfiguration

	// SetConfiguration sets the configuration for the connection to database
	SetConfiguration(config ArangoDBConfiguration)
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

	// Header gets the first value associated with the given key.
	// If there are no values associated with the key, Get returns "".
	Header(name string) string

	RawResponse() *http.Response
}
