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
	"context"
	"fmt"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// BenchmarkCollectionExists measures the CollectionExists operation.
func BenchmarkCollectionExists(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "collection_exist_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.CollectionExists(nil, col.Name()); err != nil {
			b.Errorf("CollectionExists failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkCollection measures the Collection operation.
func BenchmarkCollection(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "collection_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Collection(nil, col.Name()); err != nil {
			b.Errorf("Collection failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkCollections measures the Collections operation.
func BenchmarkCollections(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	for i := 0; i < 10; i++ {
		ensureCollection(nil, db, fmt.Sprintf("col%d", i), nil, b)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Collections(nil); err != nil {
			b.Errorf("Collections failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// runComprehensiveDocumentOperationsV1 runs comprehensive document operations with a specific number of pre-created documents using V1 API
func runComprehensiveDocumentOperationsV1(b *testing.B, col driver.Collection, numDocs int) {
	// Pre-create documents for read/update/delete operations
	b.Logf("Pre-creating %d documents for comprehensive V1 benchmark", numDocs)
	metas := make([]driver.DocumentMeta, numDocs)
	for i := 0; i < numDocs; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("ComprehensiveUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(context.TODO(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, describe(err))
		}
		metas[i] = meta
	}

	b.ResetTimer()

	// Measure create operations
	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc := UserDoc{
				Name: fmt.Sprintf("CreateUser_%d_%d", i, b.N),
				Age:  20 + (i % 50),
			}
			if _, err := col.CreateDocument(context.TODO(), doc); err != nil {
				b.Errorf("CreateDocument failed: %s", describe(err))
			}
		}
		b.ReportAllocs()
	})

	// Measure read operations using pre-created documents
	b.Run("Read", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			docIndex := i % numDocs
			var result UserDoc
			if _, err := col.ReadDocument(context.TODO(), metas[docIndex].Key, &result); err != nil {
				b.Errorf("ReadDocument failed: %s", describe(err))
			}
		}
		b.ReportAllocs()
	})

	// Measure update operations using pre-created documents
	b.Run("Update", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			docIndex := i % numDocs
			update := map[string]interface{}{
				"age": 30 + (i % 20),
			}
			if _, err := col.UpdateDocument(context.TODO(), metas[docIndex].Key, update); err != nil {
				b.Errorf("UpdateDocument failed: %s", describe(err))
			}
		}
		b.ReportAllocs()
	})

	// Measure delete operations (create fresh documents for each deletion)
	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a fresh document for deletion
			doc := UserDoc{
				Name: fmt.Sprintf("DeleteUser_%d", i),
				Age:  20 + (i % 50),
			}

			// Create the document
			meta, err := col.CreateDocument(context.TODO(), doc)
			if err != nil {
				b.Errorf("CreateDocument failed: %s", describe(err))
				continue
			}

			// Delete the document
			if _, err := col.RemoveDocument(context.TODO(), meta.Key); err != nil {
				b.Errorf("RemoveDocument failed: %s", describe(err))
			}
		}
		b.ReportAllocs()
	})
}

// BenchmarkComprehensiveDocumentOperations_1K measures comprehensive document operations with 1000 pre-created documents using V1 API
func BenchmarkComprehensiveDocumentOperations_1K(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(context.TODO(), c, "benchmark_comprehensive_1k_test", nil, b)
	defer func() {
		err := db.Remove(context.TODO())
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(context.TODO(), db, "benchmark_comprehensive_1k_docs", nil, b)

	runComprehensiveDocumentOperationsV1(b, col, 1000)
}

// BenchmarkComprehensiveDocumentOperations_10K measures comprehensive document operations with 10000 pre-created documents using V1 API
func BenchmarkComprehensiveDocumentOperations_10K(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(context.TODO(), c, "benchmark_comprehensive_10k_test", nil, b)
	defer func() {
		err := db.Remove(context.TODO())
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(context.TODO(), db, "benchmark_comprehensive_10k_docs", nil, b)

	runComprehensiveDocumentOperationsV1(b, col, 10000)
}
