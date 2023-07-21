//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
	"io"
	"net/http"

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

func (d databaseAnalyzer) Analyzer(ctx context.Context, name string) (Analyzer, error) {
	url := d.db.url("_api", "analyzer", name)

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newAnalyzer(d.db, name), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}

}

// Analyzers lists returns a list of all analyzers
func (d databaseAnalyzer) Analyzers(ctx context.Context) (AnalyzersResponseReader, error) {
	url := d.db.url("_api", "analyzer")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Analyzers             connection.Array `json:"result,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response)
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

	analyzerResponse := struct {
		Name       string          `json:"name"`
		Type       string          `json:"type,omitempty"`
		Properties json.RawMessage `json:"properties,omitempty"`
		Features   []string        `json:"features,omitempty"`
	}{}

	if err := reader.array.Unmarshal(newUnmarshalInto(analyzerResponse)); err != nil {
		if err == io.EOF {
			return nil, shared.NoMoreDocumentsError{}
		}
		return nil, err
	}

	return newAnalyzer(reader.db, analyzerResponse.Name), nil
}
