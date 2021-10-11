//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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

	options := arangodb.CreateCollectionOptions{
		ReplicationFactor: 2,
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
				assert.Equal(t, 2, shards.ReplicationFactor)
				assert.Equal(t, false, shards.WaitForSync)
				assert.Equal(t, 1, shards.WriteConcern)
			})
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
					println(query)

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
					println(query)

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

// TestLoadUnloadCollection unloads and loads the collection checking the appropriate statue.
func TestLoadUnloadCollection(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContext(30*time.Second, func(ctx context.Context) error {
			WithDatabase(t, client, nil, func(db arangodb.Database) {
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					status, err := col.Status(ctx)
					require.NoErrorf(t, err, "failed to get status of the collection")
					require.Equal(t, arangodb.CollectionStatusLoaded, status)

					err = col.Unload(ctx)
					require.NoErrorf(t, err, "failed to unload the collection")
					err = waitForCollectionStatus(ctx, col, arangodb.CollectionStatusUnloaded)
					require.NoErrorf(t, err, "the collection should be unloaded")

					err = col.Load(ctx)
					require.NoErrorf(t, err, "failed to load the collection")
					err = waitForCollectionStatus(ctx, col, arangodb.CollectionStatusLoaded)
					require.NoErrorf(t, err, "the collection should be loaded")
				})

				require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
			})

			return nil
		})
	})
}

// TestCollectionTruncate creates a collection, adds some documents and truncates it.
func TestCollectionTruncate(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContext(30*time.Minute, func(ctx context.Context) error {
			WithDatabase(t, client, nil, func(db arangodb.Database) {
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					var docs []Book
					numberDocs := 10
					for i := 0; i < numberDocs; i++ {
						docs = append(docs, Book{Title: fmt.Sprintf("Book %d", i)})
					}

					_, err := col.CreateDocumentsWithOptions(ctx, docs, nil)
					require.NoErrorf(t, err, "failed to create documents")

					count, err := col.Count(ctx)
					require.NoErrorf(t, err, "failed to get count of the documents")
					require.Equal(t, int64(numberDocs), count)

					err = col.Truncate(ctx)
					require.NoErrorf(t, err, "failed to truncate the collection")

					count, err = col.Count(ctx)
					require.NoErrorf(t, err, "failed to get count of the documents")
					require.Equal(t, int64(0), count)
				})

				require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
			})

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

// waitForCollectionStatus wait for the expected status of the collection.
func waitForCollectionStatus(ctx context.Context, col arangodb.Collection, status arangodb.CollectionStatus) error {
	for {
		if currentStatus, err := col.Status(ctx); err != nil {
			return err
		} else if currentStatus == status {
			return nil
		}

		time.Sleep(time.Millisecond * 10)
	}
}
