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

func Test_DatabaseCollectionDocCreateOverwrite(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-overwrite",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)
					require.NotEmpty(t, meta.Rev)
					require.Empty(t, meta.Old)
					require.Empty(t, meta.New)

					var oldDoc DocWithRev
					var newDoc DocWithRev

					t.Run("same doc should be created with different key", func(t *testing.T) {
						meta2, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{
							OldObject: &oldDoc,
							NewObject: &newDoc,
						})
						require.NoError(t, err)
						require.Empty(t, meta2.Old)
						require.NotEmpty(t, meta2.New)
						require.Equal(t, meta2.Rev, newDoc.Rev)
						require.NotEqual(t, meta2.Key, meta.Key)
					})

					docOverwrite := DocWithRev{
						Key:  meta.Key,
						Name: "test-overwrite-2",
					}
					t.Run("doc should not be replaced if the key is the same and overwrite is not allowed", func(t *testing.T) {
						meta, err := col.CreateDocument(ctx, docOverwrite)
						require.Error(t, err)
						require.Empty(t, meta.Rev)

						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeConflict
						metaConflict, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							OldObject:     &oldDoc,
							NewObject:     &newDoc,
							OverwriteMode: overwriteMode.New(),
						})
						require.Error(t, err)
						require.Empty(t, metaConflict.Rev)
						require.Empty(t, metaConflict.Old)
						require.Empty(t, metaConflict.New)
					})

					t.Run("replace doc should be ignored", func(t *testing.T) {
						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeIgnore
						metaIgnore, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							OldObject:     &oldDoc,
							NewObject:     &newDoc,
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaIgnore.Rev, meta.Rev)
					})

					t.Run("replace doc should be allowed", func(t *testing.T) {
						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeReplace
						metaReplaced, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							OldObject:     &oldDoc,
							NewObject:     &newDoc,
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaReplaced.Rev, newDoc.Rev)
						require.NotEqual(t, metaReplaced.Rev, meta.Rev)

						require.NotEmpty(t, metaReplaced.Old)
						require.NotEmpty(t, metaReplaced.New)
						require.Equal(t, oldDoc.Name, doc.Name)
						require.Equal(t, newDoc.Name, docOverwrite.Name)

						t.Run("replace doc should be allowed (simple approach)", func(t *testing.T) {
							metaReplacedSimple, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
								OldObject: &oldDoc,
								NewObject: &newDoc,
								Overwrite: utils.NewT(true),
							})
							require.NoError(t, err)
							require.NotEqual(t, metaReplacedSimple.Rev, metaReplaced.Rev)
							require.NotEqual(t, metaReplacedSimple.Rev, meta.Rev)
						})
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocCreateKeepNull(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

					doc := DocWithRev{
						Name: "test-keep-null",
						Age:  utils.NewT(10),
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)
					require.NotEmpty(t, meta.Rev)

					t.Run("update doc with keepNull==true, should leave the field in struct", func(t *testing.T) {
						docOverwrite := DocWithRev{
							Key:  meta.Key,
							Name: "test-keep-null-2",
							Age:  nil,
						}

						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeUpdate
						metaUpdated, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							KeepNull:      utils.NewT(true),
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docRawAfterUpdate map[string]interface{}
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docRawAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Key, metaUpdated.Key)

						// Age field should be in the document
						require.Contains(t, docRawAfterUpdate, "age")
						require.Equal(t, docRawAfterUpdate["age"], nil)
					})

					t.Run("update doc with keepNull==false, should remove empty fields from the struct", func(t *testing.T) {
						docOverwrite := DocWithRev{
							Key:  meta.Key,
							Name: "test-keep-null-3",
							Age:  nil,
						}

						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeUpdate
						metaUpdated, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							KeepNull:      utils.NewT(false),
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docRawAfterUpdate map[string]interface{}
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docRawAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Key, metaUpdated.Key)

						// Age field should be removed from the document
						require.NotContains(t, docRawAfterUpdate, "age")
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocCreateMergeObjects(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-merge",
						Countries: map[string]int{
							"Germany": 1,
							"France":  2,
						},
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)
					require.NotEmpty(t, meta.Rev)

					t.Run("update doc with mergeObjects==true", func(t *testing.T) {
						docOverwrite := DocWithRev{
							Key: meta.Key,
							Countries: map[string]int{
								"Poland": 3,
								"Spain":  4,
							},
						}

						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeUpdate
						metaUpdated, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							MergeObjects:  utils.NewT(true),
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docReadAfterUpdate DocWithRev
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docReadAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Rev, metaUpdated.Rev)
						require.NotEqual(t, docReadAfterUpdate.Name, doc.Name)

						// Countries are merged
						require.Len(t, docReadAfterUpdate.Countries, 4)
						require.Contains(t, docReadAfterUpdate.Countries, "Poland")
						require.Contains(t, docReadAfterUpdate.Countries, "Spain")
						require.Contains(t, docReadAfterUpdate.Countries, "Germany")
					})

					t.Run("update doc with mergeObjects==false", func(t *testing.T) {
						docOverwrite := DocWithRev{
							Key: meta.Key,
							Countries: map[string]int{
								"Portugal": 5,
							},
						}

						overwriteMode := arangodb.CollectionDocumentCreateOverwriteModeUpdate
						metaUpdated, err := col.CreateDocumentWithOptions(ctx, docOverwrite, &arangodb.CollectionDocumentCreateOptions{
							MergeObjects:  utils.NewT(false),
							OverwriteMode: overwriteMode.New(),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docReadAfterUpdate DocWithRev
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docReadAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Rev, metaUpdated.Rev)
						require.NotEqual(t, docReadAfterUpdate.Name, doc.Name)

						// Countries are not merged
						require.Len(t, docReadAfterUpdate.Countries, 1)
						require.Contains(t, docReadAfterUpdate.Countries, "Portugal")
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocCreateSilent(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.12", t)

					doc := DocWithRev{
						Name: "test-silent",
						Age:  utils.NewT(42),
					}

					meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{
						Silent: utils.NewT(true),
					})
					require.NoError(t, err)
					require.Empty(t, meta.Key, "response should be empty (silent)!")
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocCreateWaitForSync(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-wait-for-sync",
						Age:  utils.NewT(23),
					}

					t.Run("WithWaitForSync==false should not return an error", func(t *testing.T) {
						meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{
							WithWaitForSync: utils.NewT(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})

					t.Run("WithWaitForSync==true should not return an error", func(t *testing.T) {
						meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{
							WithWaitForSync: utils.NewT(true),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocCreateReplaceWithVersionAttribute(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		skipBelowVersion(client, nil, "3.12", t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-version-attribute",
						Age:  utils.NewT(23),
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not replace if age is lower", func(t *testing.T) {
						var newDoc DocWithRev
						var oldDoc DocWithRev

						docReplaced := DocWithRev{
							Name: "test-check-Replaced",
							Age:  utils.NewT(19),
							Key:  meta.Key,
						}

						metaDoc, err := col.CreateDocumentWithOptions(ctx, docReplaced, &arangodb.CollectionDocumentCreateOptions{
							NewObject:        &newDoc,
							OldObject:        &oldDoc,
							Overwrite:        utils.NewT(true),
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
							Age:  utils.NewT(99),
							Key:  meta.Key,
						}

						metaDoc, err := col.CreateDocumentWithOptions(ctx, docReplaced, &arangodb.CollectionDocumentCreateOptions{
							NewObject:        &newDoc,
							OldObject:        &oldDoc,
							Overwrite:        utils.NewT(true),
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
