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

// BenchmarkConnectionInitialization measures the time to create a new client connection.
func BenchmarkConnectionInitialization(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := createClient(b, nil)
		if c == nil {
			b.Error("Failed to create client")
		}
	}
	b.ReportAllocs()
}

// BenchmarkCreateCollection measures the time to create a new collection.
func BenchmarkCreateCollection(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		colName := fmt.Sprintf("bench_col_%d", i)
		col, err := db.CreateCollection(nil, colName, nil)
		if err != nil {
			b.Errorf("CreateCollection failed: %s", describe(err))
		}
		// Clean up the collection immediately to avoid accumulation
		if err := col.Remove(nil); err != nil {
			b.Logf("Failed to remove collection %s: %s", colName, err)
		}
	}
	b.ReportAllocs()
}

// BenchmarkInsertSingleDocument measures the time to insert a single document.
func BenchmarkInsertSingleDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_document_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_docs", nil, b)

	// Pre-create documents to avoid allocation during benchmark
	docs := make([]UserDoc, b.N)
	for i := 0; i < b.N; i++ {
		docs[i] = UserDoc{
			Name: fmt.Sprintf("User_%d", i),
			Age:  20 + (i % 50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := col.CreateDocument(nil, docs[i]); err != nil {
			b.Errorf("CreateDocument failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkInsertBatchDocuments measures the time to insert documents in batches.
func BenchmarkInsertBatchDocuments(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_batch_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_batch_docs", nil, b)

	// Use batch size of 100 documents
	batchSize := 100
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to avoid allocation during benchmark
	docs := make([]UserDoc, batchSize)
	for i := 0; i < batchSize; i++ {
		docs[i] = UserDoc{
			Name: fmt.Sprintf("BatchUser_%d", i),
			Age:  20 + (i % 50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i += batchSize {
		currentBatchSize := batchSize
		if i+batchSize > b.N {
			currentBatchSize = b.N - i
		}
		if _, _, err := col.CreateDocuments(nil, docs[:currentBatchSize]); err != nil {
			b.Errorf("CreateDocuments failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkSimpleQuery measures the time to execute simple AQL queries.
func BenchmarkSimpleQuery(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_query_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_query_docs", nil, b)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("QueryUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	if _, _, err := col.CreateDocuments(nil, testDocs); err != nil {
		b.Fatalf("Failed to create test documents: %s", describe(err))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := "FOR doc IN benchmark_query_docs FILTER doc.age > 30 RETURN doc"
		cur, err := db.Query(nil, query, nil)
		if err != nil {
			b.Errorf("Query failed: %s", describe(err))
		} else {
			cur.Close()
		}
	}
	b.ReportAllocs()
}

// BenchmarkAQLWithBindParameters measures the time to execute AQL queries with bind parameters.
func BenchmarkAQLWithBindParameters(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_bind_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_bind_docs", nil, b)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("BindUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	if _, _, err := col.CreateDocuments(nil, testDocs); err != nil {
		b.Fatalf("Failed to create test documents: %s", describe(err))
	}

	query := "FOR doc IN benchmark_bind_docs FILTER doc.age > @minAge AND doc.age < @maxAge RETURN doc"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindVars := map[string]interface{}{
			"minAge": 20 + (i % 20),
			"maxAge": 40 + (i % 20),
		}
		cur, err := db.Query(nil, query, bindVars)
		if err != nil {
			b.Errorf("Query with bind parameters failed: %s", describe(err))
		} else {
			cur.Close()
		}
	}
	b.ReportAllocs()
}

// BenchmarkCursorIteration measures the time to iterate over query results.
func BenchmarkCursorIteration(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_cursor_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_cursor_docs", nil, b)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("CursorUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	if _, _, err := col.CreateDocuments(nil, testDocs); err != nil {
		b.Fatalf("Failed to create test documents: %s", describe(err))
	}

	query := "FOR doc IN benchmark_cursor_docs RETURN doc"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cur, err := db.Query(nil, query, nil)
		if err != nil {
			b.Errorf("Query failed: %s", describe(err))
			continue
		}

		// Iterate through all results
		for cur.HasMore() {
			var doc UserDoc
			if _, err := cur.ReadDocument(nil, &doc); err != nil {
				b.Errorf("ReadDocument failed: %s", describe(err))
				break
			}
		}
		cur.Close()
	}
	b.ReportAllocs()
}

// BenchmarkUpdateDocument measures the time to update documents.
func BenchmarkUpdateDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_update_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_update_docs", nil, b)

	// Pre-create documents to update
	metas := make([]driver.DocumentMeta, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("UpdateUser_%d", i),
			Age:  20,
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, describe(err))
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		update := map[string]interface{}{
			"age": 30 + (i % 20),
		}
		if _, err := col.UpdateDocument(nil, metas[i].Key, update); err != nil {
			b.Errorf("UpdateDocument failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkDeleteDocument measures the time to delete documents.
func BenchmarkDeleteDocument(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_delete_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	// Create a new collection for each benchmark iteration to avoid running out of documents
	colName := fmt.Sprintf("benchmark_delete_docs_%d", b.N)
	col := ensureCollection(nil, db, colName, nil, b)

	// Pre-create documents to delete
	metas := make([]driver.DocumentMeta, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("DeleteUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, describe(err))
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := col.RemoveDocument(nil, metas[i].Key); err != nil {
			b.Errorf("RemoveDocument failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkBatchUpdateDocuments measures the time to update multiple documents in a batch.
func BenchmarkBatchUpdateDocuments(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_batch_update_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "benchmark_batch_update_docs", nil, b)

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to update
	metas := make([]driver.DocumentMeta, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("BatchUpdateUser_%d", i),
			Age:  20,
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, describe(err))
		}
		metas[i] = meta
	}

	// Pre-create update data
	updates := make([]map[string]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		updates[i] = map[string]interface{}{
			"age": 30 + (i % 20),
		}
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

		if _, _, err := col.UpdateDocuments(nil, keys, updates[:currentBatchSize]); err != nil {
			b.Errorf("UpdateDocuments failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}

// BenchmarkBatchDeleteDocuments measures the time to delete multiple documents in a batch.
func BenchmarkBatchDeleteDocuments(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "benchmark_batch_delete_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			b.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Create a new collection for each benchmark iteration to avoid running out of documents
	colName := fmt.Sprintf("benchmark_batch_delete_docs_%d", b.N)
	col := ensureCollection(nil, db, colName, nil, b)

	// Pre-create documents to delete
	metas := make([]driver.DocumentMeta, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("BatchDeleteUser_%d", i),
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

		if _, _, err := col.RemoveDocuments(nil, keys); err != nil {
			b.Errorf("RemoveDocuments failed: %s", describe(err))
		}
	}
	b.ReportAllocs()
}
