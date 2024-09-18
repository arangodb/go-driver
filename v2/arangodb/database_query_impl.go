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
	"encoding/json"
	"net/http"

	"github.com/arangodb/go-driver/v2/arangodb/shared"

	"github.com/arangodb/go-driver/v2/connection"
)

func newDatabaseQuery(db *database) *databaseQuery {
	return &databaseQuery{
		db: db,
	}
}

var _ DatabaseQuery = &databaseQuery{}

type databaseQuery struct {
	db *database
}

func (d databaseQuery) Query(ctx context.Context, query string, opts *QueryOptions) (Cursor, error) {
	return d.getCursor(ctx, query, opts, nil)
}

func (d databaseQuery) getCursor(ctx context.Context, query string, opts *QueryOptions, result interface{}) (*cursor, error) {
	url := d.db.url("_api", "cursor")

	req := struct {
		*QueryOptions
		*QueryRequest
	}{
		QueryOptions: opts,
		QueryRequest: &QueryRequest{Query: query},
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
		cursorData            `json:",inline"`
	}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &response, &req, append(d.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		if result != nil {
			if err := json.Unmarshal(response.cursorData.Result.in, result); err != nil {
				return nil, err
			}
		}
		return newCursor(d.db, resp.Endpoint(), response.cursorData), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) QueryBatch(ctx context.Context, query string, opts *QueryOptions, result interface{}) (CursorBatch, error) {
	return d.getCursor(ctx, query, opts, result)
}

func (d databaseQuery) ValidateQuery(ctx context.Context, query string) error {
	url := d.db.url("_api", "query")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	queryStruct := QueryRequest{Query: query}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &response, &queryStruct, d.db.modifiers...)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) ExplainQuery(ctx context.Context, query string, bindVars map[string]interface{}, opts *ExplainQueryOptions) (ExplainQueryResult, error) {
	url := d.db.url("_api", "explain")

	var request = struct {
		Query    string                 `json:"query"`
		BindVars map[string]interface{} `json:"bindVars,omitempty"`
		Opts     *ExplainQueryOptions   `json:"options,omitempty"`
	}{
		Query:    query,
		BindVars: bindVars,
		Opts:     opts,
	}
	var response struct {
		shared.ResponseStruct `json:",inline"`
		ExplainQueryResult
	}
	resp, err := connection.CallPost(ctx, d.db.connection(), url, &response, &request, d.db.modifiers...)
	if err != nil {
		return ExplainQueryResult{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ExplainQueryResult, nil
	default:
		return ExplainQueryResult{}, response.AsArangoErrorWithCode(code)
	}
}
