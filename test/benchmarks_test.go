//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

type TestDoc struct {
	Key   string `json:"_key"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

var (
	globalClient driver.Client
	globalDB     driver.Database
	globalCol    driver.Collection
	once         sync.Once
)

func setup(b *testing.B) (driver.Database, driver.Collection) {
	once.Do(func() {
		globalClient = createClient(b, nil)
		globalDB = ensureDatabase(context.TODO(), globalClient, "bench_db_v1", nil, b)
		globalCol = ensureCollection(context.TODO(), globalDB, "bench_col_v1", nil, b)
	})
	return globalDB, globalCol
}

func bulkInsert(b *testing.B, docSize int) {
	_, col := setup(b)

	docs := make([]TestDoc, docSize)
	for i := 0; i < docSize; i++ {
		docs[i] = TestDoc{
			Key:   fmt.Sprintf("doc_%d", i),
			Name:  strconv.Itoa(i),
			Value: i,
		}
	}

	ctx := context.TODO()
	err := col.Truncate(ctx)
	require.NoError(b, err, "failed to truncate collection before insert")

	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, _, err := col.CreateDocuments(ctx, docs)
			require.NoError(b, err)
			require.Equal(b, len(resp.Keys()), docSize)
		}
	})
}

func BenchmarkV1BulkInsert100KDocs(b *testing.B) {
	bulkInsert(b, 100000)
}

func bulkRead(b *testing.B, docSize int) {
	db, col := setup(b)

	// -----------------------------
	// Prepare and insert documents
	// -----------------------------
	docs := make([]TestDoc, docSize)
	for i := 0; i < docSize; i++ {
		docs[i] = TestDoc{
			Key:   fmt.Sprintf("doc_%d", i),
			Name:  strconv.Itoa(i),
			Value: i,
		}
	}

	ctx := context.TODO()
	err := col.Truncate(ctx)
	require.NoError(b, err, "failed to truncate collection before insert")

	// Insert all docs once before benchmarking
	_, _, err = col.CreateDocuments(ctx, docs)
	require.NoError(b, err)

	// -----------------------------------------
	// Sub-benchmark 1: Read entire collection
	// -----------------------------------------
	b.Run("ReadAllDocsOnce", func(b *testing.B) {
		query := fmt.Sprintf("FOR d IN %s RETURN d", col.Name())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cursor, err := db.Query(ctx, query, nil)
			require.NoError(b, err)

			count := 0
			for {
				var doc TestDoc
				_, err := cursor.ReadDocument(ctx, &doc)
				if driver.IsNoMoreDocuments(err) {
					break
				}
				require.NoError(b, err)
				count++
			}
			// require.Equal(b, docSize, count, "expected to read all documents")
			_ = cursor.Close()
			// sanity check
			if count != docSize {
				b.Fatalf("expected to read %d docs, got %d", docSize, count)
			}
		}
	})
}

func BenchmarkV1BulkRead100KDocs(b *testing.B) {
	bulkRead(b, 100000)
}

func bulkUpdate(b *testing.B, docSize int) {
	_, col := setup(b)

	// -----------------------------
	// Prepare and insert documents
	// -----------------------------
	docs := make([]TestDoc, docSize)
	for i := 0; i < docSize; i++ {
		docs[i] = TestDoc{
			Key:   fmt.Sprintf("doc_%d", i),
			Name:  strconv.Itoa(i),
			Value: i,
		}
	}

	ctx := context.TODO()
	err := col.Truncate(ctx)
	require.NoError(b, err, "failed to truncate collection before insert")

	// Insert all docs once before benchmarking
	metas, _, err := col.CreateDocuments(ctx, docs)
	require.NoError(b, err)

	// -----------------------------------------
	// Sub-benchmark 1: Update entire collection at once
	// -----------------------------------------
	b.Run("UpdateAllDocsOnce", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			updatedDocs := make([]TestDoc, docSize)
			for j := 0; j < docSize; j++ {
				updatedDocs[j] = TestDoc{
					Key:   docs[j].Key,
					Name:  fmt.Sprintf("updated_%d", j),
					Value: docs[j].Value + 1,
				}
			}

			_, _, err := col.UpdateDocuments(ctx, metas.Keys(), updatedDocs)
			require.NoError(b, err)
		}
	})
}

func BenchmarkV1bulkUpdate100KDocs(b *testing.B) {
	bulkUpdate(b, 100000)
}

func bulkDelete(b *testing.B, docSize int) {
	_, col := setup(b) // setup() initializes connection & collection

	// -----------------------------
	// Prepare initial dataset
	// -----------------------------
	docs := make([]TestDoc, docSize)
	for i := 0; i < docSize; i++ {
		docs[i] = TestDoc{
			Key:   fmt.Sprintf("doc_%d", i),
			Name:  strconv.Itoa(i),
			Value: i,
		}
	}

	ctx := context.TODO()
	err := col.Truncate(ctx)
	require.NoError(b, err, "failed to truncate collection before insert")

	// Insert all docs before benchmarking
	_, _, err = col.CreateDocuments(ctx, docs)
	require.NoError(b, err)

	// -----------------------------------------
	// Sub-benchmark 1: Delete entire collection at once
	// -----------------------------------------
	b.Run("DeleteAllDocsOnce", func(b *testing.B) {
		// Extract keys from docs
		keys := make([]string, docSize)
		for j := 0; j < docSize; j++ {
			keys[j] = docs[j].Key
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Re-insert documents before each delete iteration
			_, _, err := col.CreateDocuments(ctx, docs)
			require.NoError(b, err)

			_, _, err = col.RemoveDocuments(ctx, keys)
			require.NoError(b, err)
		}
	})
}

func BenchmarkV1BulkDelete100KDocs(b *testing.B) {
	bulkDelete(b, 100000)
}
