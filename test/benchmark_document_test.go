//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// BenchmarkCreateDocument measures the CreateDocument operation for a simple document.
func BenchmarkCreateDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			"Jan",
			40 + i,
		}
		if _, err := col.CreateDocument(nil, doc); err != nil {
			b.Fatalf("Failed to create new document: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkCreateDocumentParallel measures parallel CreateDocument operations for a simple document.
func BenchmarkCreateDocumentParallel(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, b)

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			doc := UserDoc{
				"Jan",
				40,
			}
			if _, err := col.CreateDocument(nil, doc); err != nil {
				b.Fatalf("Failed to create new document: %s", describe(err))
			}
		}
	})
	b.ReportAllocs()
}

// BenchmarkReadDocument measures the ReadDocument operation for a simple document.
func BenchmarkReadDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, b)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		b.Fatalf("Failed to create new document: %s", describe(err))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result UserDoc
		if _, err := col.ReadDocument(nil, meta.Key, &result); err != nil {
			b.Errorf("Failed to read document: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkReadDocumentParallel measures parallel ReadDocument operations for a simple document.
func BenchmarkReadDocumentParallel(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, b)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		b.Fatalf("Failed to create new document: %s", describe(err))
	}

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result UserDoc
			if _, err := col.ReadDocument(nil, meta.Key, &result); err != nil {
				b.Errorf("Failed to read document: %s", describe(err))
			}
		}
	})
	b.ReportAllocs()
}

// BenchmarkRemoveDocument measures the RemoveDocument operation for a simple document.
func BenchmarkRemoveDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create document (we don't measure that)
		b.StopTimer()
		doc := UserDoc{
			"Jan",
			40 + i,
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			b.Fatalf("Failed to create new document: %s", describe(err))
		}

		// Now do the real test
		b.StartTimer()
		if _, err := col.RemoveDocument(nil, meta.Key); err != nil {
			b.Errorf("Failed to remove document: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkBatchReadDocuments measures the time to read multiple documents in a batch.
func BenchmarkBatchReadDocuments(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_batch_read_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	col := ensureCollection(nil, db, "benchmark_batch_read_docs", nil, b)

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to read
	metas := make([]driver.DocumentMeta, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("BatchReadUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, describe(err))
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i += batchSize {
		currentBatchSize := batchSize
		if i+batchSize > b.N {
			currentBatchSize = b.N - i
		}

		keys := make([]string, currentBatchSize)
		for j := 0; j < currentBatchSize; j++ {
			keys[j] = metas[i+j].Key
		}

		results := make([]UserDoc, currentBatchSize)
		if _, _, err := col.ReadDocuments(nil, keys, results); err != nil {
			b.Errorf("ReadDocuments failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}
