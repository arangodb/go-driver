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
	return d.GetCollection(ctx, name, nil)
}

func (d databaseCollection) GetCollection(ctx context.Context, name string, options *GetCollectionOptions) (Collection, error) {
	col := newCollection(d.db, name)

	if options != nil && options.SkipExistCheck {
		return col, nil
	}

	urlEndpoint := d.db.url("_api", "collection", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return col, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseCollection) CollectionExists(ctx context.Context, name string) (bool, error) {
	_, err := d.GetCollection(ctx, name, nil)
	if err == nil {
		return true, nil
	}

	if shared.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (d databaseCollection) Collections(ctx context.Context) ([]Collection, error) {
	urlEndpoint := d.db.url("_api", "collection")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                []CollectionInfo `json:"result,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		result := make([]Collection, len(response.Result))

		for id, info := range response.Result {
			result[id] = newCollection(d.db, info.Name)
		}

		return result, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseCollection) CreateCollection(ctx context.Context, name string, props *CreateCollectionProperties) (Collection, error) {
	return d.CreateCollectionWithOptions(ctx, name, props, nil)
}

func (d databaseCollection) CreateCollectionWithOptions(ctx context.Context, name string, props *CreateCollectionProperties, options *CreateCollectionOptions) (Collection, error) {
	props.Init()

	urlEndpoint := d.db.url("_api", "collection")
	reqData := struct {
		Name string `json:"name"`
		*CreateCollectionProperties
	}{
		Name:                       name,
		CreateCollectionProperties: props,
	}

	var respData shared.ResponseStruct

	resp, err := connection.CallPost(ctx, d.db.connection(), urlEndpoint, &respData, &reqData, append(d.db.modifiers, options.modifyRequest)...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newCollection(d.db, name), nil
	default:
		return nil, respData.AsArangoErrorWithCode(code)
	}
}
