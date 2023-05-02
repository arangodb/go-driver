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

	velocypack "github.com/arangodb/go-velocypack"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/vst/protocol"
)

// Authentication implements a kind of authentication.
type vstAuthentication interface {
	// PrepareFunc is called when the given Connection has been created.
	// The returned function is then called once.
	PrepareFunc(c *vstConnection) func(ctx context.Context, conn *protocol.Connection) error
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

// newJWTAuthentication creates a JWT token authentication implementation based on the given username & password.
func newRawJWTAuthentication(token string) vstAuthentication {
	return &vstAuthenticationImpl{
		encryption: "rawjwt",
		token:      token,
	}
}

// vstAuthenticationImpl implements VST implementation for JWT & Plain.
type vstAuthenticationImpl struct {
	encryption string
	userName   string
	password   string
	token      string
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
func (a *vstAuthenticationImpl) PrepareFunc(vstConn *vstConnection) func(ctx context.Context, conn *protocol.Connection) error {
	return func(ctx context.Context, conn *protocol.Connection) error {
		var authReq velocypack.Slice
		var err error

		switch a.encryption {
		case "jwt":
			// Call _open/auth
			// Prepare request
			r, err := vstConn.NewRequest("POST", "/_open/auth")
			if err != nil {
				return driver.WithStack(err)
			}
			r.SetBody(jwtOpenRequest{
				UserName: a.userName,
				Password: a.password,
			})

			// Perform request
			resp, err := vstConn.do(ctx, r, conn)
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
		case "rawjwt":
			// Create request
			var b velocypack.Builder
			b.OpenArray()
			b.AddValue(velocypack.NewIntValue(1))          // Version
			b.AddValue(velocypack.NewIntValue(1000))       // Type (1000=Auth)
			b.AddValue(velocypack.NewStringValue("jwt"))   // Encryption type
			b.AddValue(velocypack.NewStringValue(a.token)) // Token
			b.Close()                                      // request
			authReq, err = b.Slice()
			if err != nil {
				return driver.WithStack(err)
			}
		case "plain":
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
		respChan, err := conn.Send(ctx, authReq)
		if err != nil {
			return driver.WithStack(err)
		}

		// Wait for response
		m := <-respChan
		resp, err := newResponse(m.Data, "", nil)
		if err != nil {
			return driver.WithStack(err)
		}
		if err := resp.CheckStatus(200); err != nil {
			return driver.WithStack(err)
		}

		// Ok
		return nil
	}
}
