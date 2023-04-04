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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

// Test_CollectionShards creates a collection and gets the shards' information.
func Test_CollectionShards(t *testing.T) {
	requireClusterMode(t)

	rf := arangodb.ReplicationFactor(2)
	options := arangodb.CreateCollectionOptions{
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
				optionsSatellite := arangodb.CreateCollectionOptions{
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

func Test_DatabaseCollectionOperations(t *testing.T) {

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				ctx, c := context.WithTimeout(context.Background(), 5*time.Minute)
				defer c()

				size := 512

				docs := newDocs(size)

				for i := 0; i < size; i++ {
					docs[i].Fields = uuid.New().String()
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
						newDocs[i].Fields = uuid.New().String()
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
}

func TestDatabaseNameUnicode(t *testing.T) {
	databaseExtendedNamesRequired(t)

	Wrap(t, func(t *testing.T, c arangodb.Client) {
		withContext(30*time.Second, func(ctx context.Context) error {
			random := uuid.New().String()
			dbName := "\u006E\u0303\u00f1" + random
			_, err := c.CreateDatabase(ctx, dbName, nil)
			require.EqualError(t, err, "database name is not properly UTF-8 NFC-normalized")

			normalized := norm.NFC.String(dbName)
			_, err = c.CreateDatabase(ctx, normalized, nil)
			require.NoError(t, err)

			// The database should not be found by the not normalized name.
			_, err = c.Database(ctx, dbName)
			require.NotNil(t, err)

			// The database should be found by the normalized name.
			exist, err := c.DatabaseExists(ctx, normalized)
			require.NoError(t, err)
			require.True(t, exist)

			var found bool
			databases, err := c.Databases(ctx)
			require.NoError(t, err)
			for _, database := range databases {
				if database.Name() == normalized {
					found = true
					break
				}
			}
			require.Truef(t, found, "the database %s should have been found", normalized)

			// The database should return handler to the database by the normalized name.
			db, err := c.Database(ctx, normalized)
			require.NoError(t, err)
			require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")

			return nil
		})
	})
}

// databaseExtendedNamesRequired skips test if the version is < 3.9.0 or the ArangoDB has not been launched
// with the option --database.extended-names-databases=true.
func databaseExtendedNamesRequired(t *testing.T) {
	c := newClient(t, connectionJsonHttp(t))

	ctx := context.Background()
	version, err := c.Version(ctx)
	require.NoError(t, err)

	if version.Version.CompareTo("3.9.0") < 0 {
		t.Skipf("Version of the ArangoDB should be at least 3.9.0")
	}

	// If the database can be created with the below name then it means that it excepts unicode names.
	dbName := "\u006E\u0303\u00f1"
	normalized := norm.NFC.String(dbName)
	db, err := c.CreateDatabase(ctx, normalized, nil)
	if err == nil {
		require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
	}

	if shared.IsArangoErrorWithErrorNum(err, shared.ErrArangoDatabaseNameInvalid) {
		t.Skipf("ArangoDB is not launched with the option --database.extended-names-databases=true")
	}

	// Some other error which has not been expected.
	require.NoError(t, err)
}

func Test_DatabaseCollectionDelete(t *testing.T) {

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				ctx, c := context.WithTimeout(context.Background(), 5*time.Minute)
				defer c()

				size := 10
				docs := newDocs(size)
				for i := 0; i < size; i++ {
					docs[i].Fields = uuid.New().String()
				}

				docsIds := docs.asBasic().getKeys()
				docsMeta, err := col.CreateDocuments(ctx, docs)
				fmt.Print(docsMeta)
				require.NoError(t, err)

				t.Run("Delete single doc", func(t *testing.T) {
					key := docsIds[0]

					var doc document
					meta, err := col.ReadDocument(ctx, key, &doc)
					require.NoError(t, err)

					require.Equal(t, docs[0].Key, meta.Key)
					require.Equal(t, docs[0], doc)

					metaDel, err := col.DeleteDocument(ctx, key)
					require.NoError(t, err)
					require.Equal(t, docs[0].Key, metaDel.Key)

					_, err = col.DeleteDocument(ctx, key)
					require.Error(t, err)
				})

				t.Run("Delete single doc with options (silent)", func(t *testing.T) {
					key := docsIds[1]

					var doc document
					meta, err := col.ReadDocument(ctx, key, &doc)
					require.NoError(t, err)

					require.Equal(t, docs[1].Key, meta.Key)
					require.Equal(t, docs[1], doc)

					opts := arangodb.CollectionDocumentDeleteOptions{
						Silent: newBool(true),
					}
					resp, err := col.DeleteDocumentWithOptions(ctx, key, &opts)
					require.NoError(t, err)
					require.Equal(t, "", resp.Key, "response should be empty (silent)!")

					_, err = col.DeleteDocumentWithOptions(ctx, key, &opts)
					require.Error(t, err)
				})

				t.Run("Delete single doc with options (old)", func(t *testing.T) {
					key := docsIds[2]

					var doc document
					meta, err := col.ReadDocument(ctx, key, &doc)
					require.NoError(t, err)

					require.Equal(t, docs[2].Key, meta.Key)
					require.Equal(t, docs[2], doc)

					opts := arangodb.CollectionDocumentDeleteOptions{
						ReturnOld: newBool(true),
					}
					resp, err := col.DeleteDocumentWithOptions(ctx, key, &opts)
					require.NoError(t, err)
					require.NotEmpty(t, resp.Old)

					_, err = col.DeleteDocumentWithOptions(ctx, key, &opts)
					require.Error(t, err)
				})

				t.Run("Delete multiple docs", func(t *testing.T) {
					keys := []string{docsIds[3], docsIds[4], docsIds[5]}

					r, err := col.DeleteDocuments(ctx, keys)
					require.NoError(t, err)

					for i := 0; ; i++ {
						var doc document

						meta, err := r.Read(&doc)
						if shared.IsNoMoreDocuments(err) {
							break
						}
						require.NoError(t, err, meta)
						require.Equal(t, keys[i], meta.Key)
					}
				})

				t.Run("Delete multiple docs with error", func(t *testing.T) {
					alreadyRemovedDoc := docsIds[4]
					keys := []string{docsIds[6], docsIds[7], alreadyRemovedDoc}

					r, err := col.DeleteDocuments(ctx, keys)
					require.NoError(t, err)

					for i := 0; ; i++ {
						var doc document

						meta, err := r.Read(&doc)
						if shared.IsNoMoreDocuments(err) {
							break
						}
						require.NoError(t, err, meta)

						if keys[i] == alreadyRemovedDoc {
							require.NotNil(t, meta.Error)
							require.True(t, *meta.Error)
							require.True(t, shared.IsNotFound(meta.AsArangoError()))
							require.Empty(t, meta.Key)
						} else {
							require.Equal(t, keys[i], meta.Key)
						}
					}
				})

				t.Run("Delete multiple docs with options (old)", func(t *testing.T) {
					keys := []string{docsIds[8], docsIds[9]}

					opts := arangodb.CollectionDocumentDeleteOptions{
						ReturnOld: newBool(true),
					}
					r, err := col.DeleteDocumentsWithOptions(ctx, keys, &opts)
					require.NoError(t, err)

					for i := 0; ; i++ {
						var doc document

						meta, err := r.Read(&doc)
						if shared.IsNoMoreDocuments(err) {
							break
						}
						require.NoError(t, err, meta)
						require.Equal(t, keys[i], meta.Key)
						require.NotEmpty(t, meta.Old)
					}
				})
			})
		})
	})
}
