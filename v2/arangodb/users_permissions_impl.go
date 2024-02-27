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

func newUser(client *client, userData *userResponse) *user {
	return &user{
		client:   client,
		userData: userData,
	}
}

type user struct {
	client   *client
	userData *userResponse
}

// creates the path to this User (`_api/user/<user-name>`)
func (u user) url(parts ...string) string {
	p := append([]string{"_api", "user", url.PathEscape(u.Name())}, parts...)
	return connection.NewUrl(p...)
}

func (u user) Name() string {
	return u.userData.Name
}

func (u user) IsActive() bool {
	return u.userData.Active
}

func (u user) Extra(result interface{}) error {
	return json.Unmarshal(u.userData.Extra, result)
}

func (u user) AccessibleDatabases(ctx context.Context) (map[string]Grant, error) {
	urlEndpoint := u.url("database")

	response := struct {
		Result                map[string]Grant `json:"result,omitempty"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, u.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return nil, response.AsArangoError()
	}
}

func (u user) AccessibleDatabasesFull(ctx context.Context) (map[string]DatabasePermissions, error) {
	urlEndpoint := u.url("database")

	response := struct {
		Result                map[string]DatabasePermissions `json:"result,omitempty"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, u.client.connection, urlEndpoint, &response, connection.WithQuery("full", "true"))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return nil, response.AsArangoError()
	}
}

func (u user) GetDatabaseAccess(ctx context.Context, db string) (Grant, error) {
	urlEndpoint := u.url("database", url.PathEscape(db))

	response := struct {
		Result                Grant `json:"result,omitempty"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, u.client.connection, urlEndpoint, &response)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return "", response.AsArangoError()
	}
}

func (u user) GetCollectionAccess(ctx context.Context, db, col string) (Grant, error) {
	urlEndpoint := u.url("database", url.PathEscape(db), url.PathEscape(col))

	response := struct {
		Result                Grant `json:"result,omitempty"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, u.client.connection, urlEndpoint, &response)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return "", response.AsArangoError()
	}
}

func (u user) SetDatabaseAccess(ctx context.Context, db string, access Grant) error {
	urlEndpoint := u.url("database", url.PathEscape(db))

	setRequest := struct {
		Grant `json:"grant"`
	}{
		Grant: access,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallPut(ctx, u.client.connection, urlEndpoint, &response, &setRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoError()
	}
}

func (u user) SetCollectionAccess(ctx context.Context, db, col string, access Grant) error {
	urlEndpoint := u.url("database", url.PathEscape(db), url.PathEscape(col))

	setRequest := struct {
		Grant `json:"grant"`
	}{
		Grant: access,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallPut(ctx, u.client.connection, urlEndpoint, &response, &setRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoError()
	}
}

func (u user) RemoveDatabaseAccess(ctx context.Context, db string) error {
	urlEndpoint := u.url("database", url.PathEscape(db))

	resp, err := connection.CallDelete(ctx, u.client.connection, urlEndpoint, nil)
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

func (u user) RemoveCollectionAccess(ctx context.Context, db, col string) error {
	urlEndpoint := u.url("database", url.PathEscape(db), url.PathEscape(col))

	resp, err := connection.CallDelete(ctx, u.client.connection, urlEndpoint, nil)
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
