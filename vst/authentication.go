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
	"fmt"
	"sync"
	"sync/atomic"

	driver "github.com/arangodb/go-driver"
	velocypack "github.com/arangodb/go-velocypack"
)

// Authentication implements a kind of authentication.
type vstAuthentication interface {
	// Prepare is called before the first request of the given connection is made.
	Prepare(ctx context.Context, conn driver.Connection) error

	// Configure is called for every request made on a connection.
	//Configure(req driver.Request) error
}

// newBasicAuthentication creates an authentication implementation based on the given username & password.
func newBasicAuthentication(userName, password string) vstAuthentication {
	return &vstAuthenticationImpl{
		encryption: "plain",
		userName:   userName,
		password:   password,
	}
}

// newJWTAuthentication creates a JWT token authentication implementation based on the given username & password.
func newJWTAuthentication(userName, password string) vstAuthentication {
	return &vstAuthenticationImpl{
		encryption: "jwt",
		userName:   userName,
		password:   password,
	}
}

// vstAuthenticationImpl implements VST implementation for JWT & Plain.
type vstAuthenticationImpl struct {
	encryption string
	userName   string
	password   string
}

type jwtOpenRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type jwtOpenResponse struct {
	Token              string `json:"jwt"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}

// Prepare is called before the first request of the given connection is made.
func (a *vstAuthenticationImpl) Prepare(ctx context.Context, conn driver.Connection) error {
	var authReq velocypack.Slice
	var err error

	if a.encryption == "jwt" {
		// Call _open/auth
		// Prepare request
		r, err := conn.NewRequest("POST", "/_open/auth")
		if err != nil {
			return driver.WithStack(err)
		}
		r.SetBody(jwtOpenRequest{
			UserName: a.userName,
			Password: a.password,
		})

		// Perform request
		resp, err := conn.Do(ctx, r)
		if err != nil {
			return driver.WithStack(err)
		}
		if err := resp.CheckStatus(200); err != nil {
			return driver.WithStack(err)
		}

		// Parse response
		var data jwtOpenResponse
		if err := resp.ParseBody("", &data); err != nil {
			return driver.WithStack(err)
		}

		// Create request
		var b velocypack.Builder
		b.OpenArray()
		b.AddValue(velocypack.NewIntValue(1))             // Version
		b.AddValue(velocypack.NewIntValue(1000))          // Type (1000=Auth)
		b.AddValue(velocypack.NewStringValue("jwt"))      // Encryption type
		b.AddValue(velocypack.NewStringValue(data.Token)) // Token
		b.Close()                                         // request
		authReq, err = b.Slice()
		if err != nil {
			return driver.WithStack(err)
		}
	} else {
		// Create request
		var b velocypack.Builder
		b.OpenArray()
		b.AddValue(velocypack.NewIntValue(1))               // Version
		b.AddValue(velocypack.NewIntValue(1000))            // Type (1000=Auth)
		b.AddValue(velocypack.NewStringValue(a.encryption)) // Encryption type
		b.AddValue(velocypack.NewStringValue(a.userName))   // Username
		b.AddValue(velocypack.NewStringValue(a.password))   // Password
		b.Close()                                           // request
		authReq, err = b.Slice()
		if err != nil {
			return driver.WithStack(err)
		}
	}

	// Send request
	vstConn, ok := conn.(*vstConnection)
	if !ok {
		return driver.WithStack(fmt.Errorf("*vstConnection expected"))
	}
	respChan, err := vstConn.transport.Send(ctx, authReq)
	if err != nil {
		return driver.WithStack(err)
	}

	// Wait for response
	m := <-respChan
	resp, err := newResponse(m, vstConn.endpoint.String(), nil)
	if err != nil {
		return driver.WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return driver.WithStack(err)
	}

	// Ok
	return nil
}

// newAuthenticatedConnection creates a Connection that applies the given connection on the given underlying connection.
func newAuthenticatedConnection(conn driver.Connection, auth vstAuthentication) (driver.Connection, error) {
	if conn == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "conn is nil"})
	}
	if auth == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "auth is nil"})
	}
	return &authenticatedConnection{
		conn: conn,
		auth: auth,
	}, nil
}

// authenticatedConnection implements authentication behavior for connections.
type authenticatedConnection struct {
	conn         driver.Connection // Un-authenticated connection
	auth         vstAuthentication
	prepareMutex sync.Mutex
	prepared     int32
}

// NewRequest creates a new request with given method and path.
func (c *authenticatedConnection) NewRequest(method, path string) (driver.Request, error) {
	r, err := c.conn.NewRequest(method, path)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return r, nil
}

// Do performs a given request, returning its response.
func (c *authenticatedConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if atomic.LoadInt32(&c.prepared) == 0 {
		// Probably we're not yet prepared
		if err := c.prepare(ctx); err != nil {
			// Authentication failed
			return nil, driver.WithStack(err)
		}
	}
	// Configure the request for authentication.
	/*if err := c.auth.Configure(req); err != nil {
		// Failed to configure request for authentication
		return nil, driver.WithStack(err)
	}*/
	// Do the authenticated request
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return resp, nil
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *authenticatedConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	if err := c.conn.Unmarshal(data, result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Endpoints returns the endpoints used by this connection.
func (c *authenticatedConnection) Endpoints() []string {
	return c.conn.Endpoints()
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *authenticatedConnection) UpdateEndpoints(endpoints []string) error {
	if err := c.conn.UpdateEndpoints(endpoints); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Configure the authentication used for this connection.
func (c *authenticatedConnection) SetAuthentication(auth driver.Authentication) (driver.Connection, error) {
	result, err := c.conn.SetAuthentication(auth)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return result, nil
}

// prepare calls Authentication.Prepare if needed.
func (c *authenticatedConnection) prepare(ctx context.Context) error {
	c.prepareMutex.Lock()
	defer c.prepareMutex.Unlock()
	if c.prepared == 0 {
		// We need to prepare first
		if err := c.auth.Prepare(ctx, c.conn); err != nil {
			// Authentication failed
			return driver.WithStack(err)
		}
		// We're now prepared
		atomic.StoreInt32(&c.prepared, 1)
	} else {
		// We're already prepared, do nothing
	}
	return nil
}
