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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_CollectionDocumentsReadWithStringErrorCode(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

					type DocWithNoCode struct {
						Name string `json:"name"`
					}
					doc_with_no_code := DocWithNoCode{
						Name: "DocWithNoCode",
					}
					meta_with_no_code, err := col.CreateDocument(ctx, doc_with_no_code)
					require.NoError(t, err)

					type DocWithCode struct {
						Name  string `json:"name"`
						Error string `json:"error"`
						Code  string `json:"code"`
					}
					doc_with_code := DocWithCode{
						Name: "DocWithCode",
						Code: "777",
					}
					meta_with_code, err := col.CreateDocument(ctx, doc_with_code)
					require.NoError(t, err)

					doc_with_code_2 := DocWithCode{
						Name: "DocWithCode2",
						Code: "222",
					}
					meta_with_code_2, err := col.CreateDocument(ctx, doc_with_code_2)
					require.NoError(t, err)

					type DocWithResponselike struct {
						Rev          string `json:"_rev,omitempty"`
						Key          string `json:"_key,omitempty"`
						Name         string `json:"name"`
						Error        bool   `json:"error,omitempty"`
						Code         int    `json:"code,omitempty"`
						ErrorMessage string `json:"errorMessage,omitempty"`
						ErrorNum     int    `json:"errorNum,omitempty"`
					}
					doc_with_responselike := DocWithResponselike{
						Key:  "key",
						Name: "DocWithResponselike",
						Code: 777,
					}
					meta_with_responselike, err := col.CreateDocument(ctx, doc_with_responselike)
					require.NoError(t, err)

					_, _, _, _ = meta_with_no_code, meta_with_code, meta_with_code_2, meta_with_responselike

					t.Run("sanity check, proper doc should have no error", func(t *testing.T) {
						var docRead DocWithNoCode
						meta, err := col.ReadDocumentWithOptions(ctx, meta_with_no_code.Key, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, meta_with_no_code.Key, meta.Key)
					})
					t.Run("sanity check, proper doc that doesn't exist should have error", func(t *testing.T) {
						var docRead DocWithNoCode
						_, err := col.ReadDocumentWithOptions(ctx, "404", &docRead, nil)
						require.Error(t, err)
						require.Equal(t, 404, err.(shared.ArangoError).Code)
					})
					t.Run("doc with code should have no error", func(t *testing.T) {
						var docRead DocWithCode
						meta, err := col.ReadDocumentWithOptions(ctx, meta_with_code.Key, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, "777", docRead.Code)
						require.Equal(t, meta_with_code.Key, meta.Key)
					})
					t.Run("doc with code that doesn't exist should have error", func(t *testing.T) {
						var docRead DocWithCode
						_, err := col.ReadDocumentWithOptions(ctx, "404", &docRead, nil)
						require.Error(t, err)
						require.Equal(t, 404, err.(shared.ArangoError).Code)
					})
					t.Run("doc with responselike format shouldn't have error", func(t *testing.T) {
						var docRead DocWithResponselike
						meta, err := col.ReadDocumentWithOptions(ctx, meta_with_responselike.Key, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, "key", docRead.Key)
						require.Equal(t, meta_with_responselike.Key, meta.Key)
					})
					t.Run("doc with responselike format that doesn't exist should have error", func(t *testing.T) {
						var docRead DocWithResponselike
						_, err := col.ReadDocumentWithOptions(ctx, "404", &docRead, nil)
						require.Error(t, err)
						require.Equal(t, 404, err.(shared.ArangoError).Code)
					})
					t.Run("docs with code should exist", func(t *testing.T) {
						docsKeys := []DocWithRev{
							{
								Key: meta_with_code.Key,
							},
							{
								Key: meta_with_code_2.Key,
							},
						}

						resp, err := col.ReadDocumentsWithOptions(ctx, &docsKeys, nil)
						require.NoError(t, err)

						var docRead DocWithCode

						_, err = resp.Read(&docRead)
						require.NoError(t, err)
						require.Equal(t, "777", docRead.Code)

						_, err = resp.Read(&docRead)
						require.NoError(t, err)
						require.Equal(t, "222", docRead.Code)

					})

					t.Run("docs with code that doesn't exist should return empty", func(t *testing.T) {
						docsKeys := []DocWithRev{
							{
								Key: "404",
							},
							{
								Key: "404_2",
							},
						}

						resp, err := col.ReadDocumentsWithOptions(ctx, &docsKeys, nil)
						require.NoError(t, err)

						var docRead DocWithCode

						_, err = resp.Read(&docRead)
						require.Error(t, err)
						require.Equal(t, 1202, err.(shared.ArangoError).ErrorNum)

						_, err = resp.Read(&docRead)
						require.Error(t, err)
						require.Equal(t, 1202, err.(shared.ArangoError).ErrorNum)

					})

					t.Run("docs with code mixed existence", func(t *testing.T) {
						docsKeys := []DocWithRev{
							{
								Key: "404",
							},
							{
								Key: meta_with_code_2.Key,
							},
						}

						resp, err := col.ReadDocumentsWithOptions(ctx, &docsKeys, nil)
						require.NoError(t, err)

						var docRead DocWithCode
						_, err = resp.Read(&docRead)

						require.Error(t, err)
						require.Equal(t, 1202, err.(shared.ArangoError).ErrorNum)

						_, err = resp.Read(&docRead)
						require.NoError(t, err)
						require.Equal(t, "222", docRead.Code)

					})
				})
			})
		})
	})
}
