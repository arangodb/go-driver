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

					readerCreate, err := col.CreateDocuments(ctx, []any{
						doc, doc2,
					})
					require.NoError(t, err)
					require.Equal(t, 2, readerCreate.Len(), "CreateDocuments should return a reader with 2 documents")

					docs, err := col.ReadDocuments(ctx, []string{
						"test",
						"tz44",
						"test2",
					})
					require.NoError(t, err)
					require.Equal(t, 3, docs.Len(), "ReadDocuments should return a reader with 3 documents")

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

					readeRed, err := col.ReadDocuments(ctx, []string{
						"test", "test2", "nonexistent",
					})
					require.NoError(t, err)
					require.Equal(t, 3, readeRed.Len(), "ReadDocuments should return a reader with 3 documents")
					metaRed, errs := readeRed.ReadAll(&docRedRead)
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
