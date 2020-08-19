//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package tests

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type document struct {
	Key    string      `json:"_key"`
	Fields interface{} `json:",inline"`
}

func insertDocuments(t testing.TB, col arangodb.Collection, documents, batch int, factory func(i int) interface{}) {
	b := make([]document, 0, batch)

	for i := 0; i < documents; i++ {
		b = append(b, document{
			Key:    uuid.New().String(),
			Fields: factory(i),
		})

		if len(b) == batch {
			insertBatch(t, context.Background(), col, &arangodb.CollectionDocumentCreateOptions{
				WithWaitForSync: newBool(true),
			}, b)
			b = b[:0]
		}
	}

	if len(b) > 0 {
		insertBatch(t, context.Background(), col, nil, b)
	}
}

func insertBatch(t testing.TB, ctx context.Context, col arangodb.Collection, opts *arangodb.CollectionDocumentCreateOptions, documents interface{}) {
	results, err := col.CreateDocumentsWithOptions(ctx, documents, opts)
	require.NoError(t, err)
	for {
		meta, next, err := results.Read()
		require.NoError(t, err)
		if !next {
			break
		}

		require.False(t, getBool(meta.Error, false))
	}
}

func Test_BatchInsert(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				insertDocuments(t, col, 2048, 128, func(i int) interface{} {
					return i
				})
			})
		})
	})
}
