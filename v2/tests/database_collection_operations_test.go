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

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

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
	options := arangodb.CreateCollectionPropertiesV2{
		ReplicationFactor: &rf,
		NumberOfShards:    utils.NewType(2),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, &options, func(col arangodb.Collection) {
				shards, err := col.Shards(context.Background(), true)
				require.NoError(t, err)

				assert.NotEmpty(t, shards.ID)
				assert.Equal(t, col.Name(), shards.Name)
				assert.NotEmpty(t, shards.Status)
				assert.Equal(t, arangodb.CollectionTypeDocument, shards.Type)
				assert.Equal(t, false, *shards.IsSystem)
				assert.NotEmpty(t, *shards.GloballyUniqueId)
				assert.Equal(t, false, *shards.CacheEnabled)
				assert.Equal(t, false, *shards.IsSmart)
				assert.Equal(t, arangodb.KeyGeneratorTraditional, shards.KeyOptions.Type)
				assert.Equal(t, true, *shards.KeyOptions.AllowUserKeys)
				assert.Equal(t, 2, *shards.NumberOfShards)
				assert.Equal(t, arangodb.ShardingStrategyHash, shards.ShardingStrategy)
				assert.Equal(t, []string{"_key"}, *shards.ShardKeys)
				require.Len(t, shards.Shards, 2, "expected 2 shards")
				var leaders []arangodb.ServerID
				for _, dbServers := range shards.Shards {
					require.Lenf(t, dbServers, 2, "expected 2 DB servers for the shard")
					leaders = append(leaders, dbServers[0])
				}
				assert.NotEqualf(t, leaders[0], leaders[1], "the leader shard can not be on the same server")
				assert.Equal(t, rf, *shards.ReplicationFactor)
				assert.Equal(t, false, *shards.WaitForSync)
				assert.Equal(t, 1, *shards.WriteConcern)
			})

			version, err := client.Version(context.Background())
			require.NoError(t, err)

			if version.IsEnterprise() {
				optionsSatellite := arangodb.CreateCollectionPropertiesV2{
					ReplicationFactor: utils.NewType(arangodb.ReplicationFactorSatellite),
				}
				WithCollectionV2(t, db, &optionsSatellite, func(col arangodb.Collection) {
					shards, err := col.Shards(context.Background(), true)
					require.NoError(t, err)
					assert.Equal(t, arangodb.ReplicationFactorSatellite, *shards.ReplicationFactor)
				})
			}
		})
	})
}

// Test_CollectionSetProperties tries to set properties to collection
func Test_CollectionSetProperties(t *testing.T) {
	createOpts := arangodb.CreateCollectionPropertiesV2{
		WaitForSync:       utils.NewType(false),
		ReplicationFactor: utils.NewType(arangodb.ReplicationFactor(2)),
		JournalSize:       utils.NewType(int64(1048576 * 2)),
		NumberOfShards:    utils.NewType(2),
		CacheEnabled:      utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, &createOpts, func(col arangodb.Collection) {
				ctx := context.Background()
				props, err := col.Properties(ctx)
				require.NoError(t, err)

				// Dereference both sides for comparison
				require.Equal(t, *createOpts.WaitForSync, *props.WaitForSync)

				t.Run("rf-check-before", func(t *testing.T) {
					requireClusterMode(t)
					require.Equal(t, *createOpts.ReplicationFactor, *props.ReplicationFactor)
					require.Equal(t, *createOpts.NumberOfShards, *props.NumberOfShards)
				})

				newProps := arangodb.SetCollectionPropertiesOptionsV2{
					WaitForSync:       utils.NewType(true),
					ReplicationFactor: utils.NewType(arangodb.ReplicationFactor(3)),
					WriteConcern:      utils.NewType(2),
					CacheEnabled:      utils.NewType(true),
					Schema:            nil,
				}

				err = col.SetPropertiesV2(ctx, newProps)
				require.NoError(t, err)

				props, err = col.Properties(ctx)
				require.NoError(t, err)

				// Dereference both sides
				require.Equal(t, *newProps.WaitForSync, *props.WaitForSync)
				require.Equal(t, int64(0), props.JournalSize) // Default JournalSize is 0
				require.Equal(t, *newProps.CacheEnabled, *props.CacheEnabled)

				t.Run("rf-check-after", func(t *testing.T) {
					requireClusterMode(t)
					require.Equal(t, *newProps.ReplicationFactor, *props.ReplicationFactor)
					require.Equal(t, *createOpts.NumberOfShards, *props.NumberOfShards)
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
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				t.Run("Cursor - optimizer rules", func(t *testing.T) {

					ctx, c := context.WithTimeout(context.Background(), 1*time.Minute)
					defer c()

					col, err := db.CreateCollectionWithOptionsV2(ctx, "test", nil, &arangodb.CreateCollectionOptions{
						EnforceReplicationFactor: utils.NewType(false),
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
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
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
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
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

func assertCollectionFigures(t *testing.T, col arangodb.Collection, stats arangodb.CollectionStatistics) {
	assert.NotEmpty(t, stats.ID)
	assert.Equal(t, col.Name(), stats.Name)
	assert.NotEmpty(t, stats.Status)
	assert.Equal(t, arangodb.CollectionTypeDocument, stats.Type)

	// Safe nil checks before dereferencing
	if stats.IsSystem != nil {
		assert.Equal(t, false, *stats.IsSystem)
	} else {
		t.Log("IsSystem field is nil, skipping assertion")
	}

	if stats.GloballyUniqueId != nil {
		assert.NotEmpty(t, *stats.GloballyUniqueId)
	} else {
		t.Log("GloballyUniqueId field is nil, skipping assertion")
	}

	assert.NotEmpty(t, stats.Figures)
}

func Test_CollectionStatistics(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					docs := []map[string]interface{}{
						{"_key": "doc1", "name": "Alice"},
						{"_key": "doc2", "name": "Bob"},
						{"_key": "doc3", "name": "Charlie"},
					}

					for _, doc := range docs {
						_, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
					}
					_, err := col.DeleteDocument(ctx, "doc2")
					require.NoError(t, err)

					stats, err := col.Statistics(ctx, true)
					require.NoError(t, err)
					assertCollectionFigures(t, col, stats)

					// Safe nil checks before dereferencing
					if stats.Figures.Alive.Count != nil {
						assert.GreaterOrEqual(t, *stats.Figures.Alive.Count, int64(0))
					}
					if stats.Figures.Dead.Count != nil {
						assert.GreaterOrEqual(t, *stats.Figures.Dead.Count, int64(0))
					}
					if stats.Figures.DataFiles.Count != nil {
						assert.GreaterOrEqual(t, *stats.Figures.DataFiles.Count, int64(0))
					}
					if stats.Figures.Journals.FileSize != nil {
						assert.GreaterOrEqual(t, *stats.Figures.Journals.FileSize, int64(0))
					}
					if stats.Figures.Revisions.Size != nil {
						assert.GreaterOrEqual(t, *stats.Figures.Revisions.Size, int64(0))
					}

					t.Run("Statistics with details=false", func(t *testing.T) {
						stats, err := col.Statistics(ctx, false)
						require.NoError(t, err)
						assertCollectionFigures(t, col, stats)
					})
				})
			})
		})
	})
}

func Test_CollectionRevision(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// Get initial revision
					initialRev, err := col.Revision(ctx)
					require.NoError(t, err)
					require.NotEmpty(t, initialRev.Revision)

					// Create documents
					docs := []map[string]interface{}{
						{"_key": "doc1", "name": "Alice"},
						{"_key": "doc2", "name": "Bob"},
						{"_key": "doc3", "name": "Charlie"},
					}
					for _, doc := range docs {
						_, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
					}

					// Delete a document
					_, err = col.DeleteDocument(ctx, "doc2")
					require.NoError(t, err)

					// Get final revision
					finalRev, err := col.Revision(ctx)
					require.NoError(t, err)

					// Ensure finalRev is not nil and can be marshaled
					require.NotEmpty(t, finalRev.Revision)
					// Ensure revision changed
					require.NotEqual(t, initialRev.Revision, finalRev.Revision)
				})
			})
		})
	})
}

func Test_CollectionChecksum(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					trueValue := true
					falseValue := false
					stats, err := col.Checksum(ctx, &falseValue, &falseValue)
					require.NoError(t, err)
					require.NotEmpty(t, stats.Revision)
					require.NotEmpty(t, stats.CollectionInfo.ID)
					require.Equal(t, col.Name(), stats.CollectionInfo.Name)
					require.NotEmpty(t, stats.CollectionInfo.Status)
					require.Equal(t, arangodb.CollectionTypeDocument, stats.CollectionInfo.Type)
					require.NotEmpty(t, stats.CollectionInfo.GloballyUniqueId)
					t.Run("Checksum with withRevisions=false and withData=true", func(t *testing.T) {
						stats, err := col.Checksum(ctx, &falseValue, &trueValue)
						require.NoError(t, err)
						require.NotEmpty(t, stats.Revision)
					})

					t.Run("Checksum with withRevisions=true and withData=true", func(t *testing.T) {
						stats, err := col.Checksum(ctx, &trueValue, &trueValue)
						require.NoError(t, err)
						require.NotEmpty(t, stats.Revision)
					})
				})
			})
		})
	})
}

func Test_CollectionResponsibleShard(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					role, err := client.ServerRole(ctx)
					require.NoError(t, err)

					if role != arangodb.ServerRoleCoordinator {
						t.Skipf("Skipping test: ResponsibleShard is only supported on Coordinator, got role %s", role)
					}

					// Create some documents first
					docs := []map[string]interface{}{
						{"_key": "doc1", "name": "Alice"},
						{"_key": "doc2", "name": "Bob"},
						{"_key": "doc3", "name": "Charlie"},
					}
					for _, doc := range docs {
						_, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
					}

					// Check ResponsibleShard for a document key (does not need to exist)
					stats, err := col.ResponsibleShard(ctx, map[string]interface{}{
						"_key": "doc10",
					})
					require.NoError(t, err)
					require.NotEmpty(t, stats, "Responsible shard for doc10 should not be empty")

					// Check ResponsibleShard for an existing document
					stats, err = col.ResponsibleShard(ctx, map[string]interface{}{
						"_key": "doc1",
					})
					require.NoError(t, err)
					require.NotEmpty(t, stats, "Responsible shard for doc1 should not be empty")

					// Check ResponsibleShard for a non-existing document
					stats, err = col.ResponsibleShard(ctx, map[string]interface{}{
						"_key": "non-existing-doc",
					})
					require.NoError(t, err)
					require.NotEmpty(t, stats, "Responsible shard for non-existing doc should not be empty")
				})
			})
		})
	})
}

func Test_CollectionLoadIndexesIntoMemory(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			graph := sampleGraphWithEdges(db)

			//Create the graph in the database
			createdGraph, err := db.CreateGraph(context.Background(), graph.Name, graph, nil)
			require.NoError(t, err)
			require.NotNil(t, createdGraph)

			// Now access the edge collection from the created graph
			col, err := db.GetCollection(context.Background(), graph.EdgeDefinitions[0].Collection, nil)
			require.NoError(t, err)

			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				// Load indexes into memory
				loaded, err := col.LoadIndexesIntoMemory(ctx)
				require.NoError(t, err)
				require.True(t, loaded, "Expected edge index to be loaded")
			})
		})
	})
}
func Test_CollectionRename(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			role, err := client.ServerRole(ctx)
			require.NoError(t, err)

			if role != arangodb.ServerRoleSingle {
				t.Skip("Rename collection is not supported in cluster mode")
			}

			WithDatabase(t, client, nil, func(db arangodb.Database) {
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					newName := "test-renamed-collection"
					info, err := col.Rename(ctx, arangodb.RenameCollectionRequest{
						Name: newName,
					})
					require.NoError(t, err)
					require.Equal(t, newName, info.Name)
				})
			})
		})
	})
}

func Test_CollectionRecalculateCount(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// Create documents
					docs := []map[string]interface{}{
						{"_key": "doc1", "name": "Alice"},
						{"_key": "doc2", "name": "Bob"},
						{"_key": "doc3", "name": "Charlie"},
					}
					for _, doc := range docs {
						_, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
					}
					colCount, err := col.Count(ctx)
					require.NoError(t, err)
					require.Equal(t, int64(len(docs)), colCount)
					role, err := client.ServerRole(ctx)
					require.NoError(t, err)

					if role == arangodb.ServerRoleSingle {
						result, colRecalCount, err := col.RecalculateCount(ctx)
						require.NoError(t, err)
						require.True(t, result, "Recalculate count should return true")
						require.Greater(t, *colRecalCount, int64(0), "Recalculated count should be greater than 0")
					} else {
						result, _, err := col.RecalculateCount(ctx)
						require.NoError(t, err)
						require.True(t, result, "Recalculate count should return true")
					}
				})
			})
		})
	})
}

func Test_CollectionCompact(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					role, err := client.ServerRole(ctx)
					require.NoError(t, err)

					if role == arangodb.ServerRoleSingle {
						// Create dummy data
						for i := 0; i < 10; i++ {
							_, err := col.CreateDocument(ctx, map[string]interface{}{
								"_key": fmt.Sprintf("key%d", i),
								"val":  i,
							})
							require.NoError(t, err)
						}

						result, err := col.Compact(ctx)
						require.NoError(t, err)
						fmt.Printf("Compacted Collection: %s, ID: %s, Status: %d\n", result.Name, *result.ID, result.Status)
					} else {
						t.Skip("Compaction is not supported in cluster mode")
					}
				})
			})
		})
	})
}
