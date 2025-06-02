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

// TestCreateVertex creates an vertex and then checks that it exists.
func TestCreateVertex(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "create_vertex_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	book := Book{Title: "Graphs are cool"}
	meta, err := vc.CreateDocument(ctx, book)
	if err != nil {
		t.Fatalf("Failed to create new vertex: %s", describe(err))
	}
	// Document must exists now
	if found, err := vc.DocumentExists(nil, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if !found {
		t.Errorf("DocumentExists returned false for '%s', expected true", meta.Key)
	}

	// Read document
	var readDoc Book
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read vertex '%s': %s", meta.Key, describe(err))
	} else {
		if !reflect.DeepEqual(book, readDoc) {
			t.Errorf("Got invalid document. Expected '%+v', got '%+v'", book, readDoc)
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateVertexReturnNew creates a document and checks the document returned in in ReturnNew.
func TestCreateVertexReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "create_vertex_return_new_est", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	doc := UserDoc{
		Name: "Fern",
		Age:  31,
	}
	var newDoc UserDoc
	meta, err := vc.CreateDocument(driver.WithReturnNew(ctx, &newDoc), doc)
	if err != nil {
		t.Fatalf("Failed to create new vertex: %s", describe(err))
	}
	// NewDoc must equal doc
	if !reflect.DeepEqual(doc, newDoc) {
		t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", doc, newDoc)
	}
	// Document must exists now
	var readDoc UserDoc
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateVertexSilent creates a document with WithSilent.
func TestCreateVertexSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "create_vertex_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	doc := UserDoc{
		Name: "Fern",
		Age:  31,
	}
	if meta, err := vc.CreateDocument(driver.WithSilent(ctx), doc); err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateVertexNil creates a document with a nil document.
func TestCreateVertexNil(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "create_vertex_nil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	if _, err := vc.CreateDocument(nil, nil); !driver.IsInvalidArgument(err) {
		t.Fatalf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
