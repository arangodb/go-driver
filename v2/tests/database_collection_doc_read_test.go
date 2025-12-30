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
)

func Test_DatabaseCollectionDocReadIfMatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-if-match",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not fetch if rev doesn't match", func(t *testing.T) {
						var docRead DocWithRev
						metaError, err := col.ReadDocumentWithOptions(ctx, meta.Key, &docRead, &arangodb.CollectionDocumentReadOptions{
							IfMatch: "wrong-rev",
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do a fetch if rev does match", func(t *testing.T) {
						var docRead DocWithRev
						metaRead, err := col.ReadDocumentWithOptions(ctx, meta.Key, &docRead, &arangodb.CollectionDocumentReadOptions{
							IfMatch: meta.Rev,
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaRead.Rev)
						require.Equal(t, metaRead.Rev, meta.Rev)
						require.Equal(t, docRead.Name, doc.Name)
					})

					t.Run("do a fetch if NONE rev doesn't match", func(t *testing.T) {
						var docRead DocWithRev
						metaRead, err := col.ReadDocumentWithOptions(ctx, meta.Key, &docRead, &arangodb.CollectionDocumentReadOptions{
							IfNoneMatch: "wrong-rev",
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaRead.Rev)
						require.Equal(t, metaRead.Rev, meta.Rev)
						require.Equal(t, docRead.Name, doc.Name)
					})

					t.Run("do not fetch if NONE rev match", func(t *testing.T) {
						var docRead DocWithRev
						metaError, err := col.ReadDocumentWithOptions(ctx, meta.Key, &docRead, &arangodb.CollectionDocumentReadOptions{
							IfNoneMatch: meta.Rev,
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocReadIgnoreRevs(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-IgnoreRevs",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not fetch if rev doesn't match", func(t *testing.T) {
						docToRead := []DocWithRev{
							{
								Key: meta.Key,
								Rev: "wrong-rev",
							},
						}

						r, err := col.ReadDocumentsWithOptions(ctx, &docToRead, &arangodb.CollectionDocumentReadOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.NoError(t, err)

						var docRead DocWithRev
						metaErr, errDel := r.Read(&docRead)
						require.Error(t, errDel)
						require.NotEmpty(t, metaErr.ErrorNum)
						require.Equal(t, 1200, *metaErr.ErrorNum)
					})

					t.Run("read a doc if rev match", func(t *testing.T) {
						docToRead := []DocWithRev{
							{
								Key: meta.Key,
								Rev: meta.Rev,
							},
						}

						r, err := col.ReadDocumentsWithOptions(ctx, &docToRead, &arangodb.CollectionDocumentReadOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.NoError(t, err)

						var docRead DocWithRev
						_, errDel := r.Read(&docRead)
						require.NoError(t, errDel)
					})

					t.Run("read a doc if rev is missing", func(t *testing.T) {
						docToRead := []DocWithRev{
							{
								Key: meta.Key,
							},
						}

						r, err := col.ReadDocumentsWithOptions(ctx, &docToRead, &arangodb.CollectionDocumentReadOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.NoError(t, err)

						var docRead DocWithRev
						_, errDel := r.Read(&docRead)
						require.NoError(t, errDel)
					})
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_DatabaseCollectionDocReadAllowDirtyReads(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					t.Run("WithWaitForSync==false should not return an error", func(t *testing.T) {
						doc := DocWithRev{
							Name: "test-wait-for-sync-false",
							Age:  utils.NewType(23),
						}
						meta, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						tr, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{}, nil)
						require.NoError(t, err)

						metaRead, err := col.ReadDocumentWithOptions(ctx, meta.Key, &DocWithRev{}, &arangodb.CollectionDocumentReadOptions{
							TransactionID: string(tr.ID()),
						})
						require.NoError(t, err)
						require.Equal(t, metaRead.Key, meta.Key)
					})
				})
			})
		})
	})
}
