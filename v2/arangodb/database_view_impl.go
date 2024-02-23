//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newDatabaseView(db *database) *databaseView {
	return &databaseView{
		db: db,
	}
}

var _ DatabaseView = &databaseView{}

type databaseView struct {
	db *database
}

func (d databaseView) View(ctx context.Context, name string) (View, error) {
	urlEndpoint := d.db.url("_api", "view", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`

		Name string   `json:"name,omitempty"`
		Type ViewType `json:"type,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newView(d.db, response.Name, response.Type), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseView) ViewExists(ctx context.Context, name string) (bool, error) {
	urlEndpoint := d.db.url("_api", "view", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return false, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return true, nil
	default:
		err = response.AsArangoErrorWithCode(code)
		if shared.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
}

func (d databaseView) Views(ctx context.Context) (ViewsResponseReader, error) {
	urlEndpoint := d.db.url("_api", "view")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Views                 connection.Array `json:"result,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return newViewsResponseReader(d.db, &response.Views), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseView) ViewsAll(ctx context.Context) ([]View, error) {
	urlEndpoint := d.db.url("_api", "view")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Views                 []ViewBase `json:"result,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		result := make([]View, len(response.Views))
		for id, view := range response.Views {
			result[id] = newView(d.db, view.Name, view.Type)
		}
		return result, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseView) CreateArangoSearchView(ctx context.Context, name string, options *ArangoSearchViewProperties) (ArangoSearchView, error) {
	urlEndpoint := d.db.url("_api", "view")
	input := struct {
		Name string   `json:"name"`
		Type ViewType `json:"type"`
		ArangoSearchViewProperties
	}{
		Name: name,
		Type: ViewTypeArangoSearch,
	}
	if options != nil {
		input.ArangoSearchViewProperties = *options
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}
	resp, err := connection.CallPost(ctx, d.db.connection(), urlEndpoint, &response, input)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusCreated:
		v := newView(d.db, name, input.Type)
		result, err := v.ArangoSearchView()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return result, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseView) CreateArangoSearchAliasView(ctx context.Context, name string, options *ArangoSearchAliasViewProperties) (ArangoSearchViewAlias, error) {
	urlEndpoint := d.db.url("_api", "view")
	input := struct {
		Name string   `json:"name"`
		Type ViewType `json:"type"`
		ArangoSearchAliasViewProperties
	}{
		Name: name,
		Type: ViewTypeSearchAlias,
	}
	if options != nil {
		input.ArangoSearchAliasViewProperties = *options
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}
	resp, err := connection.CallPost(ctx, d.db.connection(), urlEndpoint, &response, input)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusCreated:
		v := newView(d.db, name, input.Type)
		result, err := v.ArangoSearchViewAlias()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return result, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func newViewsResponseReader(db *database, arr *connection.Array) ViewsResponseReader {
	return &viewsResponseReader{
		array: arr,
		db:    db,
	}
}

type viewsResponseReader struct {
	array *connection.Array
	db    *database
}

func (reader *viewsResponseReader) Read() (View, error) {
	if !reader.array.More() {
		return nil, shared.NoMoreDocumentsError{}
	}

	viewResponse := struct {
		Name string   `json:"name,omitempty"`
		Type ViewType `json:"type,omitempty"`
	}{}

	if err := reader.array.Unmarshal(newUnmarshalInto(&viewResponse)); err != nil {
		if err == io.EOF {
			return nil, shared.NoMoreDocumentsError{}
		}
		return nil, err
	}

	return newView(reader.db, viewResponse.Name, viewResponse.Type), nil
}
