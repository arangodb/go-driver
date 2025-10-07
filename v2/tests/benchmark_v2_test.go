//
// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

// createBenchmarkClient creates a client for benchmarks without the complex connection waiting
func createBenchmarkClient(b *testing.B) arangodb.Client {
	conn := connectionJsonHttp(b)
	client := arangodb.NewClient(conn)
	return client
}

// setupBenchmarkDB creates a database and collection for benchmarks with shorter timeouts
func setupBenchmarkDB(b *testing.B, client arangodb.Client) (arangodb.Database, arangodb.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := GenerateUUID("bench-db")
	db, err := client.CreateDatabase(ctx, dbName, nil)
	if err != nil {
		b.Fatalf("Failed to create database: %s", err)
	}

	colName := GenerateUUID("bench-col")
	col, err := db.CreateCollectionV2(ctx, colName, nil)
	if err != nil {
		b.Fatalf("Failed to create collection: %s", err)
	}

	return db, col
}

// cleanupBenchmarkDB removes the database created for benchmarks
func cleanupBenchmarkDB(b *testing.B, db arangodb.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.Remove(ctx); err != nil {
		b.Logf("Failed to remove database: %s", err)
	}
}

// BenchmarkV2ConnectionInitialization measures the time to create a new V2 client connection.
func BenchmarkV2ConnectionInitialization(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := connectionJsonHttp(b)
		client := arangodb.NewClient(conn)
		if client == nil {
			b.Error("Failed to create V2 client")
		}
	}
}

// BenchmarkCreateDocument measures the CreateDocument operation for a simple document.
func BenchmarkCreateDocumentV2(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	colName := GenerateUUID("bench-col")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("CreateCollectionV2 failed: %s", err)
	}
	defer func() {
		if err := col.Remove(context.Background()); err != nil {
			b.Logf("Failed to remove collection %s: %s", colName, err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			"Jan",
			40 + i,
		}
		if _, err := col.CreateDocument(nil, doc); err != nil {
			b.Fatalf("Failed to create new document: %v", err)
		}
	}
}

// BenchmarkCreateDocumentParallel measures parallel CreateDocument operations for a simple document.
func BenchmarkCreateDocumentParallel(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	colName := GenerateUUID("bench-col")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("CreateCollectionV2 failed: %s", err)
	}
	defer func() {
		if err := col.Remove(context.Background()); err != nil {
			b.Logf("Failed to remove collection %s: %s", colName, err)
		}
	}()

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			doc := UserDoc{
				"Jan",
				40,
			}
			if _, err := col.CreateDocument(nil, doc); err != nil {
				b.Fatalf("Failed to create new document: %v", err)
			}
		}
	})
}

// BenchmarkV2CreateCollection measures the time to create a new collection using V2 API.
func BenchmarkV2CreateCollection(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		colName := GenerateUUID("bench-col")
		col, err := db.CreateCollectionV2(context.Background(), colName, nil)
		if err != nil {
			b.Errorf("CreateCollectionV2 failed: %s", err)
		} else {
			// Clean up the collection immediately to avoid accumulation
			if err := col.Remove(context.Background()); err != nil {
				b.Logf("Failed to remove collection %s: %s", colName, err)
			}
		}
	}
}

// BenchmarkV2InsertSingleDocument measures the time to insert a single document using V2 API.
func BenchmarkV2InsertSingleDocument(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Pre-create documents to avoid allocation during benchmark
	docs := make([]UserDoc, b.N)
	for i := 0; i < b.N; i++ {
		docs[i] = UserDoc{
			Name: fmt.Sprintf("V2User_%d", i),
			Age:  20 + (i % 50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := col.CreateDocument(context.Background(), docs[i]); err != nil {
			b.Errorf("CreateDocument failed: %s", err)
		}
	}
}

// BenchmarkV2InsertBatchDocuments measures the time to insert documents in batches using V2 API.
func BenchmarkV2InsertBatchDocuments(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Use batch size of 100 documents
	batchSize := 100
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to avoid allocation during benchmark
	docs := make([]UserDoc, batchSize)
	for i := 0; i < batchSize; i++ {
		docs[i] = UserDoc{
			Name: fmt.Sprintf("V2BatchUser_%d", i),
			Age:  20 + (i % 50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i += batchSize {
		currentBatchSize := batchSize
		if i+batchSize > b.N {
			currentBatchSize = b.N - i
		}
		reader, err := col.CreateDocuments(context.Background(), docs[:currentBatchSize])
		if err != nil {
			b.Errorf("CreateDocuments failed: %s", err)
		} else {
			// Consume the reader to complete the operation
			for {
				_, err := reader.Read()
				if shared.IsNoMoreDocuments(err) {
					break
				}
				if err != nil {
					b.Errorf("CreateDocuments read failed: %s", err)
					break
				}
			}
		}
	}
}

// BenchmarkV2SimpleQuery measures the time to execute simple AQL queries using V2 API.
func BenchmarkV2SimpleQuery(b *testing.B) {
	client := createBenchmarkClient(b)
	db, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("V2QueryUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	reader, err := col.CreateDocuments(context.Background(), testDocs)
	if err != nil {
		b.Fatalf("Failed to create test documents: %s", err)
	}
	// Consume the reader
	for {
		_, err := reader.Read()
		if shared.IsNoMoreDocuments(err) {
			break
		}
		if err != nil {
			b.Fatalf("Failed to read test documents: %s", err)
		}
	}

	query := fmt.Sprintf("FOR doc IN `%s` FILTER doc.age > 30 RETURN doc", col.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cur, err := db.Query(context.Background(), query, nil)
		if err != nil {
			b.Errorf("Query failed: %s", err)
		} else {
			cur.Close()
		}
	}
}

// BenchmarkV2AQLWithBindParameters measures the time to execute AQL queries with bind parameters using V2 API.
func BenchmarkV2AQLWithBindParameters(b *testing.B) {
	client := createBenchmarkClient(b)
	db, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("V2BindUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	reader, err := col.CreateDocuments(context.Background(), testDocs)
	if err != nil {
		b.Fatalf("Failed to create test documents: %s", err)
	}
	// Consume the reader
	for {
		_, err := reader.Read()
		if shared.IsNoMoreDocuments(err) {
			break
		}
		if err != nil {
			b.Fatalf("Failed to read test documents: %s", err)
		}
	}

	query := fmt.Sprintf("FOR doc IN `%s` FILTER doc.age > @minAge AND doc.age < @maxAge RETURN doc", col.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindVars := map[string]interface{}{
			"minAge": 20 + (i % 20),
			"maxAge": 40 + (i % 20),
		}
		cur, err := db.Query(context.Background(), query, &arangodb.QueryOptions{
			BindVars: bindVars,
		})
		if err != nil {
			b.Errorf("Query with bind parameters failed: %s", err)
		} else {
			cur.Close()
		}
	}
}

// BenchmarkV2CursorIteration measures the time to iterate over query results using V2 API.
func BenchmarkV2CursorIteration(b *testing.B) {
	client := createBenchmarkClient(b)
	db, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Pre-populate collection with test data
	testDocs := make([]UserDoc, 1000)
	for i := 0; i < 1000; i++ {
		testDocs[i] = UserDoc{
			Name: fmt.Sprintf("V2CursorUser_%d", i),
			Age:  20 + (i % 50),
		}
	}
	reader, err := col.CreateDocuments(context.Background(), testDocs)
	if err != nil {
		b.Fatalf("Failed to create test documents: %s", err)
	}
	// Consume the reader
	for {
		_, err := reader.Read()
		if shared.IsNoMoreDocuments(err) {
			break
		}
		if err != nil {
			b.Fatalf("Failed to read test documents: %s", err)
		}
	}

	query := fmt.Sprintf("FOR doc IN `%s` RETURN doc", col.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cur, err := db.Query(context.Background(), query, nil)
		if err != nil {
			b.Errorf("Query failed: %s", err)
			continue
		}

		// Iterate through all results
		for {
			var doc UserDoc
			_, err := cur.ReadDocument(context.Background(), &doc)
			if shared.IsNoMoreDocuments(err) {
				break
			}
			if err != nil {
				b.Errorf("ReadDocument failed: %s", err)
				break
			}
		}
		cur.Close()
	}
}

// BenchmarkV2UpdateDocument measures the time to update documents using V2 API.
func BenchmarkV2UpdateDocument(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Pre-create documents to update
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2UpdateUser_%d", i),
			Age:  20,
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		update := map[string]interface{}{
			"age": 30 + (i % 20),
		}
		if _, err := col.UpdateDocument(context.Background(), metas[i].Key, update); err != nil {
			b.Errorf("UpdateDocument failed: %s", err)
		}
	}
}

// BenchmarkV2DeleteDocument measures the time to delete documents using V2 API.
func BenchmarkV2DeleteDocument(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Create a new collection for each benchmark iteration to avoid running out of documents
	colName := GenerateUUID("benchmark-delete-docs")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("Failed to create collection: %s", err)
	}

	// Pre-create documents to delete
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2DeleteUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := col.DeleteDocument(context.Background(), metas[i].Key); err != nil {
			b.Errorf("DeleteDocument failed: %s", err)
		}
	}
}

// BenchmarkV2BatchUpdateDocuments measures the time to update multiple documents in a batch using V2 API.
func BenchmarkV2BatchUpdateDocuments(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to update
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2BatchUpdateUser_%d", i),
			Age:  20,
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
		}
		metas[i] = meta
	}

	// Pre-create update data
	updates := make([]map[string]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		updates[i] = map[string]interface{}{
			"_key": "", // Will be set per batch
			"age":  30 + (i % 20),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i += batchSize {
		currentBatchSize := batchSize
		if i+batchSize > b.N {
			currentBatchSize = b.N - i
		}

		// Set the keys for this batch
		for j := 0; j < currentBatchSize; j++ {
			updates[j]["_key"] = metas[i+j].Key
		}

		reader, err := col.UpdateDocuments(context.Background(), updates[:currentBatchSize])
		if err != nil {
			b.Errorf("UpdateDocuments failed: %s", err)
		} else {
			// Consume the reader to complete the operation
			for {
				_, err := reader.Read()
				if shared.IsNoMoreDocuments(err) {
					break
				}
				if err != nil {
					b.Errorf("UpdateDocuments read failed: %s", err)
					break
				}
			}
		}
	}
}

// BenchmarkV2BatchDeleteDocuments measures the time to delete multiple documents in a batch using V2 API.
func BenchmarkV2BatchDeleteDocuments(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Create a new collection for each benchmark iteration to avoid running out of documents
	colName := GenerateUUID("benchmark-batch-delete-docs")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("Failed to create collection: %s", err)
	}

	// Pre-create documents to delete
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2BatchDeleteUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
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

		reader, err := col.DeleteDocuments(context.Background(), keys)
		if err != nil {
			b.Errorf("DeleteDocuments failed: %s", err)
		} else {
			// Consume the reader to complete the operation
			for {
				var result arangodb.CollectionDocumentDeleteResponse
				_, err := reader.Read(&result)
				if shared.IsNoMoreDocuments(err) {
					break
				}
				if err != nil {
					b.Errorf("DeleteDocuments read failed: %s", err)
					break
				}
			}
		}
	}
}

// BenchmarkV2ReadDocument measures the time to read documents using V2 API.
func BenchmarkV2ReadDocument(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Pre-create documents to read
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2ReadUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
		}
		metas[i] = meta
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var doc UserDoc
		if _, err := col.ReadDocument(context.Background(), metas[i].Key, &doc); err != nil {
			b.Errorf("ReadDocument failed: %s", err)
		}
	}
}

// BenchmarkReadDocumentParallel measures parallel ReadDocument operations for a simple document.
func BenchmarkV2ReadDocumentParallel(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	colName := GenerateUUID("bench-col")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("CreateCollectionV2 failed: %s", err)
	}
	defer func() {
		if err := col.Remove(context.Background()); err != nil {
			b.Logf("Failed to remove collection %s: %s", colName, err)
		}
	}()

	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(context.Background(), doc)
	if err != nil {
		b.Fatalf("Failed to create new document: %s", err)
	}

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result UserDoc
			if _, err := col.ReadDocument(context.Background(), meta.Key, &result); err != nil {
				b.Errorf("Failed to read document: %s", err)
			}
		}
	})
}

// BenchmarkV2BatchReadDocuments measures the time to read multiple documents in a batch using V2 API.
func BenchmarkV2BatchReadDocuments(b *testing.B) {
	client := createBenchmarkClient(b)
	_, col := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, col.Database())

	// Use batch size of 50 documents
	batchSize := 50
	if b.N < batchSize {
		batchSize = b.N
	}

	// Pre-create documents to read
	metas := make([]arangodb.CollectionDocumentCreateResponse, b.N)
	for i := 0; i < b.N; i++ {
		doc := UserDoc{
			Name: fmt.Sprintf("V2BatchReadUser_%d", i),
			Age:  20 + (i % 50),
		}
		meta, err := col.CreateDocument(context.Background(), doc)
		if err != nil {
			b.Fatalf("Failed to create document %d: %s", i, err)
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

		reader, err := col.ReadDocuments(context.Background(), keys)
		if err != nil {
			b.Errorf("ReadDocuments failed: %s", err)
		} else {
			// Consume the reader to complete the operation
			for {
				var result UserDoc
				_, err := reader.Read(&result)
				if shared.IsNoMoreDocuments(err) {
					break
				}
				if err != nil {
					b.Errorf("ReadDocuments read failed: %s", err)
					break
				}
			}
		}
	}
}

// BenchmarkV2CollectionExists measures the time to check if a collection exists.
func BenchmarkV2CollectionExists(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	colName := GenerateUUID("bench-col")
	col, err := db.CreateCollectionV2(context.Background(), colName, nil)
	if err != nil {
		b.Fatalf("Failed to create collection: %s", err)
	}
	defer col.Remove(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exists, err := db.CollectionExists(context.Background(), colName)
		if err != nil {
			b.Errorf("Database.CollectionExists failed: %s", err)
		}
		if !exists {
			b.Error("Collection should exist")
		}
	}
}

// BenchmarkV2ListCollections measures the time to list all collections in a database.
func BenchmarkV2ListCollections(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	// Create multiple collections
	for i := 0; i < 10; i++ {
		colName := GenerateUUID(fmt.Sprintf("bench-col-%d", i))
		col, err := db.CreateCollectionV2(context.Background(), colName, nil)
		if err != nil {
			b.Fatalf("Failed to create collection %d: %s", i, err)
		}
		defer col.Remove(context.Background())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collections, err := db.Collections(context.Background())
		if err != nil {
			b.Errorf("Database.Collections failed: %s", err)
		}
		if len(collections) == 0 {
			b.Error("Should have collections")
		}
	}
}

// BenchmarkV2DatabaseExists measures the time to check if a database exists.
func BenchmarkV2DatabaseExists(b *testing.B) {
	client := createBenchmarkClient(b)
	db, _ := setupBenchmarkDB(b, client)
	defer cleanupBenchmarkDB(b, db)

	dbName := db.Name()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exists, err := client.DatabaseExists(context.Background(), dbName)
		if err != nil {
			b.Errorf("Client.DatabaseExists failed: %s", err)
		}
		if !exists {
			b.Error("Database should exist")
		}
	}
}
