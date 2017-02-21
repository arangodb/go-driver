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

package driver

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Authentication implements a kind of authentication.
type Authentication interface {
	// Prepare is called before the first request of the given connection is made.
	Prepare(ctx context.Context, conn Connection) error

	// Configure is called for every request made on a connection.
	Configure(req Request) error
}

// BasicAuthentication creates an authentication implementation based on the given username & password.
func BasicAuthentication(userName, password string) Authentication {
	auth := fmt.Sprintf("%s:%s", userName, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return &basicAuthentication{
		authorizationValue: "Basic " + encoded,
	}
}

// JWTAuthentication creates a JWT token authentication implementation based on the given username & password.
func JWTAuthentication(userName, password string) Authentication {
	return &jwtAuthentication{
		userName: userName,
		password: password,
	}
}

// basicAuthentication implements HTTP Basic authentication.
type basicAuthentication struct {
	authorizationValue string
}

// Prepare is called before the first request of the given connection is made.
func (a *basicAuthentication) Prepare(ctx context.Context, conn Connection) error {
	// No need to do anything here
	return nil
}

// Configure is called for every request made on a connection.
func (a *basicAuthentication) Configure(req Request) error {
	req.SetHeader("Authorization", a.authorizationValue)
	return nil
}

// jwtAuthentication implements JWT token authentication.
type jwtAuthentication struct {
	userName string
	password string
	token    string
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
func (a *jwtAuthentication) Prepare(ctx context.Context, conn Connection) error {
	// Prepare request
	r, err := conn.NewRequest("POST", "/_open/auth")
	if err != nil {
		return err
	}
	r.SetBody(jwtOpenRequest{
		UserName: a.userName,
		Password: a.password,
	})

	// Perform request
	resp, err := conn.Do(ctx, r)
	if err != nil {
		return err
	}
	if err := resp.CheckStatus(200); err != nil {
		return err
	}

	// Parse response
	var data jwtOpenResponse
	if err := resp.ParseBody(&data); err != nil {
		return err
	}

	// Store token
	a.token = data.Token

	// Ok
	return nil
}

// Configure is called for every request made on a connection.
func (a *jwtAuthentication) Configure(req Request) error {
	req.SetHeader("Authorization", "bearer "+a.token)
	return nil
}
