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

package arangodb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newView(db *database, name string, viewType ViewType, modifiers ...connection.RequestModifier) *view {
	d := &view{db: db, name: name, viewType: viewType, modifiers: append(db.modifiers, modifiers...)}

	return d
}

var _ View = &view{}

type view struct {
	name      string
	viewType  ViewType
	db        *database
	modifiers []connection.RequestModifier
}

func (v *view) Name() string {
	return v.name
}

func (v *view) Database() Database {
	return v.db
}

func (v *view) Type() ViewType {
	return v.viewType
}

func (v *view) Rename(ctx context.Context, newName string) error {
	if newName == "" {
		return errors.WithStack(shared.InvalidArgumentError{Message: "newName is empty"})
	}

	urlEndpoint := v.db.url("_api", "view", url.PathEscape(v.name), "rename")
	input := struct {
		Name string `json:"name"`
	}{
		Name: newName,
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, v.db.connection(), urlEndpoint, &response, input, v.db.modifiers...)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		v.name = newName
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (v *view) Remove(ctx context.Context) error {
	return v.RemoveWithOptions(ctx, nil)
}

func (v *view) RemoveWithOptions(ctx context.Context, opts *RemoveViewOptions) error {
	urlEndpoint := v.db.url("_api", "view", url.PathEscape(v.name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, v.db.connection(), urlEndpoint, &response, append(v.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (v *view) ArangoSearchView() (ArangoSearchView, error) {
	if v.viewType != ViewTypeArangoSearch {
		err := shared.ArangoError{
			HasError:     true,
			Code:         http.StatusConflict,
			ErrorNum:     0,
			ErrorMessage: fmt.Sprintf("Type must be '%s', got '%s'", ViewTypeArangoSearch, v.viewType),
		}
		return nil, errors.WithStack(err)
	}
	return &viewArangoSearch{view: v}, nil
}

func (v *view) ArangoSearchViewAlias() (ArangoSearchViewAlias, error) {
	if v.viewType != ViewTypeSearchAlias {
		err := shared.ArangoError{
			HasError:     true,
			Code:         http.StatusConflict,
			ErrorNum:     0,
			ErrorMessage: fmt.Sprintf("Type must be '%s', got '%s'", ViewTypeSearchAlias, v.viewType),
		}
		return nil, err
	}
	return &viewArangoSearchAlias{view: v}, nil
}

type RemoveViewOptions struct {
	// IsSystem when set to true allows to remove system views.
	// Use on your own risk!
	IsSystem *bool
}

func (o *RemoveViewOptions) modifyRequest(r connection.Request) error {
	if o == nil {
		return nil
	}
	if o.IsSystem != nil {
		r.AddQuery("isSystem", boolToString(*o.IsSystem))
	}
	return nil
}
