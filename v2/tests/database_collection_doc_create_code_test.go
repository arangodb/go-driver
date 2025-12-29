//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/utils"
)

type DocWithCode struct {
	Key  string `json:"_key,omitempty"`
	Code string `json:"code"`
}

func Test_DatabaseCollectionDocCreateCode(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithCode{
						Key: "test",
					}

					meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
					require.NoError(t, err)
					require.NotEmpty(t, meta.Rev)
					require.Empty(t, meta.Old)
					require.Empty(t, meta.New)

					rdoc, err := col.ReadDocument(ctx, "test", &doc)
					require.NoError(t, err)

					require.EqualValues(t, "test", rdoc.Key)
				})
			})
		})

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := DocWithCode{
						Key: "test",
					}
					doc2 := DocWithCode{
						Key: "test2",
					}

					_, err := col.CreateDocuments(ctx, []any{
						doc, doc2,
					})
					require.NoError(t, err)

					docs, err := col.ReadDocuments(ctx, []string{
						"test",
						"tz44",
						"test2",
					})
					require.NoError(t, err)

					var z DocWithCode

					meta, err := docs.Read(&z)
					require.NoError(t, err)
					require.Equal(t, "test", meta.Key)

					_, err = docs.Read(&z)
					require.Error(t, err)
					require.True(t, shared.IsNotFound(err))

					meta, err = docs.Read(&z)
					require.NoError(t, err)
					require.Equal(t, "test2", meta.Key)

					_, err = docs.Read(&z)
					require.Error(t, err)
					require.True(t, shared.IsNoMoreDocuments(err))
				})
			})
		})

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc1 := DocWithCode{
						Key:  "test",
						Code: "code1",
					}
					doc2 := DocWithCode{
						Key:  "test2",
						Code: "code2",
					}
					readerCrt, err := col.CreateDocuments(ctx, []any{doc1, doc2})
					require.NoError(t, err)
					metaCrt, errs := readerCrt.ReadAll()
					require.Equal(t, 2, len(metaCrt)) // Verify we got 2 results
					require.ElementsMatch(t, []any{doc1.Key, doc2.Key}, []any{metaCrt[0].Key, metaCrt[1].Key})
					require.ElementsMatch(t, []any{nil, nil}, errs)

					var docRedRead []DocWithCode

					readerRead, err := col.ReadDocuments(ctx, []string{
						"test", "test2", "nonexistent",
					})
					require.NoError(t, err)
					require.Equal(t, 3, readerRead.Len(), "ReadDocuments should return a reader with 3 documents")
					metaRed, errs := readerRead.ReadAll(&docRedRead)
					require.ElementsMatch(t, []any{doc1.Key, doc2.Key}, []any{metaRed[0].Key, metaRed[1].Key})
					require.Nil(t, errs[0])
					require.Nil(t, errs[1])
					require.Error(t, errs[2])
					require.True(t, shared.IsArangoErrorWithErrorNum(errs[2], shared.ErrArangoDocumentNotFound))

					var docOldObject DocWithCode
					var docDelRead []DocWithCode

					readerDel, err := col.DeleteDocumentsWithOptions(ctx, []string{
						"test", "test2", "nonexistent",
					}, &arangodb.CollectionDocumentDeleteOptions{OldObject: &docOldObject})
					require.NoError(t, err)
					metaDel, errs := readerDel.ReadAll(&docDelRead)
					require.Equal(t, 3, readerDel.Len(), "ReadAll() should return 3 results matching number of delete attempts")

					require.ElementsMatch(t, []any{doc1.Key, doc2.Key, ""}, []any{metaDel[0].Key, metaDel[1].Key, metaDel[2].Key})
					require.Nil(t, errs[0])
					require.Nil(t, errs[1])
					require.Error(t, errs[2])
					require.True(t, shared.IsArangoErrorWithErrorNum(errs[2], shared.ErrArangoDocumentNotFound))

					// Now this should work correctly with separate Old objects
					require.ElementsMatch(t, []any{doc1.Code, doc2.Code}, []any{metaDel[0].Old.(*DocWithCode).Code, metaDel[1].Old.(*DocWithCode).Code})

				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})

}

func Test_DatabaseCollectionDocReaderLen(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// Create test documents
					doc1 := DocWithCode{Key: "len-test-1", Code: "code1"}
					doc2 := DocWithCode{Key: "len-test-2", Code: "code2"}
					doc3 := DocWithCode{Key: "len-test-3", Code: "code3"}

					// Test CreateDocuments reader Len() behavior
					readerCreate, err := col.CreateDocuments(ctx, []any{doc1, doc2, doc3})
					require.NoError(t, err)

					// Test Len() before any Read() calls
					initialLen := readerCreate.Len()
					require.Equal(t, 3, initialLen, "Len() should return 3 before any Read() calls")

					// Read one document
					meta1, err := readerCreate.Read()
					require.NoError(t, err)
					require.Equal(t, "len-test-1", meta1.Key)

					// Test Len() during iteration - should still return the same value
					midLen := readerCreate.Len()
					require.Equal(t, 3, midLen, "Len() should return 3 during iteration")
					require.Equal(t, initialLen, midLen, "Len() should be consistent throughout iteration")

					// Read another document
					meta2, err := readerCreate.Read()
					require.NoError(t, err)
					require.Equal(t, "len-test-2", meta2.Key)

					// Test Len() again during iteration
					midLen2 := readerCreate.Len()
					require.Equal(t, 3, midLen2, "Len() should still return 3 after reading 2 documents")

					// Read the final document
					meta3, err := readerCreate.Read()
					require.NoError(t, err)
					require.Equal(t, "len-test-3", meta3.Key)

					// Test Len() after all documents are read
					finalLen := readerCreate.Len()
					require.Equal(t, 3, finalLen, "Len() should return 3 after all documents are read")

					// Try to read one more time - should get NoMoreDocuments error
					_, err = readerCreate.Read()
					require.Error(t, err)
					require.True(t, shared.IsNoMoreDocuments(err))

					// Test Len() after iteration is complete
					afterCompleteLen := readerCreate.Len()
					require.Equal(t, 3, afterCompleteLen, "Len() should return 3 even after iteration is complete")

					// Test ReadDocuments reader Len() behavior
					readerRead, err := col.ReadDocuments(ctx, []string{"len-test-1", "len-test-2", "len-test-3"})
					require.NoError(t, err)

					// Test Len() before any Read() calls
					readInitialLen := readerRead.Len()
					require.Equal(t, 3, readInitialLen, "ReadDocuments Len() should return 3 before any Read() calls")

					// Read all documents using ReadAll to test that Len() doesn't interfere
					var readResults []DocWithCode
					metas, errs := readerRead.ReadAll(&readResults)
					require.Equal(t, 3, len(metas))
					require.Equal(t, 3, len(readResults))
					require.ElementsMatch(t, []any{nil, nil, nil}, errs)

					// Test Len() after ReadAll
					readAfterAllLen := readerRead.Len()
					require.Equal(t, 3, readAfterAllLen, "Len() should return 3 after ReadAll()")

					// Test DeleteDocuments reader Len() behavior with OldObject
					var oldObj DocWithCode
					readerDelete, err := col.DeleteDocumentsWithOptions(ctx, []string{"len-test-1", "len-test-2", "len-test-3"},
						&arangodb.CollectionDocumentDeleteOptions{OldObject: &oldObj})
					require.NoError(t, err)

					deleteLen := readerDelete.Len()
					require.Equal(t, 3, deleteLen, "DeleteDocuments Len() should return 3")

					// Use ReadAll to consume the reader and test Len() after
					var deleteResults []DocWithCode
					deleteMetas, _ := readerDelete.ReadAll(&deleteResults)
					require.Equal(t, 3, len(deleteMetas))
					require.Equal(t, 3, len(deleteResults))

					// Len() should still be 3 after ReadAll
					deleteAfterAllLen := readerDelete.Len()
					require.Equal(t, 3, deleteAfterAllLen, "Len() should remain 3 after ReadAll on delete reader")
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})

}
