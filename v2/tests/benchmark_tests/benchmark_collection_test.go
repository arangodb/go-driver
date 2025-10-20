package benchmarktests

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

func createClient(b *testing.B) arangodb.Client {
	// Create an HTTP connection to the database
	endpoint, err := connection.NewMaglevHashEndpoints(
		[]string{"https://localhost:7001"},
		connection.RequestDBNameValueExtractor,
	)

	conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

	// Add authentication
	auth := connection.NewBasicAuth("root", "")
	err = conn.SetAuthentication(auth)
	if err != nil {
		log.Fatalf("Failed to set authentication: %v", err)
	}

	// Create a client
	client := arangodb.NewClient(conn)

	return client
}

func ensureDatabase(b *testing.B, client arangodb.Client, name string, opts *arangodb.CreateDatabaseOptions) arangodb.Database {
	db, err := client.CreateDatabase(context.TODO(), name, opts)
	require.NoError(b, err)
	return db
}

func ensureCollection(b *testing.B, db arangodb.Database, name string, opts *arangodb.CreateCollectionPropertiesV2) arangodb.Collection {
	col, err := db.CreateCollectionV2(context.TODO(), name, opts)
	require.NoError(b, err)
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
		globalDB = ensureDatabase(b, globalClient, "bench_db", nil)
		colProps := &arangodb.CreateCollectionPropertiesV2{
			WaitForSync: utils.NewType(false),
		}
		globalCol = ensureCollection(b, globalDB, "bench_col", colProps)
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

	// -----------------------------------------
	// Sub-benchmark 2: Read one document at a time
	// -----------------------------------------
	b.Run("ReadSingleDoc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("doc_%d", i%docSize)
			var doc TestDoc
			_, err := col.ReadDocument(ctx, key, &doc)
			require.NoError(b, err)
		}
	})
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

	// -----------------------------------------
	// Sub-benchmark 2: Update one document at a time
	// -----------------------------------------
	b.Run("UpdateSingleDoc", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("doc_%d", i%docSize)
			var doc TestDoc
			// Read existing document
			_, err := col.ReadDocument(ctx, key, &doc)
			require.NoError(b, err)

			// Update the document
			doc.Name = fmt.Sprintf("updated_%d", i)
			doc.Value += 1

			_, err = col.UpdateDocument(ctx, key, doc)
			require.NoError(b, err)
		}
	})
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

	// -----------------------------------------
	// Sub-benchmark 2: Delete one document at a time
	// -----------------------------------------
	b.Run("DeleteSingleDoc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("doc_%d", i%docSize)
			doc := TestDoc{
				Key:   key,
				Name:  fmt.Sprintf("doc_%d", i),
				Value: i,
			}

			// Create before delete for consistent test
			_, err := col.CreateDocument(ctx, doc)
			require.NoError(b, err)

			_, err = col.DeleteDocumentWithOptions(ctx, key, deleteOpts)
			require.NoError(b, err)
		}
	})
}

func BenchmarkV2BulkInsert10KDocs(b *testing.B) {
	bulkInsert(b, 10000)
}

func BenchmarkV2BulkInsert100KDocs(b *testing.B) {
	bulkInsert(b, 100000)
}

func BenchmarkV2BulkRead10KDocs(b *testing.B) {
	bulkRead(b, 10000)
}

func BenchmarkV2BulkRead100KDocs(b *testing.B) {
	bulkRead(b, 100000)
}

func BenchmarkV2bulkUpdate10KDocs(b *testing.B) {
	bulkUpdate(b, 10000)
}

func BenchmarkV2bulkUpdate100KDocs(b *testing.B) {
	bulkUpdate(b, 100000)
}

func BenchmarkV2BulkDelete10KDocs(b *testing.B) {
	bulkDelete(b, 10000)
}

func BenchmarkV2BulkDelete100KDocs(b *testing.B) {
	bulkDelete(b, 100000)
}
