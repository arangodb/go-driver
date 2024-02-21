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

func newDatabase(c *client, name string, modifiers ...connection.RequestModifier) *database {
	d := &database{client: c, name: name, modifiers: modifiers}

	d.databaseCollection = newDatabaseCollection(d)
	d.databaseTransaction = newDatabaseTransaction(d)
	d.databaseQuery = newDatabaseQuery(d)
	d.databaseView = newDatabaseView(d)
	d.databaseAnalyzer = newDatabaseAnalyzer(d)
	d.databaseGraph = newDatabaseGraph(d)

	return d
}

var _ Database = &database{}

type database struct {
	client    *client
	name      string
	modifiers []connection.RequestModifier

	*databaseCollection
	*databaseTransaction
	*databaseQuery
	*databaseView
	*databaseAnalyzer
	*databaseGraph
}

func (d database) Remove(ctx context.Context) error {
	urlEndpoint := connection.NewUrl("_api", "database", url.PathEscape(d.name))

	resp, err := connection.CallDelete(ctx, d.client.connection, urlEndpoint, nil)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (d database) connection() connection.Connection {
	return d.client.connection
}

func (d database) url(parts ...string) string {
	return connection.NewUrl(append([]string{"_db", d.name}, parts...)...)
}

func (d database) Name() string {
	return d.name
}

func (d database) Info(ctx context.Context) (DatabaseInfo, error) {
	urlEndpoint := d.url("_api", "database", "current")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Database              DatabaseInfo `json:"result"`
	}

	resp, err := connection.CallGet(ctx, d.client.connection, urlEndpoint, &response)
	if err != nil {
		return DatabaseInfo{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Database, nil
	default:
		return DatabaseInfo{}, response.AsArangoErrorWithCode(code)
	}
}

func (d database) TransactionJS(ctx context.Context, options TransactionJSOptions) (interface{}, error) {
	urlEndpoint := d.url("_api", "transaction")

	var transactionResponse struct {
		shared.ResponseStruct `json:",inline"`
		Result                interface{} `json:"result"`
	}

	resp, err := connection.CallPost(ctx, d.client.connection, urlEndpoint, &transactionResponse, &options)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return transactionResponse.Result, nil
	default:
		return nil, transactionResponse.AsArangoError()
	}
}
