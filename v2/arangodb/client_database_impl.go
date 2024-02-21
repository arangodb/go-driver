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

package arangodb

import (
	"context"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newClientDatabase(client *client) *clientDatabase {
	return &clientDatabase{
		client: client,
	}
}

var _ ClientDatabase = &clientDatabase{}

type clientDatabase struct {
	client *client
}

func (c clientDatabase) CreateDatabase(ctx context.Context, name string, options *CreateDatabaseOptions) (Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database")

	createRequest := struct {
		*CreateDatabaseOptions `json:",inline,omitempty"`
		Name                   string `json:"name"`
	}{
		CreateDatabaseOptions: options,
		Name:                  name,
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return newDatabase(c.client, name), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientDatabase) AccessibleDatabases(ctx context.Context) ([]Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database", "user")
	return c.databases(ctx, url)
}

func (c clientDatabase) DatabaseExists(ctx context.Context, name string) (bool, error) {
	_, err := c.GetDatabase(ctx, name, nil)
	if err == nil {
		return true, nil
	}

	if shared.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (c clientDatabase) Databases(ctx context.Context) ([]Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database")
	return c.databases(ctx, url)
}

func (c clientDatabase) Database(ctx context.Context, name string) (Database, error) {
	return c.GetDatabase(ctx, name, nil)
}

func (c clientDatabase) GetDatabase(ctx context.Context, name string, options *GetDatabaseOptions) (Database, error) {
	db := newDatabase(c.client, name)

	if options != nil && options.SkipExistCheck {
		return db, nil
	}

	urlEndpoint := connection.NewUrl("_db", url.PathEscape(name), "_api", "database", "current")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		VersionInfo           `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return db, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c clientDatabase) databases(ctx context.Context, url string) ([]Database, error) {
	databases := struct {
		shared.ResponseStruct `json:",inline"`
		Result                []string `json:"result,omitempty"`
	}{}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &databases)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		dbs := make([]Database, len(databases.Result))

		for id, name := range databases.Result {
			dbs[id] = newDatabase(c.client, name)
		}

		return dbs, nil
	default:
		return nil, databases.AsArangoErrorWithCode(code)
	}
}
