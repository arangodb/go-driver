//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb/shared"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

type DocWithCode struct {
	Key  string `json:"_key,omitempty"`
	Code string `json:"code"`
}

func Test_DatabaseCollectionDocCreateCode(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithCode{
						Key: "test",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)
					require.NotEmpty(t, meta.Rev)
					require.Empty(t, meta.Old)
					require.Empty(t, meta.New)

					rdoc, err := col.ReadDocument(ctx, "test", &doc)
					require.NoError(t, err)

					require.EqualValues(t, "test", rdoc.Key)
				})
			})
		})

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithCode{
						Key: "test",
					}
					doc2 := DocWithCode{
						Key: "test2",
					}

					_, err := col.CreateDocuments(ctx, []any{
						doc, doc2,
					})
					require.NoError(t, err)

					docs, err := col.ReadDocuments(ctx, []string{
						"test",
						"tz44",
						"test2",
					})
					require.NoError(t, err)

					var z DocWithCode

					meta, err := docs.Read(&z)
					require.NoError(t, err)
					require.EqualValues(t, "test", meta.Key)

					_, err = docs.Read(&z)
					require.Error(t, err)
					require.True(t, shared.IsNotFound(err))

					meta, err = docs.Read(&z)
					require.NoError(t, err)
					require.EqualValues(t, "test2", meta.Key)

					_, err = docs.Read(&z)
					require.Error(t, err)
					require.True(t, shared.IsNoMoreDocuments(err))
				})
			})
		})
	})
}
