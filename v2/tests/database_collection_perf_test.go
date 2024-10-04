//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"testing"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func insertDocuments(t testing.TB, col arangodb.Collection, documents, batch int, factory func(i int) interface{}) {
	b := make([]document, 0, batch)

	for i := 0; i < documents; i++ {
		b = append(b, document{
			basicDocument: newBasicDocument(),
			Fields:        factory(i),
		})

		if len(b) == batch {
			insertBatch(t, context.Background(), col, &arangodb.CollectionDocumentCreateOptions{
				WithWaitForSync: utils.NewType(true),
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
		meta, err := results.Read()
		if shared.IsNoMoreDocuments(err) {
			break
		}
		require.NoError(t, err)

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

func bInsert(b *testing.B, db arangodb.Database, threads int) {
	WithCollection(b, db, nil, func(col arangodb.Collection) {
		withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			b.Run(fmt.Sprintf("With %d", threads), func(b *testing.B) {
				b.SetParallelism(threads)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for {
						if !pb.Next() {
							return
						}

						d := newBasicDocument()

						_, err := col.CreateDocument(ctx, d)
						require.NoError(b, err)
					}
				})
				b.ReportAllocs()
			})
		})
	})
}

func bBatchInsert(b *testing.B, db arangodb.Database, threads int) {
	WithCollection(b, db, nil, func(col arangodb.Collection) {
		withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			b.Run(fmt.Sprintf("With %d", threads), func(b *testing.B) {
				b.SetParallelism(threads)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for {
						if !pb.Next() {
							return
						}

						d := newDocs(512)

						r, err := col.CreateDocuments(ctx, d)
						require.NoError(b, err)

						for {
							_, err := r.Read()
							if shared.IsNoMoreDocuments(err) {
								break
							}
							require.NoError(b, err)
						}
					}
				})
				b.ReportAllocs()
			})
		})
	})
}

func Benchmark_Insert(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			bInsert(b, db, 1)
			bInsert(b, db, 4)
		})
	})
}

func Benchmark_BatchInsert(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			bBatchInsert(b, db, 1)
			bBatchInsert(b, db, 4)
		})
	})
}
