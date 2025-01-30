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
						Rev  string `json:"_rev,omitempty"`
						Key  string `json:"_key,omitempty"`
						Name string `json:"name"`
					}
					doc_with_no_code := DocWithNoCode{
						Name: "DocWithNoCode",
					}
					meta_with_no_code, err := col.CreateDocument(ctx, doc_with_no_code)
					require.NoError(t, err)

					type DocWithCode struct {
						Rev  string `json:"_rev,omitempty"`
						Key  string `json:"_key,omitempty"`
						Name string `json:"name"`
						Code string `json:"code"`
					}
					doc_with_code := DocWithCode{
						Name: "DocWithCode",
						Code: "777",
					}
					meta_with_code, err := col.CreateDocument(ctx, doc_with_code)
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
						Name: "DocWithResponselike",
						Code: 777,
					}
					meta_with_responselike, err := col.CreateDocument(ctx, doc_with_responselike)
					require.NoError(t, err)

					t.Run("sanity check, proper doc should have no error", func(t *testing.T) {
						var docRead DocWithNoCode
						_, err := col.ReadDocumentWithOptions(ctx, meta_with_no_code.Key, &docRead, nil)
						require.NoError(t, err)
					})
					t.Run("sanity check, proper doc that doesn't exist should have error", func(t *testing.T) {
						var docRead DocWithNoCode
						_, err := col.ReadDocumentWithOptions(ctx, "404", &docRead, nil)
						require.Error(t, err)
						require.Equal(t, err.(shared.ArangoError).Code, 404)
					})
					t.Run("doc with code should have no error", func(t *testing.T) {
						var docRead DocWithCode
						_, err := col.ReadDocumentWithOptions(ctx, meta_with_code.Key, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, docRead.Code, "777")
					})
					t.Run("doc with code that doesn't exist should have error", func(t *testing.T) {
						var docRead DocWithCode
						_, err := col.ReadDocumentWithOptions(ctx, "404", &docRead, nil)
						require.Error(t, err)
						require.Equal(t, err.(shared.ArangoError).Code, 404)
					})
					t.Run("doc with responselike format should be doc", func(t *testing.T) {
						var docRead DocWithResponselike
						_, err := col.ReadDocumentWithOptions(ctx, meta_with_responselike.Key, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, docRead.Code, 777)
					})
				})
			})
		})
	})
}
