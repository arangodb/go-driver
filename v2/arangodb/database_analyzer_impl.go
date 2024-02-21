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

func newDatabaseAnalyzer(db *database) *databaseAnalyzer {
	return &databaseAnalyzer{
		db: db,
	}
}

var _ DatabaseAnalyzer = &databaseAnalyzer{}

type databaseAnalyzer struct {
	db *database
}

func (d databaseAnalyzer) EnsureAnalyzer(ctx context.Context, analyzer *AnalyzerDefinition) (bool, Analyzer, error) {
	urlEndpoint := d.db.url("_api", "analyzer")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		AnalyzerDefinition
	}
	resp, err := connection.CallPost(ctx, d.db.connection(), urlEndpoint, &response, analyzer)
	if err != nil {
		return false, nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated, http.StatusOK:
		return code == http.StatusOK, newAnalyzer(d.db, response.AnalyzerDefinition), nil
	default:
		return false, nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseAnalyzer) Analyzer(ctx context.Context, name string) (Analyzer, error) {
	urlEndpoint := d.db.url("_api", "analyzer", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
		AnalyzerDefinition
	}
	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newAnalyzer(d.db, response.AnalyzerDefinition), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

// Analyzers lists returns a list of all analyzers
func (d databaseAnalyzer) Analyzers(ctx context.Context) (AnalyzersResponseReader, error) {
	urlEndpoint := d.db.url("_api", "analyzer")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Analyzers             connection.Array `json:"result,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newAnalyzersResponseReader(d.db, &response.Analyzers), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func newAnalyzersResponseReader(db *database, arr *connection.Array) AnalyzersResponseReader {
	return &analyzerResponseReader{
		array: arr,
		db:    db,
	}
}

type analyzerResponseReader struct {
	array *connection.Array
	db    *database
}

func (reader *analyzerResponseReader) Read() (Analyzer, error) {
	if !reader.array.More() {
		return nil, shared.NoMoreDocumentsError{}
	}

	analyzerResponse := AnalyzerDefinition{}

	if err := reader.array.Unmarshal(newUnmarshalInto(&analyzerResponse)); err != nil {
		if err == io.EOF {
			return nil, shared.NoMoreDocumentsError{}
		}
		return nil, err
	}

	return newAnalyzer(reader.db, analyzerResponse), nil
}
