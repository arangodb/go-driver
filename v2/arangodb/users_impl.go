//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newClientUser(client *client) *clientUser {
	return &clientUser{
		client: client,
	}
}

var _ ClientUsers = &clientUser{}

type clientUser struct {
	client *client
}

type userResponse struct {
	Name   string          `json:"user,omitempty"`
	Active bool            `json:"active,omitempty"`
	Extra  json.RawMessage `json:"extra,omitempty"`
}

func (c clientUser) User(ctx context.Context, name string) (User, error) {
	urlEndpoint := connection.NewUrl("_api", "user", url.PathEscape(name))

	response := struct {
		userResponse          `json:",inline"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newUser(c.client, &response.userResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientUser) UserExists(ctx context.Context, name string) (bool, error) {
	_, err := c.User(ctx, name)
	if err == nil {
		return true, nil
	}

	if shared.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (c clientUser) Users(ctx context.Context) ([]User, error) {
	urlEndpoint := connection.NewUrl("_api", "user")

	response := struct {
		Result                []userResponse `json:"result,omitempty"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		result := make([]User, len(response.Result))
		for id, user := range response.Result {
			result[id] = newUser(c.client, &user)
		}
		return result, nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientUser) CreateUser(ctx context.Context, name string, options *UserOptions) (User, error) {
	urlEndpoint := connection.NewUrl("_api", "user")

	createRequest := struct {
		*UserOptions `json:",inline,omitempty"`
		Name         string `json:"user"`
	}{
		UserOptions: options,
		Name:        name,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
		userResponse          `json:",inline"`
	}{}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return newUser(c.client, &response.userResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientUser) ReplaceUser(ctx context.Context, name string, options *UserOptions) (User, error) {
	urlEndpoint := connection.NewUrl("_api", "user", url.PathEscape(name))

	createRequest := struct {
		*UserOptions `json:",inline,omitempty"`
	}{
		UserOptions: options,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
		userResponse          `json:",inline"`
	}{}

	resp, err := connection.CallPut(ctx, c.client.connection, urlEndpoint, &response, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newUser(c.client, &response.userResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientUser) UpdateUser(ctx context.Context, name string, options *UserOptions) (User, error) {
	urlEndpoint := connection.NewUrl("_api", "user", url.PathEscape(name))

	createRequest := struct {
		*UserOptions `json:",inline,omitempty"`
	}{
		UserOptions: options,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
		userResponse          `json:",inline"`
	}{}

	resp, err := connection.CallPatch(ctx, c.client.connection, urlEndpoint, &response, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newUser(c.client, &response.userResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientUser) RemoveUser(ctx context.Context, name string) error {
	urlEndpoint := connection.NewUrl("_api", "user", url.PathEscape(name))

	resp, err := connection.CallDelete(ctx, c.client.connection, urlEndpoint, nil)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}
