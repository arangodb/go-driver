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

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCollection(t *testing.T) {
	Wrap(t, func(t *testing.T, c arangodb.Client) {
		WithDatabase(t, c, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				// The collection should not be found
				_, err := db.GetCollection(ctx, "wrong-name", nil)
				require.NotNil(t, err)

				// IsExist validation should be skipped
				_, err = db.GetCollection(ctx, "wrong-name", &arangodb.GetCollectionOptions{SkipExistCheck: true})
				require.Nil(t, err)
			})
		})
	})
}

// Test_CollectionShards creates a collection and gets the shards' information.
func Test_CollectionShards(t *testing.T) {
	requireClusterMode(t)

	rf := arangodb.ReplicationFactor(2)
	options := arangodb.CreateCollectionProperties{
		ReplicationFactor: rf,
		NumberOfShards:    2,
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, &options, func(col arangodb.Collection) {
				shards, err := col.Shards(context.Background(), true)
				require.NoError(t, err)

				assert.NotEmpty(t, shards.ID)
				assert.Equal(t, col.Name(), shards.Name)
				assert.NotEmpty(t, shards.Status)
				assert.Equal(t, arangodb.CollectionTypeDocument, shards.Type)
				assert.Equal(t, false, shards.IsSystem)
				assert.NotEmpty(t, shards.GloballyUniqueId)
				assert.Equal(t, false, shards.CacheEnabled)
				assert.Equal(t, false, shards.IsSmart)
				assert.Equal(t, arangodb.KeyGeneratorTraditional, shards.KeyOptions.Type)
				assert.Equal(t, true, shards.KeyOptions.AllowUserKeys)
				assert.Equal(t, 2, shards.NumberOfShards)
				assert.Equal(t, arangodb.ShardingStrategyHash, shards.ShardingStrategy)
				assert.Equal(t, []string{"_key"}, shards.ShardKeys)
				require.Len(t, shards.Shards, 2, "expected 2 shards")
				var leaders []arangodb.ServerID
				for _, dbServers := range shards.Shards {
					require.Lenf(t, dbServers, 2, "expected 2 DB servers for the shard")
					leaders = append(leaders, dbServers[0])
				}
				assert.NotEqualf(t, leaders[0], leaders[1], "the leader shard can not be on the same server")
				assert.Equal(t, rf, shards.ReplicationFactor)
				assert.Equal(t, false, shards.WaitForSync)
				assert.Equal(t, 1, shards.WriteConcern)
			})

			version, err := client.Version(context.Background())
			require.NoError(t, err)

			if version.IsEnterprise() {
				optionsSatellite := arangodb.CreateCollectionProperties{
					ReplicationFactor: arangodb.ReplicationFactorSatellite,
				}
				WithCollection(t, db, &optionsSatellite, func(col arangodb.Collection) {
					shards, err := col.Shards(context.Background(), true)
					require.NoError(t, err)
					assert.Equal(t, arangodb.ReplicationFactorSatellite, shards.ReplicationFactor)
				})
			}
		})
	})
}

// Test_CollectionSetProperties tries to set properties to collection
func Test_CollectionSetProperties(t *testing.T) {
	createOpts := arangodb.CreateCollectionProperties{
		WaitForSync:       false,
		ReplicationFactor: 2,
		JournalSize:       1048576 * 2,
		NumberOfShards:    2,
		CacheEnabled:      newBool(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, &createOpts, func(col arangodb.Collection) {
				ctx := context.Background()

				props, err := col.Properties(ctx)
				require.NoError(t, err)
				require.Equal(t, createOpts.WaitForSync, props.WaitForSync)
				require.Equal(t, *createOpts.CacheEnabled, props.CacheEnabled)

				t.Run("rf-check-before", func(t *testing.T) {
					requireClusterMode(t)
					require.Equal(t, createOpts.ReplicationFactor, props.ReplicationFactor)
					require.Equal(t, createOpts.NumberOfShards, props.NumberOfShards)
				})

				newProps := arangodb.SetCollectionPropertiesOptions{
					WaitForSync:       newBool(true),
					ReplicationFactor: 3,
					WriteConcern:      2,
					CacheEnabled:      newBool(true),
					Schema:            nil,
				}
				err = col.SetProperties(ctx, newProps)
				require.NoError(t, err)

				props, err = col.Properties(ctx)
				require.NoError(t, err)
				require.Equal(t, *newProps.WaitForSync, props.WaitForSync)
				require.Equal(t, newProps.JournalSize, props.JournalSize)
				require.Equal(t, *newProps.CacheEnabled, props.CacheEnabled)

				t.Run("rf-check-after", func(t *testing.T) {
					requireClusterMode(t)
					require.Equal(t, newProps.ReplicationFactor, props.ReplicationFactor)
					require.Equal(t, createOpts.NumberOfShards, props.NumberOfShards)
				})
			})
		})
	})
}

// Test_WithQueryOptimizerRules tests optimizer rules for query.
func Test_WithQueryOptimizerRules(t *testing.T) {
	tests := map[string]struct {
		OptimizerRules   []string
		ExpectedRules    []string
		NotExpectedRules []string
	}{
		"include optimizer rule: use-indexes": {
			OptimizerRules: []string{"+use-indexes"},
			ExpectedRules:  []string{"use-indexes"},
		},
		"exclude optimizer rule: use-indexes": {
			OptimizerRules:   []string{"-use-indexes"},
			NotExpectedRules: []string{"use-indexes"},
		},
		"overwrite excluded optimizer rule: use-indexes": {
			OptimizerRules: []string{"-use-indexes", "+use-indexes"},
			ExpectedRules:  []string{"use-indexes"},
		},
		"overwrite included optimizer rule: use-indexes": {
			OptimizerRules:   []string{"+use-indexes", "-use-indexes"},
			NotExpectedRules: []string{"use-indexes"},
		},
		"turn off all optimizer rule": {
			OptimizerRules:   []string{"-all"},        // some rules can not be disabled.
			NotExpectedRules: []string{"use-indexes"}, // this rule will be disabled with all disabled rules.
		},
		"turn on all optimizer rule": {
			OptimizerRules: []string{"+all"},
			ExpectedRules:  []string{"use-indexes"}, // this rule will be enabled with all enabled rules.
		},
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				t.Run("Cursor - optimizer rules", func(t *testing.T) {

					ctx, c := context.WithTimeout(context.Background(), 1*time.Minute)
					defer c()

					col, err := db.CreateCollectionWithOptions(ctx, "test", nil, &arangodb.CreateCollectionOptions{
						EnforceReplicationFactor: newBool(false),
					})
					require.NoError(t, err)

					fieldName := "value"
					_, _, err = col.EnsurePersistentIndex(ctx, []string{fieldName}, &arangodb.CreatePersistentIndexOptions{
						Name: "index",
					})
					require.NoErrorf(t, err, "failed to index for collection \"%s\"", col.Name())

					type testDoc struct {
						Value int `json:"value"` // variable fieldName
					}
					err = arangodb.CreateDocuments(ctx, col, 100, func(index int) any {
						return testDoc{Value: index}
					})
					require.NoErrorf(t, err, "failed to create exemplary documents")

					query := fmt.Sprintf("FOR i IN %s FILTER i.%s > 97 SORT i.%s RETURN i.%s", col.Name(), fieldName,
						fieldName, fieldName)
					for testName, test := range tests {
						t.Run(testName, func(t *testing.T) {
							opts := &arangodb.QueryOptions{
								Options: arangodb.QuerySubOptions{
									Profile: 2,
									Optimizer: arangodb.QuerySubOptionsOptimizer{
										Rules: test.OptimizerRules,
									},
								},
							}
							q, err := db.Query(ctx, query, opts)
							require.NoError(t, err)

							plan := q.Plan()
							for _, rule := range test.ExpectedRules {
								require.Contains(t, plan.Rules, rule)
							}

							for _, rule := range test.NotExpectedRules {
								require.NotContains(t, plan.Rules, rule)
							}
						})
					}
				})
			})
		})
	})
}

func Test_DatabaseCollectionOperations(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					size := 512

					docs := newDocs(size)

					for i := 0; i < size; i++ {
						docs[i].Fields = GenerateUUID("test-doc-col")
					}

					docsIds := docs.asBasic().getKeys()

					t.Run("Create", func(t *testing.T) {
						_, err := col.CreateDocuments(ctx, docs)
						require.NoError(t, err)

						r, err := col.ReadDocuments(ctx, docsIds)

						nd := docs

						for {
							var doc document

							meta, err := r.Read(&doc)
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(t, err)

							require.True(t, len(nd) > 0)

							require.Equal(t, nd[0].Key, meta.Key)
							require.Equal(t, nd[0], doc)

							if len(nd) == 1 {
								nd = nil
							} else {
								nd = nd[1:]
							}
						}

						require.Len(t, nd, 0)
					})

					t.Run("Cursor - single", func(t *testing.T) {
						nd := docs

						query := fmt.Sprintf("FOR doc IN `%s` RETURN doc", col.Name())

						q, err := db.Query(ctx, query, &arangodb.QueryOptions{
							BatchSize: size,
						})
						require.NoError(t, err)

						for {
							var doc document
							meta, err := q.ReadDocument(ctx, &doc)
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(t, err)

							require.True(t, len(nd) > 0)

							require.Equal(t, nd[0].Key, meta.Key)
							require.Equal(t, nd[0], doc)

							if len(nd) == 1 {
								nd = nil
							} else {
								nd = nd[1:]
							}
						}
					})

					t.Run("Cursor - batches", func(t *testing.T) {
						nd := docs

						query := fmt.Sprintf("FOR doc IN `%s` RETURN doc", col.Name())

						q, err := db.Query(ctx, query, &arangodb.QueryOptions{
							BatchSize: size / 10,
						})
						require.NoError(t, err)

						for {
							var doc document
							meta, err := q.ReadDocument(ctx, &doc)
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(t, err)

							require.True(t, len(nd) > 0)

							require.Equal(t, nd[0].Key, meta.Key)
							require.Equal(t, nd[0], doc)

							if len(nd) == 1 {
								nd = nil
							} else {
								nd = nd[1:]
							}
						}
					})

					t.Run("Cursor - shardIds", func(t *testing.T) {
						requireClusterMode(t)

						query := fmt.Sprintf("FOR doc IN `%s` RETURN doc", col.Name())

						q, err := db.Query(ctx, query, &arangodb.QueryOptions{})
						require.NoError(t, err)
						i := 0
						for {
							var doc document
							_, err := q.ReadDocument(ctx, &doc)
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(t, err)
							i++
						}

						// Non existing shard should error
						q, err = db.Query(ctx, query, &arangodb.QueryOptions{
							Options: arangodb.QuerySubOptions{
								ShardIds: []string{"ss1"},
							},
						})
						require.NotNil(t, err)

						// collect all docs from all shards
						s, err := col.Shards(context.Background(), true)
						j := 0
						for sk := range s.Shards {
							shardIds := []string{string(sk)}
							q, err = db.Query(ctx, query, &arangodb.QueryOptions{
								Options: arangodb.QuerySubOptions{
									ShardIds: shardIds,
								},
							})
							require.NoError(t, err)

							for {
								var doc document
								_, err := q.ReadDocument(ctx, &doc)
								if shared.IsNoMoreDocuments(err) {
									break
								}
								require.NoError(t, err)
								j++
							}
						}
						require.Equal(t, i, j)

					})

					t.Run("Cursor - close", func(t *testing.T) {
						query := fmt.Sprintf("FOR doc IN `%s` RETURN doc", col.Name())

						q, err := db.Query(ctx, query, nil)
						require.NoError(t, err)

						require.NoError(t, q.CloseWithContext(ctx))

						var doc document
						_, err = q.ReadDocument(ctx, &doc)
						require.True(t, shared.IsNoMoreDocuments(err))
					})

					t.Run("Update", func(t *testing.T) {
						newDocs := make([]document, size)
						defer func() {
							docs = newDocs
						}()

						for i := 0; i < size; i++ {
							newDocs[i] = docs[i]
							newDocs[i].Fields = GenerateUUID("test-new-doc")
						}

						ng := newDocs
						nd := docs

						var old document
						var new document

						r, err := col.UpdateDocumentsWithOptions(ctx, newDocs, &arangodb.CollectionDocumentUpdateOptions{
							OldObject: &old,
							NewObject: &new,
						})
						require.NoError(t, err)

						for {

							meta, err := r.Read()
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(t, err)

							require.True(t, len(nd) > 0)

							require.Equal(t, nd[0].Key, meta.Key)
							require.Equal(t, ng[0].Key, meta.Key)
							require.Equal(t, ng[0], new)
							require.Equal(t, nd[0], old)

							if len(nd) == 1 {
								nd = nil
								ng = nil
							} else {
								nd = nd[1:]
								ng = ng[1:]
							}
						}

						require.Len(t, nd, 0)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionTruncate(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					size := 10
					docs := newDocs(size)
					for i := 0; i < size; i++ {
						docs[i].Fields = GenerateUUID("test-doc-truncate")
					}

					_, err := col.CreateDocuments(ctx, docs)
					require.NoError(t, err)

					beforeCount, err := col.Count(ctx)
					require.NoError(t, err)
					require.Equal(t, int64(size), beforeCount)

					err = col.Truncate(ctx)
					require.NoError(t, err)

					afterCount, err := col.Count(ctx)
					require.NoError(t, err)
					require.Equal(t, int64(0), afterCount)
				})
			})
		})
	})
}
