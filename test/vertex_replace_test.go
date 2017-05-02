//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package test

import (
	"context"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestReplaceVertex creates a document, replaces it and then checks the replacement has succeeded.
func TestReplaceVertex(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "friend", t)

	doc := UserDoc{
		Name: "Bunny",
		Age:  82,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replacement doc
	replacement := Book{
		Title: "Old is nice",
	}
	if _, err := vc.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Read replaces document
	var readDoc Book
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(replacement, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", replacement, readDoc)
	}
}

// TestReplaceVertexReturnOld creates a document, replaces it checks the ReturnOld value.
func TestReplaceVertexReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.3", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_returnOld_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "Who goes there",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replace document
	replacement := UserDoc{
		Name: "Ghost",
		Age:  1011,
	}
	var old Book
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := vc.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestReplaceVertexReturnNew creates a document, replaces it checks the ReturnNew value.
func TestReplaceVertexReturnNew(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.3", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_returnNew_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	doc := UserDoc{
		Name: "Mark",
		Age:  51,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := Book{
		Title: "How to win elections",
	}
	var newDoc Book
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := vc.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := replacement
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestReplaceVertexSilent creates a document, replaces it with Silent() and then checks the meta is indeed empty.
func TestReplaceVertexSilent(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_returnNew_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "person", t)

	doc := UserDoc{
		Name: "Janna",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := UserDoc{
		Name: "Boeda",
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := vc.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestReplaceVertexRevision creates a document, replaces it with a specific (correct) revision.
// Then it attempts a replacement with an incorrect revision which must fail.
func TestReplaceVertexRevision(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_revision_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "France in spring",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Replace document with correct revision
	replacement := Book{
		Title: "France in winter",
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var replacedRevCtx context.Context
	if meta2, err := vc.ReplaceDocument(initialRevCtx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else {
		replacedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', replaced revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Replace document with incorrect revision
	replacement.Title = "France in fall"
	if _, err := vc.ReplaceDocument(initialRevCtx, meta.Key, replacement); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Replace document once more with correct revision
	replacement.Title = "France in autumn"
	if _, err := vc.ReplaceDocument(replacedRevCtx, meta.Key, replacement); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestReplaceVertexKeyEmpty replaces a document it with an empty key.
func TestReplaceVertexKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_keyEmpty_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "names", t)

	// Replace document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := vc.ReplaceDocument(nil, "", replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceVertexUpdateNil replaces a document it with a nil update.
func TestReplaceVertexUpdateNil(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertex_updateNil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "names", t)

	if _, err := vc.ReplaceDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
