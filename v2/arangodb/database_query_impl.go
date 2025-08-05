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
	"fmt"
	"net/http"
	"path"

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

func (d databaseQuery) GetQueryProperties(ctx context.Context) (QueryProperties, error) {
	url := d.db.url("_api", "query", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		QueryProperties       `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return QueryProperties{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.QueryProperties, nil
	default:
		return QueryProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) UpdateQueryProperties(ctx context.Context, options QueryProperties) (QueryProperties, error) {
	url := d.db.url("_api", "query", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		QueryProperties       `json:",inline"`
	}
	resp, err := connection.CallPut(ctx, d.db.connection(), url, &response, options, d.db.modifiers...)
	if err != nil {
		return QueryProperties{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.QueryProperties, nil
	default:
		return QueryProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) listAQLQueries(ctx context.Context, endpoint string, all *bool) ([]RunningAQLQuery, error) {
	url := d.db.url("_api", "query", endpoint)
	if all != nil && *all {
		url += "?all=true"
	}

	var response []RunningAQLQuery
	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, fmt.Errorf("API returned status %d", code)
	}
}
func (d databaseQuery) ListOfRunningAQLQueries(ctx context.Context, all *bool) ([]RunningAQLQuery, error) {
	return d.listAQLQueries(ctx, "current", all)
}

func (d databaseQuery) ListOfSlowAQLQueries(ctx context.Context, all *bool) ([]RunningAQLQuery, error) {
	return d.listAQLQueries(ctx, "slow", all)
}

func (d databaseQuery) deleteQueryEndpoint(ctx context.Context, path string, all *bool) error {
	url := d.db.url(path)

	if all != nil && *all {
		url += "?all=true"
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, d.db.connection(), url, &response, d.db.modifiers...)
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

func (d databaseQuery) ClearSlowAQLQueries(ctx context.Context, all *bool) error {
	return d.deleteQueryEndpoint(ctx, "_api/query/slow", all)
}

func (d databaseQuery) KillAQLQuery(ctx context.Context, queryId string, all *bool) error {
	return d.deleteQueryEndpoint(ctx, path.Join("_api/query", queryId), all)
}

func (d databaseQuery) GetAllOptimizerRules(ctx context.Context) ([]OptimizerRules, error) {
	url := d.db.url("_api", "query", "rules")

	var response []OptimizerRules

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, fmt.Errorf("API returned status %d", code)
	}
}

func (d databaseQuery) GetQueryPlanCache(ctx context.Context) ([]QueryPlanCacheRespObject, error) {
	url := d.db.url("_api", "query-plan-cache")

	var response []QueryPlanCacheRespObject

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, fmt.Errorf("API returned status %d", code)
	}
}

func (d databaseQuery) ClearQueryPlanCache(ctx context.Context) error {
	url := d.db.url("_api", "query-plan-cache")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, d.db.connection(), url, &response, d.db.modifiers...)
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

func (d databaseQuery) GetQueryEntriesCache(ctx context.Context) ([]QueryCacheEntriesRespObject, error) {
	url := d.db.url("_api", "query-cache", "entries")

	var response []QueryCacheEntriesRespObject

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, fmt.Errorf("API returned status %d", code)
	}
}

func (d databaseQuery) ClearQueryCache(ctx context.Context) error {
	url := d.db.url("_api", "query-cache")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, d.db.connection(), url, &response, d.db.modifiers...)
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

func (d databaseQuery) GetQueryCacheProperties(ctx context.Context) (QueryCacheProperties, error) {
	url := d.db.url("_api", "query-cache", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		QueryCacheProperties  `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return QueryCacheProperties{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.QueryCacheProperties, nil
	default:
		return QueryCacheProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) SetQueryCacheProperties(ctx context.Context, options QueryCacheProperties) (QueryCacheProperties, error) {
	url := d.db.url("_api", "query-cache", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		QueryCacheProperties  `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, d.db.connection(), url, &response, options, d.db.modifiers...)
	if err != nil {
		return QueryCacheProperties{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.QueryCacheProperties, nil
	default:
		return QueryCacheProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseQuery) CreateUserDefinedFunction(ctx context.Context, options UserDefinedFunctionObject) (bool, error) {
	url := d.db.url("_api", "aqlfunction")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		IsNewlyCreated        bool `json:"isNewlyCreated,omitempty"`
	}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &response, options, d.db.modifiers...)
	if err != nil {
		return false, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusCreated:
		return response.IsNewlyCreated, nil
	default:
		return false, response.AsArangoErrorWithCode(code)
	}
}
