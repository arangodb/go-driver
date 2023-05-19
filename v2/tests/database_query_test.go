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

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// Test_ExplainQuery tries to explain several AQL queries.
func Test_ExplainQuery(t *testing.T) {
	rf := arangodb.ReplicationFactor(2)
	options := arangodb.CreateCollectionOptions{
		ReplicationFactor: rf,
		NumberOfShards:    2,
	}
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, &options, func(col arangodb.Collection) {
				// Setup tests
				tests := []struct {
					Query         string
					BindVars      map[string]interface{}
					Opts          *arangodb.ExplainQueryOptions
					ExpectSuccess bool
				}{
					{
						Query:         fmt.Sprintf("FOR d IN `%s` SORT d.Title RETURN d", col.Name()),
						ExpectSuccess: true,
					},
					{
						Query: fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						BindVars: map[string]interface{}{
							"title": "Defending the Undefendable",
						},
						ExpectSuccess: true,
					},
					{
						Query: fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						BindVars: map[string]interface{}{
							"title": "Democracy: God That Failed",
						},
						Opts: &arangodb.ExplainQueryOptions{
							AllPlans:  true,
							Optimizer: arangodb.ExplainQueryOptimizerOptions{},
						},
						ExpectSuccess: true,
					},
					{
						Query:         fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						ExpectSuccess: false, // bindVars not provided
					},
					{
						Query:         fmt.Sprintf("FOR u IN `%s` FILTER u.age>>>100 SORT u.name RETURN u", col.Name()),
						ExpectSuccess: false, // syntax error
					},
					{
						Query:         "",
						ExpectSuccess: false,
					},
				}
				for i, test := range tests {
					t.Run(fmt.Sprintf("Case %d", i), func(t *testing.T) {
						_, err := db.ExplainQuery(ctx, test.Query, test.BindVars, test.Opts)
						if test.ExpectSuccess {
							require.NoError(t, err, "case %d", i)
						} else {
							require.Error(t, err, "case %d", i)
						}
					})
				}
			})
		})
	})
}

func Test_QueryBatchWithRetries(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				WithUserDocs(t, col, func(docs []UserDoc) {
					withContextT(t, 2*time.Minute, func(ctx context.Context, tb testing.TB) {
						skipBelowVersion(client, ctx, "3.11", t)

						query := fmt.Sprintf("FOR d IN `%s` SORT d.Title RETURN d", col.Name())

						t.Run("Test retry if batch size equals 1", func(t *testing.T) {
							opts := arangodb.QueryOptions{
								Count:     true,
								BatchSize: 1,
								Options: arangodb.QuerySubOptions{
									AllowRetry: true,
								},
							}

							var result []UserDoc
							cursor, err := db.QueryBatch(ctx, query, &opts, &result)
							require.NoError(t, err)
							require.Len(t, result, 1)
							require.Equal(t, docs[0].Name, result[0].Name)

							for {
								if !cursor.HasMoreBatches() {
									break
								}
								require.NoError(t, cursor.ReadNextBatch(ctx, &result))
								require.Len(t, result, 1)

								var resultRetry []UserDoc
								require.NoError(t, cursor.RetryReadBatch(ctx, &resultRetry))
								require.Len(t, resultRetry, 1)

								require.Equal(t, result[0].Name, resultRetry[0].Name)
							}

							err = cursor.Close()
							require.NoError(t, err)
						})

						t.Run("Test retry if batch size equals more than 1", func(t *testing.T) {
							opts := arangodb.QueryOptions{
								Count:     true,
								BatchSize: 2,
								Options: arangodb.QuerySubOptions{
									AllowRetry: true,
								},
							}

							var result []UserDoc
							cursor, err := db.QueryBatch(ctx, query, &opts, &result)
							require.NoError(t, err)
							require.Len(t, result, 2)
							require.Equal(t, docs[0].Name, result[0].Name)

							for {
								if !cursor.HasMoreBatches() {
									break
								}
								require.NoError(t, cursor.ReadNextBatch(ctx, &result))

								var resultRetry []UserDoc
								require.NoError(t, cursor.RetryReadBatch(ctx, &resultRetry))

								if cursor.HasMoreBatches() {
									require.Len(t, result, 2)
									require.Len(t, resultRetry, 2)
								} else {
									require.Len(t, result, 1)
									require.Len(t, resultRetry, 1)
								}

								require.Equal(t, result[0].Name, resultRetry[0].Name)
							}

							err = cursor.Close()
							require.NoError(t, err)
						})

						t.Run("Test retry double retries to ensure that result is same in every try", func(t *testing.T) {
							opts := arangodb.QueryOptions{
								Count:     true,
								BatchSize: 2,
								Options: arangodb.QuerySubOptions{
									AllowRetry: true,
								},
							}

							var result []UserDoc
							cursor, err := db.QueryBatch(ctx, query, &opts, &result)
							require.NoError(t, err)
							require.Len(t, result, 2)
							require.Equal(t, docs[0].Name, result[0].Name)

							for {
								if !cursor.HasMoreBatches() {
									break
								}
								require.NoError(t, cursor.ReadNextBatch(ctx, &result))

								var resultRetry []UserDoc
								require.NoError(t, cursor.RetryReadBatch(ctx, &resultRetry))

								var resultRetry2 []UserDoc
								require.NoError(t, cursor.RetryReadBatch(ctx, &resultRetry2))

								if cursor.HasMoreBatches() {
									require.Len(t, result, 2)
									require.Len(t, resultRetry, 2)
								} else {
									require.Len(t, result, 1)
									require.Len(t, resultRetry, 1)
								}

								require.Equal(t, result[0].Name, resultRetry[0].Name)
								require.Equal(t, result[0].Name, resultRetry2[0].Name)
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
