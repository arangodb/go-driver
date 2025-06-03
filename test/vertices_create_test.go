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
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestCreateVertices creates documents and then checks that it exists.
func TestCreateVertices(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "create_vertices_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		{
			Title: "Book1",
		},
		{
			Title: "Book2",
		},
		{
			Title: "Book3",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		// Read back using ReadDocuments
		keys := make([]string, len(docs))
		for i, m := range metas {
			keys[i] = m.Key
		}
		readDocs := make([]Book, len(docs))
		if _, _, err := vc.ReadDocuments(nil, keys, readDocs); err != nil {
			t.Fatalf("Failed to read documents: %s", describe(err))
		}
		for i, d := range readDocs {
			if !reflect.DeepEqual(docs[i], d) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], d)
			}
		}
		// Read back using individual ReadDocument requests
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}

			// Document must exists now
			var readDoc Book
			if _, err := vc.ReadDocument(nil, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateVerticesReturnNew creates documents and checks the document returned in in ReturnNew.
func TestCreateVerticesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "create_vertices_returnNew_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		{
			Title: "Book1",
		},
		{
			Title: "Book2",
		},
		{
			Title: "Book3",
		},
	}
	newDocs := make([]Book, len(docs))
	metas, errs, err := vc.CreateDocuments(driver.WithReturnNew(ctx, newDocs), docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}
			// NewDoc must equal doc
			if !reflect.DeepEqual(docs[i], newDocs[i]) {
				t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", docs[i], newDocs[i])
			}
			// Document must exists now
			var readDoc Book
			if _, err := vc.ReadDocument(ctx, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateVerticesSilent creates documents with WithSilent.
func TestCreateVerticesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "create_vertices_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	docs := []UserDoc{
		{
			Name: "Jan",
			Age:  12,
		},
		{
			Name: "Piet",
			Age:  2,
		},
	}
	if metas, errs, err := vc.CreateDocuments(driver.WithSilent(ctx), docs); err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else {
		if len(metas) != 0 {
			t.Errorf("Expected 0 metas, got %d", len(metas))
		}
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(errs))
		}
	}
}

// TestCreateVerticesNil creates multiple documents with a nil documents input.
func TestCreateVerticesNil(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "create_vertices_nil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "rivers", t)
	if _, _, err := vc.CreateDocuments(nil, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestCreateVerticesNonSlice creates multiple documents with a non-slice documents input.
func TestCreateVerticesNonSlice(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "create_vertices_nonSlice_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "failures", t)

	var obj UserDoc
	if _, _, err := vc.CreateDocuments(nil, &obj); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	var m map[string]interface{}
	if _, _, err := vc.CreateDocuments(nil, &m); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
