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
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newAnalyzer(db *database, def AnalyzerDefinition, modifiers ...connection.RequestModifier) *analyzer {
	d := &analyzer{db: db, definition: def, modifiers: append(db.modifiers, modifiers...)}

	return d
}

var _ Analyzer = &analyzer{}

type analyzer struct {
	db *database

	definition AnalyzerDefinition

	modifiers []connection.RequestModifier
}

func (a analyzer) Name() string {
	split := strings.Split(a.definition.Name, "::")
	return split[len(split)-1]
}

func (a analyzer) UniqueName() string {
	return a.definition.Name
}

func (a analyzer) Type() ArangoSearchAnalyzerType {
	return a.definition.Type
}

func (a analyzer) Definition() AnalyzerDefinition {
	return a.definition
}

func (a analyzer) Database() Database {
	return a.db
}

func (a analyzer) Remove(ctx context.Context, force bool) error {
	urlEndpoint := a.db.url("_api", "analyzer", url.PathEscape(a.Name()))
	var mods []connection.RequestModifier
	if force {
		mods = append(mods, connection.WithQuery("force", "true"))
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, a.db.connection(), urlEndpoint, &response, append(a.db.modifiers, mods...)...)
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
