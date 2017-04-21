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

package vst

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/cluster"
	"github.com/arangodb/go-driver/util"
	"github.com/arangodb/go-driver/vst/protocol"
	velocypack "github.com/arangodb/go-velocypack"
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
	// Transport allows the use of a custom round tripper.
	// If Transport is not of type `*http.Transport`, the `TLSConfig` property is not used.
	// Otherwise a `TLSConfig` property other than `nil` will overwrite the `TLSClientConfig`
	// property of `Transport`.
	Transport protocol.TransportConfig
	// Cluster configuration settings
	cluster.ConnectionConfig
}

// NewConnection creates a new Velocystream connection based on the given configuration settings.
func NewConnection(config ConnectionConfig) (driver.Connection, error) {
	c, err := cluster.NewConnection(config.ConnectionConfig, func(endpoint string) (driver.Connection, error) {
		conn, err := newVSTConnection(endpoint, config)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		return conn, nil
	}, config.Endpoints)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return c, nil
}

// newVSTConnection creates a new Velocystream connection for a single endpoint and the remainder of the given configuration settings.
func newVSTConnection(endpoint string, config ConnectionConfig) (driver.Connection, error) {
	endpoint = util.FixupEndpointURLScheme(endpoint)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	hostAddr := u.Host
	tlsConfig := config.TLSConfig
	switch strings.ToLower(u.Scheme) {
	case "http":
		tlsConfig = nil
	case "https":
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
	}
	c := &vstConnection{
		endpoint:  *u,
		transport: protocol.NewTransport(hostAddr, tlsConfig, config.Transport),
	}
	return c, nil
}

// vstConnection implements an Velocystream connection to an arangodb server.
type vstConnection struct {
	endpoint  url.URL
	transport *protocol.Transport
}

// String returns the endpoint as string
func (c *vstConnection) String() string {
	return c.endpoint.String()
}

// NewRequest creates a new request with given method and path.
func (c *vstConnection) NewRequest(method, path string) (driver.Request, error) {
	switch method {
	case "GET", "POST", "DELETE", "HEAD", "PATCH", "PUT", "OPTIONS":
	// Ok
	default:
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("Invalid method '%s'", method)})
	}
	r := &vstRequest{
		method: method,
		path:   path,
	}
	return r, nil
}

// Do performs a given request, returning its response.
func (c *vstConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	vstReq, ok := req.(*vstRequest)
	if !ok {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "request is not a *vstRequest"})
	}
	msgParts, err := vstReq.createMessageParts()
	if err != nil {
		return nil, driver.WithStack(err)
	}
	resp, err := c.transport.Send(ctx, msgParts...)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	// All data was send now
	vstReq.WroteRequest()

	// Wait for response
	msg, ok := <-resp
	if !ok {
		// Message was cancelled / timeout
		return nil, driver.WithStack(context.DeadlineExceeded)
	}

	//fmt.Printf("Received msg: %d\n", msg.ID)
	var rawResponse *[]byte
	if ctx != nil {
		if v := ctx.Value(keyRawResponse); v != nil {
			if buf, ok := v.(*[]byte); ok {
				rawResponse = buf
			}
		}
	}

	vstResp, err := newResponse(msg, c.endpoint.String(), rawResponse)
	if err != nil {
		fmt.Printf("Cannot decode msg %d: %#v\n", msg.ID, err)
		return nil, driver.WithStack(err)
	}
	if ctx != nil {
		if v := ctx.Value(keyResponse); v != nil {
			if respPtr, ok := v.(*driver.Response); ok {
				*respPtr = vstResp
			}
		}
	}
	return vstResp, nil
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *vstConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	ct := driver.ContentTypeVelocypack
	if len(data) >= 2 {
		// Poor mans auto detection of json
		l := len(data)
		if (data[0] == '{' && data[l-1] == '}') || (data[0] == '[' && data[l-1] == ']') {
			ct = driver.ContentTypeJSON
		}
	}
	switch ct {
	case driver.ContentTypeJSON:
		if err := json.Unmarshal(data, result); err != nil {
			return driver.WithStack(err)
		}
	case driver.ContentTypeVelocypack:
		//panic(velocypack.Slice(data))
		if err := velocypack.Unmarshal(velocypack.Slice(data), result); err != nil {
			return driver.WithStack(err)
		}
	default:
		return driver.WithStack(fmt.Errorf("Unsupported content type %d", int(ct)))
	}
	return nil
}

// Endpoints returns the endpoints used by this connection.
func (c *vstConnection) Endpoints() []string {
	return []string{c.endpoint.String()}
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *vstConnection) UpdateEndpoints(endpoints []string) error {
	// Do nothing here.
	// The real updating is done in cluster Connection.
	return nil
}
