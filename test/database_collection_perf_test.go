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

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/arangodb/go-driver"
)

// insertDocuments inserts documents in batches for performance testing
func insertDocuments(t testing.TB, col driver.Collection, documents, batch int, factory func(i int) interface{}) {
	b := make([]UserDoc, 0, batch)

	for i := 0; i < documents; i++ {
		b = append(b, UserDoc{
			Name: fmt.Sprintf("User%d", i),
			Age:  factory(i).(int),
		})

		if len(b) == batch {
			insertBatch(t, context.Background(), col, b)
			b = b[:0]
		}
	}

	if len(b) > 0 {
		insertBatch(t, context.Background(), col, b)
	}
}

// insertBatch inserts a batch of documents
func insertBatch(t testing.TB, ctx context.Context, col driver.Collection, documents []UserDoc) {
	metas, errs, err := col.CreateDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("Failed to create documents: %s", describe(err))
	}
	if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Verify all documents were created
	if len(metas) != len(documents) {
		t.Fatalf("Expected %d documents, got %d", len(documents), len(metas))
	}
}

// Test_BatchInsert tests batch document insertion
func Test_BatchInsert(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(context.TODO(), c, "batch_insert_test", nil, t)
	defer func() {
		err := db.Remove(context.TODO())
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(context.TODO(), db, "batch_insert_test", nil, t)

	insertDocuments(t, col, 2048, 128, func(i int) interface{} {
		return i
	})
}

// bInsert performs single document insertion benchmark
func bInsert(b *testing.B, db driver.Database, threads int) {
	col := ensureCollection(context.TODO(), db, "insert_test", nil, b)

	b.Run(fmt.Sprintf("With %d", threads), func(b *testing.B) {
		b.SetParallelism(threads)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for {
				if !pb.Next() {
					return
				}

				doc := UserDoc{
					Name: "Jan",
					Age:  40,
				}

				_, err := col.CreateDocument(context.TODO(), doc)
				if err != nil {
					b.Fatalf("Failed to create new document: %s", describe(err))
				}
			}
		})
		b.ReportAllocs()
	})
}

// bBatchInsert performs batch document insertion benchmark
func bBatchInsert(b *testing.B, db driver.Database, threads int) {
	col := ensureCollection(context.TODO(), db, "batch_insert_test", nil, b)

	b.Run(fmt.Sprintf("With %d", threads), func(b *testing.B) {
		b.SetParallelism(threads)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for {
				if !pb.Next() {
					return
				}

				// Create a batch of 512 documents
				docs := make([]UserDoc, 512)
				for i := 0; i < 512; i++ {
					docs[i] = UserDoc{
						Name: fmt.Sprintf("User%d", i),
						Age:  i,
					}
				}

				metas, errs, err := col.CreateDocuments(context.TODO(), docs)
				if err != nil {
					b.Fatalf("Failed to create documents: %s", describe(err))
				}
				if err := errs.FirstNonNil(); err != nil {
					b.Fatalf("Expected no errors, got first: %s", describe(err))
				}

				// Verify all documents were created
				if len(metas) != len(docs) {
					b.Fatalf("Expected %d documents, got %d", len(docs), len(metas))
				}
			}
		})
		b.ReportAllocs()
	})
}

// Benchmark_Insert measures single document insertion performance
func Benchmark_Insert(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(context.TODO(), c, "insert_test", nil, b)
	defer func() {
		err := db.Remove(context.TODO())
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	bInsert(b, db, 1)
	bInsert(b, db, 4)
}

// Benchmark_BatchInsert measures batch document insertion performance
func Benchmark_BatchInsert(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(context.TODO(), c, "batch_insert_test", nil, b)
	defer func() {
		err := db.Remove(context.TODO())
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	bBatchInsert(b, db, 1)
	bBatchInsert(b, db, 4)
}
