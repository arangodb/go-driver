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
)

type DocWithNameCode struct {
	Key  string `json:"_key,omitempty"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func Test_DatabaseCollectionDocImport(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					doc := `{"_key":"john","name":"John Smith","age":35}
{"_key":"katie","name":"Katie Foster","age":28}
`

					resp, err := col.ImportDocuments(ctx, doc, arangodb.ImportDocumentTypeDocuments)
					require.NoError(t, err)
					_ = resp

					var obj DocWithNameCode
					meta, err := col.ReadDocument(ctx, "john", &obj)
					require.NoError(t, err)
					require.Equal(t, "john", meta.Key)
					require.Equal(t, 35, obj.Age)

					meta, err = col.ReadDocument(ctx, "katie", &obj)
					require.NoError(t, err)
					require.Equal(t, "katie", meta.Key)
					require.Equal(t, 28, obj.Age)
				})
			})

			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// type = array / auto
					doc := `[
	{"_key":"john","name":"John Smith","age":35},
	{"_key":"katie","name":"Katie Foster","age":28}
]
`
					_, err := col.ImportDocuments(ctx, doc, arangodb.ImportDocumentTypeArray)
					require.NoError(t, err)

					var obj DocWithNameCode
					meta, err := col.ReadDocument(ctx, "john", &obj)
					require.NoError(t, err)
					require.Equal(t, "john", meta.Key)
					require.Equal(t, 35, obj.Age)

					meta, err = col.ReadDocument(ctx, "katie", &obj)
					require.NoError(t, err)
					require.Equal(t, "katie", meta.Key)
					require.Equal(t, 28, obj.Age)
				})
			})

			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// type = <omitted>
					doc := `["_key","name","age"]
["john","John Smith",35]
["katie","Katie Foster",28]
`
					_, err := col.ImportDocuments(ctx, doc, arangodb.ImportDocumentTypeTabular)
					require.NoError(t, err)

					var obj DocWithNameCode
					meta, err := col.ReadDocument(ctx, "john", &obj)
					require.NoError(t, err)
					require.Equal(t, "john", meta.Key)
					require.Equal(t, 35, obj.Age)

					meta, err = col.ReadDocument(ctx, "katie", &obj)
					require.NoError(t, err)
					require.Equal(t, "katie", meta.Key)
					require.Equal(t, 28, obj.Age)
				})
			})
		})

	})
}
