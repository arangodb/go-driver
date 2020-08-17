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

func newDatabaseCollection(db *database) *databaseCollection {
	return &databaseCollection{
		db: db,
	}
}

var _ DatabaseCollection = &databaseCollection{}

type databaseCollection struct {
	db *database
}

func (d databaseCollection) Collection(ctx context.Context, name string) (Collection, error) {
	url := d.db.url("_api", "collection", name)
	resp, err := connection.CallGet(ctx, d.db.connection(), url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return newCollection(d.db, name), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (d databaseCollection) CollectionExists(ctx context.Context, name string) (bool, error) {
	_, err := d.Collection(ctx, name)
	if err == nil {
		return true, nil
	}

	if connection.IsNotFoundError(err) {
		return false, nil
	}

	return false, err
}

func (d databaseCollection) Collections(ctx context.Context) ([]Collection, error) {
	url := d.db.url("_api", "collection")

	response := struct {
		Response
		Result []driver.CollectionInfo `json:"result,omitempty"`
	}{}

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		colls := make([]Collection, len(response.Result))

		for id, info := range response.Result {
			colls[id] = newCollection(d.db, info.Name)
		}

		return colls, nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (d databaseCollection) CreateCollection(ctx context.Context, name string, options *driver.CreateCollectionOptions) (Collection, error) {
	url := d.db.url("_api", "collection")
	reqData := struct {
		Name string `json:"name"`
		*driver.CreateCollectionOptions
	}{
		Name:                    name,
		CreateCollectionOptions: options,
	}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, nil, &reqData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return newCollection(d.db, name), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}
