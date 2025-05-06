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
	"fmt"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/stretchr/testify/require"
)

// Test_ExplainQuery tries to explain several AQL queries.
func Test_CursorRawResult(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				WithUserDocs(t, col, func(docs []UserDoc) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
						skipBelowVersion(client, ctx, "3.11", t)

						query := fmt.Sprintf("FOR d IN `%s` SORT d.Title RETURN d", col.Name())

						t.Run("Test retry read when reading raw JSON batch", func(t *testing.T) {
							opts := arangodb.QueryOptions{
								Count:     true,
								BatchSize: 2,
								Options: arangodb.QuerySubOptions{
									AllowRetry: true,
								},
							}

							cursor, err := db.QueryBatch(ctx, query, &opts, nil)
							require.NoError(t, err)
							for {
								if !cursor.HasMoreBatches() {
									break
								}
								var result connection.RawObject
								require.NoError(t, cursor.ReadNextRawBatch(ctx, &result))

								var resultRetry connection.RawObject
								require.NoError(t, cursor.RetryReadRawBatch(ctx, &resultRetry))

								require.Equal(t, len(result), len(resultRetry))
								require.Equal(t, result, resultRetry)

							}

							err = cursor.Close()
							require.NoError(t, err)
						})
					})
				})
			})
		})
	})
}
