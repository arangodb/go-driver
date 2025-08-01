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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
)

// Test_ExplainQuery tries to explain several AQL queries.
func Test_ExplainQuery(t *testing.T) {
	rf := arangodb.ReplicationFactor(2)
	options := arangodb.CreateCollectionPropertiesV2{
		ReplicationFactor: &rf,
		NumberOfShards:    utils.NewType(2),
	}
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, &options, func(col arangodb.Collection) {
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
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				WithUserDocs(t, col, func(docs []UserDoc) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
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

func Test_GetQueryProperties(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			res, err := db.GetQueryProperties(context.Background())
			require.NoError(t, err)
			jsonResp, err := utils.ToJSONString(res)
			require.NoError(t, err)
			t.Logf("Query Properties: %s", jsonResp)
			// Check that the response contains expected fields
			require.NotNil(t, res)
			require.IsType(t, true, *res.Enabled)
			require.IsType(t, true, *res.TrackSlowQueries)
			require.IsType(t, true, *res.TrackBindVars)
			require.GreaterOrEqual(t, *res.MaxSlowQueries, 0)
			require.Greater(t, *res.SlowQueryThreshold, 0.0)
			require.Greater(t, *res.MaxQueryStringLength, 0)
		})
	})
}

func Test_UpdateQueryProperties(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			res, err := db.GetQueryProperties(context.Background())
			require.NoError(t, err)
			jsonResp, err := utils.ToJSONString(res)
			require.NoError(t, err)
			t.Logf("Query Properties: %s", jsonResp)
			// Check that the response contains expected fields
			require.NotNil(t, res)
			options := arangodb.QueryProperties{
				Enabled:              utils.NewType(true),
				TrackSlowQueries:     utils.NewType(true),
				TrackBindVars:        utils.NewType(false), // optional but useful for debugging
				MaxSlowQueries:       utils.NewType(*res.MaxSlowQueries + *utils.NewType(1)),
				SlowQueryThreshold:   utils.NewType(*res.SlowQueryThreshold + *utils.NewType(0.1)),
				MaxQueryStringLength: utils.NewType(*res.MaxQueryStringLength + *utils.NewType(100)),
			}
			updateResp, err := db.UpdateQueryProperties(context.Background(), options)
			require.NoError(t, err)
			jsonUpdateResp, err := utils.ToJSONString(updateResp)
			require.NoError(t, err)
			t.Logf("Updated Query Properties: %s", jsonUpdateResp)
			// Check that the response contains expected fields
			require.NotNil(t, updateResp)
			require.Equal(t, *options.Enabled, *updateResp.Enabled)
			require.Equal(t, *options.TrackSlowQueries, *updateResp.TrackSlowQueries)
			require.Equal(t, *options.TrackBindVars, *updateResp.TrackBindVars)
			require.Equal(t, *options.MaxSlowQueries, *updateResp.MaxSlowQueries)
			require.Equal(t, *options.SlowQueryThreshold, *updateResp.SlowQueryThreshold)
			require.Equal(t, *options.MaxQueryStringLength, *updateResp.MaxQueryStringLength)
			res, err = db.GetQueryProperties(context.Background())
			require.NoError(t, err)
			jsonResp, err = utils.ToJSONString(res)
			require.NoError(t, err)
			t.Logf("Query Properties 288: %s", jsonResp)
			// Check that the response contains expected fields
			require.NotNil(t, res)
		})
	})
}

func Test_ListOfRunningAQLQueries(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		db, err := client.GetDatabase(context.Background(), "_system", nil)
		require.NoError(t, err)
		// Test that the endpoint works (should return empty list or some queries)
		queries, err := db.ListOfRunningAQLQueries(context.Background(), utils.NewType(false))
		require.NoError(t, err)
		require.NotNil(t, queries)
		t.Logf("Current running queries (all=false): %d\n", len(queries))

		// Test with all=true parameter
		t.Run("Test with all=true parameter", func(t *testing.T) {
			allQueries, err := db.ListOfRunningAQLQueries(context.Background(), utils.NewType(true))
			require.NoError(t, err)
			require.NotNil(t, allQueries)
			t.Logf("Current running queries (all=true): %d\n", len(allQueries))

			// The number with all=true should be >= the number with all=false
			require.GreaterOrEqual(t, len(allQueries), len(queries),
				"all=true should return >= queries than all=false")
		})

		t.Run("Test that queries are not empty", func(t *testing.T) {

			// Create a context we can cancel
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start a transaction with a long-running query
			queryStarted := make(chan struct{})
			go func() {
				defer close(queryStarted)

				// Use a streaming query that processes results slowly
				bindVars := map[string]interface{}{
					"max": 10000000,
				}

				cursor, err := db.Query(ctx, `
	FOR i IN 1..@max
		LET computation = (
			FOR x IN 1..100
				RETURN x * i
		)
		RETURN {i: i, sum: SUM(computation)}
`, &arangodb.QueryOptions{
					BindVars: bindVars,
				})

				if err != nil {
					if !strings.Contains(err.Error(), "canceled") {
						t.Logf("Query error: %v", err)
					}
					return
				}

				// Process results slowly to keep query active longer
				if cursor != nil {
					for cursor.HasMore() {
						var result interface{}
						_, err := cursor.ReadDocument(ctx, &result)
						if err != nil {
							break
						}
						// Add small delay to keep query running longer
						time.Sleep(10 * time.Millisecond)
					}
					cursor.Close()
				}
			}()

			// Wait for query to start and be registered
			time.Sleep(2 * time.Second)

			// Check for running queries multiple times
			var foundRunningQuery bool
			for attempt := 0; attempt < 15; attempt++ {
				queries, err := db.ListOfRunningAQLQueries(context.Background(), utils.NewType(true))
				require.NoError(t, err)

				t.Logf("Attempt %d: Found %d queries", attempt+1, len(queries))

				if len(queries) > 0 {
					foundRunningQuery = true
					t.Logf("SUCCESS: Found %d running queries on attempt %d\n", len(queries), attempt+1)
					// Log query details
					for i, query := range queries {
						bindVarsJSON, _ := utils.ToJSONString(*query.BindVars)
						t.Logf("Query %d: ID=%s, State=%s, BindVars=%s",
							i, *query.Id, *query.State, bindVarsJSON)
					}
					break
				}

				time.Sleep(300 * time.Millisecond)
			}

			// Cancel the query
			cancel()

			// Assert we found running queries
			require.True(t, foundRunningQuery, "Should have found at least one running query")
		})
	})
}

func Test_ListOfSlowAQLQueries(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		// Get the database
		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		// Get the query properties
		res, err := db.GetQueryProperties(ctx)
		require.NoError(t, err)

		jsonResp, err := utils.ToJSONString(res)
		require.NoError(t, err)
		t.Logf("Query Properties: %s", jsonResp)
		// Check that the response contains expected fields
		require.NotNil(t, res)
		// Test that the endpoint works (should return empty list or some queries)
		queries, err := db.ListOfSlowAQLQueries(ctx, utils.NewType(false))
		require.NoError(t, err)
		require.NotNil(t, queries)
		t.Logf("Current running slow queries (all=false): %d\n", len(queries))

		// Test with all=true parameter
		t.Run("Test with all=true parameter", func(t *testing.T) {
			allQueries, err := db.ListOfSlowAQLQueries(ctx, utils.NewType(true))
			require.NoError(t, err)
			require.NotNil(t, allQueries)
			t.Logf("Current running slow queries (all=true): %d\n", len(allQueries))

			// The number with all=true should be >= the number with all=false
			require.GreaterOrEqual(t, len(allQueries), len(queries),
				"all=true should return >= queries than all=false")
		})
		// Update query properties to ensure slow queries are tracked
		t.Logf("Updating query properties to track slow queries")
		// Set a low threshold to ensure we capture slow queries
		// and limit the number of slow queries to 1 for testing
		options := arangodb.QueryProperties{
			Enabled:            utils.NewType(true),
			TrackSlowQueries:   utils.NewType(true),
			TrackBindVars:      utils.NewType(true), // optional but useful for debugging
			MaxSlowQueries:     utils.NewType(1),
			SlowQueryThreshold: utils.NewType(0.0001),
		}
		// Update the query properties
		_, err = db.UpdateQueryProperties(ctx, options)
		require.NoError(t, err)
		t.Run("Test that queries are not empty", func(t *testing.T) {

			_, err := db.Query(ctx, "FOR i IN 1..1000000 COLLECT WITH COUNT INTO length RETURN length", nil)
			require.NoError(t, err)

			// Wait for query to start and be registered
			time.Sleep(2 * time.Second)

			// Check for running queries multiple times
			var foundRunningQuery bool
			for attempt := 0; attempt < 15; attempt++ {
				queries, err := db.ListOfSlowAQLQueries(ctx, utils.NewType(true))
				require.NoError(t, err)

				t.Logf("Attempt %d: Found %d queries", attempt+1, len(queries))

				if len(queries) > 0 {
					foundRunningQuery = true
					t.Logf("SUCCESS: Found %d running queries on attempt %d\n", len(queries), attempt+1)
					// Log query details
					for i, query := range queries {
						t.Logf("Query %d: ID=%s, State=%s", i, *query.Id, *query.State)
					}
					break
				}

				time.Sleep(300 * time.Millisecond)
			}

			// Assert we found running queries
			require.True(t, foundRunningQuery, "Should have found at least one running query")
		})
	})
}

func Test_KillAQLQuery(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		// Get the database
		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		// Channel to signal when query has started
		// Create a context we can cancel
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start a transaction with a long-running query
		queryStarted := make(chan struct{})
		go func() {
			defer close(queryStarted)

			// Use a streaming query that processes results slowly
			bindVars := map[string]interface{}{
				"max": 10000000,
			}

			cursor, err := db.Query(ctx, `
	FOR i IN 1..@max
		LET computation = (
			FOR x IN 1..100
				RETURN x * i
		)
		RETURN {i: i, sum: SUM(computation)}
`, &arangodb.QueryOptions{
				BindVars: bindVars,
			})

			if err != nil {
				if !strings.Contains(err.Error(), "canceled") {
					t.Logf("Query error: %v", err)
				}
				return
			}

			// Process results slowly to keep query active longer
			if cursor != nil {
				for cursor.HasMore() {
					var result interface{}
					_, err := cursor.ReadDocument(ctx, &result)
					if err != nil {
						break
					}
					// Add small delay to keep query running longer
					time.Sleep(10 * time.Millisecond)
				}
				cursor.Close()
			}
		}()

		// Wait for query to start and be registered
		time.Sleep(2 * time.Second)

		// Check for running queries multiple times
		var foundRunningQuery bool
		for attempt := 0; attempt < 15; attempt++ {
			queries, err := db.ListOfRunningAQLQueries(context.Background(), utils.NewType(true))
			require.NoError(t, err)

			t.Logf("Attempt %d: Found %d queries", attempt+1, len(queries))

			if len(queries) > 0 {
				foundRunningQuery = true
				t.Logf("SUCCESS: Found %d running queries on attempt %d\n", len(queries), attempt+1)
				// Log query details
				for i, query := range queries {
					bindVarsJSON, _ := utils.ToJSONString(*query.BindVars)
					t.Logf("Query %d: ID=%s, State=%s, BindVars=%s",
						i, *query.Id, *query.State, bindVarsJSON)
					// Kill the query
					err := db.KillAQLQuery(ctx, *query.Id, utils.NewType(true))
					require.NoError(t, err, "Failed to kill query %s", *query.Id)
					t.Logf("Killed query %s", *query.Id)
				}
				break
			}

			time.Sleep(300 * time.Millisecond)
		}

		// Cancel the query
		cancel()

		// Assert we found running queries
		require.True(t, foundRunningQuery, "Should have found at least one running query")
	})
}
