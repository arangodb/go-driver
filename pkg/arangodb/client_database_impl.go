//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
	"net/http"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/pkg/connection"
	"github.com/pkg/errors"
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

func (c clientDatabase) CreateDatabase(ctx context.Context, name string, options *driver.CreateDatabaseOptions) (Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database")

	createRequest := struct {
		*driver.CreateDatabaseOptions `json:",inline,omitempty"`
		Name                          string `json:"name"`
	}{
		CreateDatabaseOptions: options,
		Name:                  name,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, nil, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusCreated:
		return newDatabase(c.client, name), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c clientDatabase) AccessibleDatabases(ctx context.Context) ([]Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database", "user")
	return c.databases(ctx, url)
}

func (c clientDatabase) DatabaseExists(ctx context.Context, name string) (bool, error) {
	_, err := c.Database(ctx, name)
	if err == nil {
		return true, nil
	}

	if connection.IsNotFoundError(err) {
		return false, nil
	}

	return false, err
}

func (c clientDatabase) Databases(ctx context.Context) ([]Database, error) {
	url := connection.NewUrl("_db", "_system", "_api", "database")
	return c.databases(ctx, url)
}

func (c clientDatabase) Database(ctx context.Context, name string) (Database, error) {
	url := connection.NewUrl("_db", name, "_api", "database", "current")
	resp, err := connection.CallGet(ctx, c.client.connection, url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return newDatabase(c.client, name), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c clientDatabase) databases(ctx context.Context, url string) ([]Database, error) {
	databases := struct {
		ResponseStruct `json:",inline"`
		Result         []string `json:"result,omitempty"`
	}{}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &databases)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		dbs := make([]Database, len(databases.Result))

		for id, name := range databases.Result {
			dbs[id] = newDatabase(c.client, name)
		}

		return dbs, nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}

}
