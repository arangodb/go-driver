//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
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

func Test_DatabaseCollectionDocUpdateIfMatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-if-match",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					var oldDoc DocWithRev
					var newDoc DocWithRev

					docUpdate := DocWithRev{
						Name: "test-if-match-UPDATED",
					}

					t.Run("do not update if rev doesn't match", func(t *testing.T) {
						metaError, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
							OldObject: &oldDoc,
							NewObject: &newDoc,
							IfMatch:   "wrong-rev",
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do an update if rev does match", func(t *testing.T) {
						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
							OldObject: &oldDoc,
							NewObject: &newDoc,
							IfMatch:   meta.Rev,
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaUpdated.Rev)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)
					})
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_DatabaseCollectionDocUpdateIgnoreRevs(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-IgnoreRevs",
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					docUpdate := DocWithRev{
						Name: "test-IgnoreRevs-UPDATED",
					}

					t.Run("do not update if rev doesn't match", func(t *testing.T) {
						docUpdate.Rev = "wrong-rev"
						metaError, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.Error(t, err)
						require.Empty(t, metaError.Rev)
					})

					t.Run("do an update if rev match", func(t *testing.T) {
						docUpdate.Rev = meta.Rev
						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaUpdated.Rev)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)
					})

					t.Run("do an update if rev is missing", func(t *testing.T) {
						docUpdate.Rev = ""
						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
							IgnoreRevs: utils.NewType(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, metaUpdated.Rev)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)
					})
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_DatabaseCollectionDocUpdateReturnOldRev(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-ORG",
					}

					metaOrg, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					docUpdate := DocWithRev{
						Name: "test-UPDATED",
					}

					t.Run("OldRev should match", func(t *testing.T) {
						metaRep, err := col.UpdateDocumentWithOptions(ctx, metaOrg.Key, docUpdate, nil)
						require.NoError(t, err)
						require.Equal(t, metaOrg.Rev, metaRep.OldRev)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocUpdateKeepNull(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

					doc := DocWithRev{
						Name: "test-keep-null",
						Age:  utils.NewType(10),
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

						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docOverwrite, &arangodb.CollectionDocumentUpdateOptions{
							KeepNull: utils.NewType(true),
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

						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docOverwrite, &arangodb.CollectionDocumentUpdateOptions{
							KeepNull: utils.NewType(false),
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
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_DatabaseCollectionDocUpdateMergeObjects(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
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

						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docOverwrite, &arangodb.CollectionDocumentUpdateOptions{
							MergeObjects: utils.NewType(true),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docReadAfterUpdate DocWithRev
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docReadAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Key, metaUpdated.Key)
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

						metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docOverwrite, &arangodb.CollectionDocumentUpdateOptions{
							MergeObjects: utils.NewType(false),
						})
						require.NoError(t, err)
						require.Equal(t, metaUpdated.Key, meta.Key)
						require.NotEqual(t, metaUpdated.Rev, meta.Rev)

						var docReadAfterUpdate DocWithRev
						metaRead, err := col.ReadDocument(ctx, meta.Key, &docReadAfterUpdate)
						require.NoError(t, err)
						require.Equal(t, metaRead.Key, metaUpdated.Key)
						require.NotEqual(t, docReadAfterUpdate.Name, doc.Name)

						// Countries are not merged
						require.Len(t, docReadAfterUpdate.Countries, 1)
						require.Contains(t, docReadAfterUpdate.Countries, "Portugal")
					})
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_DatabaseCollectionDocUpdateSilent(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.12", t)

					doc := DocWithRev{
						Name: "test-silent",
						Age:  utils.NewType(42),
					}
					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					docUpdate := DocWithRev{
						Name: "test-silent-updated",
					}
					metaUpdated, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
						Silent: utils.NewType(true),
					})
					require.NoError(t, err)
					require.Empty(t, metaUpdated.Key, "response should be empty (silent)!")
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocUpdateWaitForSync(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-wait-for-sync",
						Age:  utils.NewType(23),
					}
					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("WithWaitForSync==false should not return an error", func(t *testing.T) {
						doc.Age = utils.NewType(42)
						meta, err := col.UpdateDocumentWithOptions(ctx, meta.Key, doc, &arangodb.CollectionDocumentUpdateOptions{
							WithWaitForSync: utils.NewType(false),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})

					t.Run("WithWaitForSync==true should not return an error", func(t *testing.T) {
						doc.Age = utils.NewType(32)
						meta, err := col.UpdateDocumentWithOptions(ctx, meta.Key, doc, &arangodb.CollectionDocumentUpdateOptions{
							WithWaitForSync: utils.NewType(true),
						})
						require.NoError(t, err)
						require.NotEmpty(t, meta.Key)
					})
				})
			})
		})
	})
}

func Test_DatabaseCollectionDocUpdateVersionAttribute(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		skipBelowVersion(client, nil, "3.12", t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithRev{
						Name: "test-version-attribute",
						Age:  utils.NewType(23),
					}

					meta, err := col.CreateDocument(ctx, doc)
					require.NoError(t, err)

					t.Run("do not update if age is lower", func(t *testing.T) {
						var newDoc DocWithRev
						var oldDoc DocWithRev

						docUpdate := DocWithRev{
							Name: "test-check-UPDATED",
							Age:  utils.NewType(19),
						}

						metaDoc, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
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

					t.Run("update if age is higher", func(t *testing.T) {
						var newDoc DocWithRev
						var oldDoc DocWithRev

						docUpdate := DocWithRev{
							Name: "test-check-UPDATED",
							Age:  utils.NewType(99),
						}

						metaDoc, err := col.UpdateDocumentWithOptions(ctx, meta.Key, docUpdate, &arangodb.CollectionDocumentUpdateOptions{
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
