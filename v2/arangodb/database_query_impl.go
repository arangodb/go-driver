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
//
// Author Adam Janikowski
//

package arangodb

import (
	"context"
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

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &response, &req, d.db.modifiers...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return newCursor(d.db, resp.Endpoint(), response.cursorData), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
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
