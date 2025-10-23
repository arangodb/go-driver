// DISCLAIMER
//
// # Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

func createClient(b *testing.B) arangodb.Client {
	// Read endpoint from environment variable
	endpointsEnv := os.Getenv("TEST_ENDPOINTS")
	b.Logf("endpointsEnv: %s", endpointsEnv)
	if endpointsEnv == "" {
		endpointsEnv = "http://localhost:7001"
	}
	endpoints := strings.Split(endpointsEnv, ",")

	// Determine if using TLS based on endpoint protocol
	useTLS := strings.HasPrefix(endpoints[0], "https://")

	// Create an HTTP connection to the database
	endpoint, err := connection.NewMaglevHashEndpoints(
		endpoints,
		connection.RequestDBNameValueExtractor,
	)
	if err != nil {
		log.Fatalf("Failed to create endpoints: %v", err)
	}

	// Use HTTP/2 for all connections
	// For plain HTTP, pass true to enable HTTP/2 cleartext (h2c)
	// For HTTPS, pass false to use standard TLS
	var conn connection.Connection
	if useTLS {
		// HTTPS: Use standard HTTP/2 with TLS
		conn = connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, false))
	} else {
		// HTTP: Enable HTTP/2 cleartext (h2c) by passing true
		// This sets up the DialTLSContext to handle plain HTTP with HTTP/2
		conn = connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))
	}

	// Add authentication if required
	// Format: basic:username:password or jwt:username:password
	authEnv := os.Getenv("TEST_AUTHENTICATION")
	if authEnv != "" {
		parts := strings.SplitN(authEnv, ":", 3)
		if len(parts) < 3 {
			log.Fatalf("Invalid TEST_AUTHENTICATION format. Expected 'type:username:password', got: %s", authEnv)
		}
		authType := parts[0]
		username := parts[1]
		password := parts[2]

		var auth connection.Authentication
		switch authType {
		case "basic":
			auth = connection.NewBasicAuth(username, password)
		case "jwt":
			// JWT authentication requires a Wrapper approach, not direct Authentication
			// For now, log a warning and treat as basic auth
			log.Printf("Warning: JWT authentication not fully implemented in benchmarks. Using basic auth instead.")
			auth = connection.NewBasicAuth(username, password)
		default:
			log.Fatalf("Unsupported authentication type: %s. Supported types: basic, jwt", authType)
		}

		err = conn.SetAuthentication(auth)
		if err != nil {
			log.Fatalf("Failed to set authentication: %v", err)
		}
	}

	// Create a client
	client := arangodb.NewClient(conn)

	return client
}

func ensureDatabase(b *testing.B, client arangodb.Client, name string, opts *arangodb.CreateDatabaseOptions) arangodb.Database {
	ctx := context.TODO()

	// Try to get existing database first
	db, err := client.GetDatabase(ctx, name, nil)
	if err == nil {
		b.Logf("Using existing database: %s", name)
		return db
	}

	// Database doesn't exist, create it
	db, err = client.CreateDatabase(ctx, name, opts)
	require.NoError(b, err)
	b.Logf("Created new database: %s", name)
	return db
}

func ensureCollection(b *testing.B, db arangodb.Database, name string, opts *arangodb.CreateCollectionPropertiesV2) arangodb.Collection {
	ctx := context.TODO()

	// Try to get existing collection first
	col, err := db.GetCollection(ctx, name, nil)
	if err == nil {
		b.Logf("Using existing collection: %s", name)
		return col
	}

	// Collection doesn't exist, create it
	col, err = db.CreateCollectionV2(ctx, name, opts)
	require.NoError(b, err)
	b.Logf("Created new collection: %s", name)
	return col
}

var (
	globalClient arangodb.Client
	globalDB     arangodb.Database
	globalCol    arangodb.Collection
	once         sync.Once
)

func setup(b *testing.B) (arangodb.Database, arangodb.Collection) {
	once.Do(func() {
		globalClient = createClient(b)
		globalDB = ensureDatabase(b, globalClient, "bench_db_v2", nil)
		colProps := &arangodb.CreateCollectionPropertiesV2{
			WaitForSync: utils.NewType(false),
		}
		globalCol = ensureCollection(b, globalDB, "bench_col_v2", colProps)
	})
	return globalDB, globalCol
}

type TestDoc struct {
	Key   string `json:"_key"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func bulkInsert(b *testing.B, docSize int) {
	_, col := setup(b)

	// prepare docs
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
	opts := &arangodb.CollectionDocumentCreateOptions{
		Overwrite:       utils.NewType(true),
		WithWaitForSync: utils.NewType(false),
		Silent:          utils.NewType(true),
	}

	// Benchmark Insert
	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := col.CreateDocumentsWithOptions(ctx, docs, opts)
			require.NoError(b, err)
			for {
				_, err := resp.Read()
				if shared.IsNoMoreDocuments(err) {
					break
				}
				require.NoError(b, err)
			}
		}
	})
}

func BenchmarkV2BulkInsert100KDocs(b *testing.B) {
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

	opts := &arangodb.CollectionDocumentCreateOptions{
		Overwrite: utils.NewType(true),
		Silent:    utils.NewType(true),
	}

	// Insert all docs once before benchmarking
	_, err = col.CreateDocumentsWithOptions(ctx, docs, opts)
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
				if shared.IsNoMoreDocuments(err) {
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

func BenchmarkV2BulkRead100KDocs(b *testing.B) {
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

	opts := &arangodb.CollectionDocumentCreateOptions{
		Overwrite: utils.NewType(true),
		Silent:    utils.NewType(true),
	}

	// Insert all docs once before benchmarking
	_, err = col.CreateDocumentsWithOptions(ctx, docs, opts)
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

			updateOpts := &arangodb.CollectionDocumentUpdateOptions{
				IgnoreRevs: utils.NewType(true),
				Silent:     utils.NewType(true),
			}

			_, err := col.UpdateDocumentsWithOptions(ctx, updatedDocs, updateOpts)
			require.NoError(b, err)
		}
	})
}

func BenchmarkV2bulkUpdate100KDocs(b *testing.B) {
	bulkUpdate(b, 100000)
}

// bulkDelete benchmarks document deletion performance
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

	createOpts := &arangodb.CollectionDocumentCreateOptions{
		Overwrite: utils.NewType(true),
		Silent:    utils.NewType(true),
	}

	deleteOpts := &arangodb.CollectionDocumentDeleteOptions{
		Silent: utils.NewType(true),
	}

	// Insert all docs before benchmarking
	_, err = col.CreateDocumentsWithOptions(ctx, docs, createOpts)
	require.NoError(b, err)

	// -----------------------------------------
	// Sub-benchmark 1: Delete entire collection at once
	// -----------------------------------------
	b.Run("DeleteAllDocsOnce", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Re-insert documents before each delete iteration
			_, err := col.CreateDocumentsWithOptions(ctx, docs, createOpts)
			require.NoError(b, err)

			keys := make([]string, docSize)
			for j := 0; j < docSize; j++ {
				keys[j] = docs[j].Key
			}

			_, err = col.DeleteDocumentsWithOptions(ctx, keys, deleteOpts)
			require.NoError(b, err)
		}
	})
}

func BenchmarkV2BulkDelete100KDocs(b *testing.B) {
	bulkDelete(b, 100000)
}
