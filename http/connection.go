//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/cluster"
)

const (
	keyRawResponse = "arangodb-rawResponse"
	keyResponse    = "arangodb-response"
)

// ConnectionConfig provides all configuration options for a HTTP connection.
type ConnectionConfig struct {
	// Endpoints holds 1 or more URL's used to connect to the database.
	// In case of a connection to an ArangoDB cluster, you must provide the URL's of all coordinators.
	Endpoints []string
	// TLSConfig holds settings used to configure a TLS (HTTPS) connection.
	// This is only used for endpoints using the HTTPS scheme.
	TLSConfig *tls.Config
	// Cluster configuration settings
	cluster.ConnectionConfig
}

// NewConnection creates a new HTTP connection based on the given configuration settings.
func NewConnection(config ConnectionConfig) (driver.Connection, error) {
	switch len(config.Endpoints) {
	case 0:
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "You must provide at least 1 endpoint"})
	case 1:
		// Single server
		c, err := newHTTPConnection(config.Endpoints[0], config)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		return c, nil
	}
	// Cluster connection
	servers := make([]driver.Connection, len(config.Endpoints))
	for i, ep := range config.Endpoints {
		c, err := newHTTPConnection(ep, config)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		servers[i] = c
	}
	c, err := cluster.NewConnection(config.ConnectionConfig, servers...)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return c, nil
}

// newHTTPConnection creates a new HTTP connection for a single endpoint and the remainder of the given configuration settings.
func newHTTPConnection(endpoint string, config ConnectionConfig) (driver.Connection, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	transport := &http.Transport{}
	if config.TLSConfig != nil {
		transport.TLSClientConfig = config.TLSConfig
	}
	c := &httpConnection{
		endpoint: *u,
		client: &http.Client{
			Transport: transport,
		},
	}
	return c, nil
}

// httpConnection implements an HTTP + JSON connection to an arangodb server.
type httpConnection struct {
	endpoint url.URL
	client   *http.Client
}

// String returns the endpoint as string
func (c *httpConnection) String() string {
	return c.endpoint.String()
}

// NewRequest creates a new request with given method and path.
func (c *httpConnection) NewRequest(method, path string) (driver.Request, error) {
	switch method {
	case "GET", "POST", "DELETE", "HEAD", "PATCH", "PUT", "OPTIONS":
	// Ok
	default:
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("Invalid method '%s'", method)})
	}
	r := &httpRequest{
		method: method,
		path:   path,
	}
	return r, nil
}

// Do performs a given request, returning its response.
func (c *httpConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	httpReq, ok := req.(*httpRequest)
	if !ok {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "request is not a httpRequest"})
	}
	r, err := httpReq.createHTTPRequest(c.endpoint)
	rctx := ctx
	if rctx == nil {
		rctx = context.Background()
	}
	rctx = httptrace.WithClientTrace(rctx, &httptrace.ClientTrace{
		WroteRequest: httpReq.WroteRequest,
	})
	r = r.WithContext(rctx)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	var rawResponse *[]byte
	if ctx != nil {
		if v := ctx.Value(keyRawResponse); v != nil {
			if buf, ok := v.(*[]byte); ok {
				rawResponse = buf
			}
		}
	}

	httpResp := &httpResponse{resp: resp, rawResponse: rawResponse}
	if ctx != nil {
		if v := ctx.Value(keyResponse); v != nil {
			if respPtr, ok := v.(*driver.Response); ok {
				*respPtr = httpResp
			}
		}
	}
	return httpResp, nil
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *httpConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	if err := json.Unmarshal(data, result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}
