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
// Author Adam Janikowski
// Author Tomasz Mielech
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

				require.Len(t, shards.Shards, 2, "expected 2 shards")
				var leaders []arangodb.ServerID

				for _, dbServers := range shards.Shards {
					require.Lenf(t, dbServers, 2, "expected 2 DB servers for the shard")
					leaders = append(leaders, dbServers[0])
				}
				assert.NotEqualf(t, leaders[0], leaders[1], "the leader shard can not be on the same server")
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
