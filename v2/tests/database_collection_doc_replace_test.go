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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_DatabaseCollectionDocReplaceIfMatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-if-match",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					var oldDoc DocWithRev
					var newDoc DocWithRev

					docReplace := DocWithRev{
						Name: "test-if-match-REPLACED",
					}

					t.Run("do not replace if rev doesn't match", func(t *testing.T) {
						metaError, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
							OldObject: &oldDoc,
							NewObject: &newDoc,
							IfMatch:   "wrong-rev",
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do a replace if rev does match", func(t *testing.T) {
						metaReplaced, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
							OldObject: &oldDoc,
							NewObject: &newDoc,
							IfMatch:   meta.Rev,
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaReplaced.Rev)
						require.NotEqual(t, metaReplaced.Rev, meta.Rev)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocReplaceIgnoreRevs(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-IgnoreRevs",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					docReplace := DocWithRev{
						Name: "test-IgnoreRevs-REPLACED",
					}

					t.Run("do not replace if rev doesn't match", func(t *testing.T) {
						docReplace.Rev = "wrong-rev"
						metaError, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
							IgnoreRevs: newBool(false),
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do an update if rev match", func(t *testing.T) {
						docReplace.Rev = meta.Rev
						metaReplaced, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
							IgnoreRevs: newBool(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaReplaced.Rev)
						require.NotEqual(t, metaReplaced.Rev, meta.Rev)
					})

					t.Run("do an update if rev is missing", func(t *testing.T) {
						docReplace.Rev = ""
						metaReplaced, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
							IgnoreRevs: newBool(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaReplaced.Rev)
						require.NotEqual(t, metaReplaced.Rev, meta.Rev)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocReplaceSilent(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.12", t)

					doc := DocWithRev{
						Name: "test-silent",
						Age:  newInt(42),
					}
					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					docReplace := DocWithRev{
						Name: "test-silent-updated",
					}
					metaUpdated, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplace, &arangodb.CollectionDocumentReplaceOptions{
						Silent: newBool(true),
					})
					require.NoError(t, err)
					require.Empty(t, metaUpdated.Key, "response should be empty (silent)!")
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocReplaceWaitForSync(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-wait-for-sync",
						Age:  newInt(23),
					}
					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("WithWaitForSync==false should not return an error", func(t *testing.T) {
						doc.Age = newInt(42)
						meta, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, doc, &arangodb.CollectionDocumentReplaceOptions{
							WithWaitForSync: newBool(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})

					t.Run("WithWaitForSync==true should not return an error", func(t *testing.T) {
						doc.Age = newInt(32)
						meta, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, doc, &arangodb.CollectionDocumentReplaceOptions{
							WithWaitForSync: newBool(true),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocReplaceVersionAttribute(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		skipBelowVersion(client, nil, "3.12", t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-version-attribute",
						Age:  newInt(23),
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not replace if age is lower", func(t *testing.T) {
						var newDoc DocWithRev
						var oldDoc DocWithRev

						docReplaced := DocWithRev{
							Name: "test-check-Replaced",
							Age:  newInt(19),
						}

						metaDoc, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplaced, &arangodb.CollectionDocumentReplaceOptions{
							NewObject:        &newDoc,
							OldObject:        &oldDoc,
							VersionAttribute: "age",
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaDoc.Key)
						require.NotEmpty(t, newDoc)
						require.NotEmpty(t, oldDoc)
						require.Equal(t, newDoc.Rev, oldDoc.Rev)
						require.Equal(t, newDoc.Age, doc.Age)
					})

					t.Run("Replace if age is higher", func(t *testing.T) {
						var newDoc DocWithRev
						var oldDoc DocWithRev

						docReplaced := DocWithRev{
							Name: "test-check-Replaced",
							Age:  newInt(99),
						}

						metaDoc, err := col.ReplaceDocumentWithOptions(ctx, meta.Key, docReplaced, &arangodb.CollectionDocumentReplaceOptions{
							NewObject:        &newDoc,
							OldObject:        &oldDoc,
							VersionAttribute: "age",
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaDoc.Key)
						require.NotEmpty(t, newDoc)
						require.NotEmpty(t, oldDoc)
						require.NotEqual(t, newDoc.Rev, oldDoc.Rev)
						require.NotEqual(t, newDoc.Age, doc.Age)
					})
				})
			})
		})
	})
}
