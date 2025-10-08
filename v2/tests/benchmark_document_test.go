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
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// BenchmarkV2CreateDocument measures the CreateDocument operation for a simple document.
func BenchmarkV2CreateDocument(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						doc := UserDoc{
							Name: "Jan",
							Age:  40 + i,
						}
						_, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
						if err != nil {
							b.Fatalf("Failed to create new document: %s", err)
						}
					}
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2CreateDocumentParallel measures parallel CreateDocument operations for a simple document.
func BenchmarkV2CreateDocumentParallel(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					// Use lower parallelism to avoid HTTP/2 "max concurrent streams exceeded" error
					// HTTP/2 has stricter limits on concurrent streams compared to HTTP/1.1
					b.SetParallelism(5)
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							doc := UserDoc{
								Name: "Jan",
								Age:  40,
							}
							// Use CreateDocumentWithOptions for better HTTP/2 compatibility
							_, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
							if err != nil {
								b.Fatalf("Failed to create new document: %s", err)
							}
						}
					})
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2ReadDocument measures the ReadDocument operation for a simple document.
func BenchmarkV2ReadDocument(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					doc := UserDoc{
						Name: "Jan",
						Age:  40,
					}
					meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
					if err != nil {
						b.Fatalf("Failed to create new document: %s", err)
					}

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						var result UserDoc
						_, err := col.ReadDocument(ctx, meta.Key, &result)
						if err != nil {
							b.Errorf("Failed to read document: %s", err)
						}
					}
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2ReadDocumentParallel measures parallel ReadDocument operations for a simple document.
func BenchmarkV2ReadDocumentParallel(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					doc := UserDoc{
						Name: "Jan",
						Age:  40,
					}
					meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
					if err != nil {
						b.Fatalf("Failed to create new document: %s", err)
					}

					// Use lower parallelism to avoid HTTP/2 "max concurrent streams exceeded" error
					b.SetParallelism(5)
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							var result UserDoc
							_, err := col.ReadDocument(ctx, meta.Key, &result)
							if err != nil {
								b.Errorf("Failed to read document: %s", err)
							}
						}
					})
					b.ReportAllocs()
				})
			})
		})
	})
}

// BenchmarkV2RemoveDocument measures the RemoveDocument operation for a simple document.
func BenchmarkV2RemoveDocument(b *testing.B) {
	WrapB(b, func(b *testing.B, client arangodb.Client) {
		WithDatabase(b, client, nil, func(db arangodb.Database) {
			WithCollectionV2(b, db, nil, func(col arangodb.Collection) {
				withContextT(b, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						// Create document (we don't measure that)
						b.StopTimer()
						doc := UserDoc{
							Name: "Jan",
							Age:  40 + i,
						}
						meta, err := col.CreateDocumentWithOptions(ctx, doc, &arangodb.CollectionDocumentCreateOptions{})
						if err != nil {
							b.Fatalf("Failed to create new document: %s", err)
						}

						// Now do the real test
						b.StartTimer()
						_, err = col.DeleteDocument(ctx, meta.Key)
						if err != nil {
							b.Errorf("Failed to remove document: %s", err)
						}
					}
					b.ReportAllocs()
				})
			})
		})
	})
}
