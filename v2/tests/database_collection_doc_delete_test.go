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
	"testing"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_DatabaseCollectionDocDeleteSimple(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

					size := 10
					docs := newDocs(size)
					for i := 0; i < size; i++ {
						docs[i].Fields = GenerateUUID("test-doc")
					}

					docsIds := docs.asBasic().getKeys()
					_, err := col.CreateDocuments(ctx, docs)
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

					t.Run("Delete single doc with options: old", func(t *testing.T) {
						key := docsIds[2]

						var doc document
						var oldDoc document
						meta, err := col.ReadDocument(ctx, key, &doc)
						require.NoError(t, err)

						require.Equal(t, docs[2].Key, meta.Key)
						require.Equal(t, docs[2], doc)

						opts := arangodb.CollectionDocumentDeleteOptions{
							OldObject: &oldDoc,
						}
						resp, err := col.DeleteDocumentWithOptions(ctx, key, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, resp.Old)
						require.Equal(t, docs[2], oldDoc)

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
							if keys[i] == alreadyRemovedDoc {
								require.Error(t, err)
								require.True(t, shared.IsNotFound(err))
								require.Empty(t, meta.Key)
							} else {
								require.NoError(t, err)
								require.Equal(t, keys[i], meta.Key)
							}
						}
					})

					t.Run("Delete multiple docs with options: old", func(t *testing.T) {
						keys := []string{docsIds[8], docsIds[9]}
						var oldDoc document

						opts := arangodb.CollectionDocumentDeleteOptions{
							OldObject: &oldDoc,
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
							require.Equal(t, keys[i], oldDoc.Key)
						}
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocDeleteIfMatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-if-match",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not delete if rev doesn't match", func(t *testing.T) {
						metaError, err := col.DeleteDocumentWithOptions(ctx, meta.Key, &arangodb.CollectionDocumentDeleteOptions{
							IfMatch: "wrong-rev",
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do a delete if rev does match", func(t *testing.T) {
						metaDeleted, err := col.DeleteDocumentWithOptions(ctx, meta.Key, &arangodb.CollectionDocumentDeleteOptions{
							IfMatch: meta.Rev,
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaDeleted.Rev)
						require.Equal(t, metaDeleted.Rev, meta.Rev)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocDeleteIgnoreRevs(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := []DocWithRev{
						{
							Name: "test-IgnoreRevs",
						},
						{
							Name: "test-IgnoreRevs-2",
						},
					}

					r, err := col.CreateDocuments(ctx, doc)
					require.NoError(t, err)

					meta, err := r.Read()
					require.NoError(t, err)

					t.Run("do not delete if rev doesn't match", func(t *testing.T) {
						docToRemove := []DocWithRev{
							{
								Key: meta.Key,
								Rev: "wrong-rev",
							},
						}

						delReader, err := col.DeleteDocumentsWithOptions(ctx, docToRemove, &arangodb.CollectionDocumentDeleteOptions{
							IgnoreRevs: utils.NewT(false),
						})
						require.NoError(t, err)

						var docRemoved DocWithRev
						_, errDel := delReader.Read(&docRemoved)
						require.Error(t, errDel)
						require.Equal(t, errDel.Error(), "conflict, _rev values do not match")

						_, err = col.ReadDocument(ctx, meta.Key, &docRemoved)
						require.NoError(t, err)
					})

					t.Run("do a delete if rev match", func(t *testing.T) {
						docToRemove := []DocWithRev{
							{
								Key: meta.Key,
								Rev: meta.Rev,
							},
						}

						delReader, err := col.DeleteDocumentsWithOptions(ctx, docToRemove, &arangodb.CollectionDocumentDeleteOptions{
							IgnoreRevs: utils.NewT(false),
						})
						require.NoError(t, err)

						var docRemoved DocWithRev
						_, errDel := delReader.Read(&docRemoved)
						require.NoError(t, errDel)

						_, err = col.ReadDocument(ctx, meta.Key, &docRemoved)
						require.Error(t, err)
					})

					meta, err = r.Read()
					require.NoError(t, err)

					t.Run("do a delete if rev is missing", func(t *testing.T) {
						docToRemove := []DocWithRev{
							{
								Key: meta.Key,
							},
						}

						delReader, err := col.DeleteDocumentsWithOptions(ctx, docToRemove, &arangodb.CollectionDocumentDeleteOptions{
							IgnoreRevs: utils.NewT(false),
						})

						require.NoError(t, err)

						var docRemoved DocWithRev
						_, errDel := delReader.Read(&docRemoved)
						require.NoError(t, errDel)

						_, err = col.ReadDocument(ctx, meta.Key, &docRemoved)
						require.Error(t, err)
					})

					meta, err = r.Read()
					require.Error(t, err)
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocDeleteSilent(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.12", t)

					doc := DocWithRev{
						Name: "test-silent",
						Age:  utils.NewT(42),
					}
					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					metaDeleted, err := col.DeleteDocumentWithOptions(ctx, meta.Key, &arangodb.CollectionDocumentDeleteOptions{
						Silent: utils.NewT(true),
					})
					require.NoError(t, err)
					require.Empty(t, metaDeleted.Key, "response should be empty (silent)!")
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocDeleteWaitForSync(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					t.Run("WithWaitForSync==false should not return an error", func(t *testing.T) {
						doc := DocWithRev{
							Name: "test-wait-for-sync-false",
							Age:  utils.NewT(23),
						}
						meta, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						metaDel, err := col.DeleteDocumentWithOptions(ctx, meta.Key, &arangodb.CollectionDocumentDeleteOptions{
							WithWaitForSync: utils.NewT(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaDel.Key)
					})

					t.Run("WithWaitForSync==true should not return an error", func(t *testing.T) {
						doc := DocWithRev{
							Name: "test-wait-for-sync-true",
							Age:  utils.NewT(23),
						}
						meta, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						metaDel, err := col.DeleteDocumentWithOptions(ctx, meta.Key, &arangodb.CollectionDocumentDeleteOptions{
							WithWaitForSync: utils.NewT(true),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaDel.Key)
					})
				})
			})
		})
	})
}
