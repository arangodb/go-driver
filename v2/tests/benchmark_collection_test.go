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

package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// BenchmarkV2ConnectionInitialization measures the time to create a new client connection.
func BenchmarkV2ConnectionInitialization(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c := newClient(b, connectionJsonHttp(b))
			if c == nil {
				b.Error("Failed to create client")
			}
		}
		b.ReportAllocs()
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	})
}

// BenchmarkV2CollectionCreation measures the time to create a new collection.
func BenchmarkV2CollectionCreation(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					colName := GenerateUUID("benchmark-COL")
					_, err := db.CreateCollectionV2(ctx, colName, nil)
					if err != nil {
						b.Errorf("CreateCollection failed: %s", err)
					}
				}
				b.ReportAllocs()
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			})
		})
	})
}

// BenchmarkV2CollectionCreationWithProperties measures the time to create a new collection with properties.
func BenchmarkV2CollectionCreationWithProperties(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				props := &arangodb.CreateCollectionPropertiesV2{
					NumberOfShards:    func() *int { v := 1; return &v }(),
					ReplicationFactor: func() *arangodb.ReplicationFactor { v := arangodb.ReplicationFactor(1); return &v }(),
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					colName := GenerateUUID("benchmark-COL")
					_, err := db.CreateCollectionV2(ctx, colName, props)
					if err != nil {
						b.Errorf("CreateCollectionWithProperties failed: %s", err)
					}
				}
				b.ReportAllocs()
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
			})
		})
	})
}

// BenchmarkV2CollectionExists measures the CollectionExists operation.
func BenchmarkV2CollectionExists(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := db.GetCollection(ctx, col.Name(), nil)
						if err != nil {
							b.Errorf("CollectionExists failed: %s", err)
						}
					}
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2Collection measures the Collection operation.
func BenchmarkV2Collection(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := db.GetCollection(ctx, col.Name(), nil)
						if err != nil {
							b.Errorf("Collection failed: %s", err)
						}
					}
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2Collections measures the Collections operation.
func BenchmarkV2Collections(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				// Create multiple collections for the BenchmarkV2
				for i := 0; i < 10; i++ {
					colName := GenerateUUID("test-COL")
					_, err := db.CreateCollectionV2(ctx, colName, nil)
					if err != nil {
						b.Fatalf("Failed to create collection %s: %s", colName, err)
					}
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := db.Collections(ctx)
					if err != nil {
						b.Errorf("Collections failed: %s", err)
					}
				}
				b.ReportAllocs()
			})
		})
	})
}

// BenchmarkV2ComprehensiveDocumentOperations measures the complete lifecycle of documents:
// create, read, update, and delete operations using V2 API in a realistic CRUD workflow.
// runComprehensiveDocumentOperations runs comprehensive document operations with a specific number of pre-created documents
func runComprehensiveDocumentOperations(b *testing.B, col arangodb.Collection, numDocs int) {
	withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		// Pre-create documents for read/update/delete operations
		b.Logf("Pre-creating %d documents for comprehensive V2 benchmark", numDocs)
		metas := make([]arangodb.CollectionDocumentCreateResponse, numDocs)
		for i := 0; i < numDocs; i++ {
			doc := UserDoc{
				Name: fmt.Sprintf("V2ComprehensiveUser_%d", i),
				Age:  20 + (i % 50),
			}
			meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
			if err != nil {
				b.Fatalf("Failed to create document %d: %s", i, err)
			}
			metas[i] = meta
		}

		b.ResetTimer()

		// Measure create operations
		b.Run("Create", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				doc := UserDoc{
					Name: fmt.Sprintf("V2CreateUser_%d_%d", i, b.N),
					Age:  20 + (i % 50),
				}
				if _, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{}); err != nil {
					b.Errorf("CreateDocument failed: %s", err)
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
				if _, err := col.ReadDocument(ctx, metas[docIndex].Key, &result); err != nil {
					b.Errorf("ReadDocument failed: %s", err)
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
				if _, err := col.UpdateDocument(ctx, metas[docIndex].Key, update); err != nil {
					b.Errorf("UpdateDocument failed: %s", err)
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
					Name: fmt.Sprintf("V2DeleteUser_%d", i),
					Age:  20 + (i % 50),
				}

				// Create the document
				meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
				if err != nil {
					b.Errorf("CreateDocument failed: %s", err)
					continue
				}

				// Delete the document
				if _, err := col.DeleteDocument(ctx, meta.Key); err != nil {
					b.Errorf("DeleteDocument failed: %s", err)
				}
			}
			b.ReportAllocs()
		})
	})
}

// BenchmarkV2ComprehensiveDocumentOperations_1K measures comprehensive document operations with 1000 pre-created documents
func BenchmarkV2ComprehensiveDocumentOperations_1K(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				runComprehensiveDocumentOperations(b, col, 1000)
			})
		})
	})
}

// BenchmarkV2ComprehensiveDocumentOperations_10K measures comprehensive document operations with 10000 pre-created documents
func BenchmarkV2ComprehensiveDocumentOperations_10K(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				runComprehensiveDocumentOperations(b, col, 10000)
			})
		})
	})
}

// BenchmarkV2SingleDocumentInsert measures single document insertion performance
func BenchmarkV2SingleDocumentInsert(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						doc := UserDoc{
							Name: fmt.Sprintf("V2SingleUser_%d", i),
							Age:  20 + (i % 50),
						}
						if _, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{}); err != nil {
							b.Errorf("CreateDocument failed: %s", err)
						}
					}
					b.ReportAllocs()
					b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
				})
			})
		})
	})
}

// runBatchDocumentInsertBenchmark runs batch document insertion benchmark with specified batch size
func runBatchDocumentInsertBenchmark(b *testing.B, col arangodb.Collection, batchSize int) {
	withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			docs := make([]UserDoc, batchSize)
			for j := 0; j < batchSize; j++ {
				docs[j] = UserDoc{
					Name: fmt.Sprintf("V2BatchUser_%d_%d", i, j),
					Age:  20 + (j % 50),
				}
			}
			results, err := col.CreateDocumentsWithOptions(ctx, docs, &arangodb.CollectionDocumentCreateOptions{})
			if err != nil {
				b.Errorf("CreateDocuments failed: %s", err)
				continue
			}
			// Consume all results to ensure proper cleanup
			for {
				_, err := results.Read()
				if err != nil {
					break
				}
			}
		}
		b.ReportAllocs()
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
	})
}

// BenchmarkV2BatchDocumentInsert_10 measures batch document insertion with 10 documents per batch
func BenchmarkV2BatchDocumentInsert_10(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				runBatchDocumentInsertBenchmark(b, col, 10)
			})
		})
	})
}

// BenchmarkV2BatchDocumentInsert_100 measures batch document insertion with 100 documents per batch
func BenchmarkV2BatchDocumentInsert_100(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				runBatchDocumentInsertBenchmark(b, col, 100)
			})
		})
	})
}

// BenchmarkV2BatchDocumentInsert_1000 measures batch document insertion with 1000 documents per batch
func BenchmarkV2BatchDocumentInsert_1000(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				runBatchDocumentInsertBenchmark(b, col, 1000)
			})
		})
	})
}
