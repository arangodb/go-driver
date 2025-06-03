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

import "testing"

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
}
