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

// TestUpdateVertex creates a document, updates it and then checks the update has succeeded.
func TestUpdateVertex(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "user", t)

	doc := UserDoc{
		Name: "Francis",
		Age:  51,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"age": 55,
	}
	if _, err := vc.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc UserDoc
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	doc.Age = 55
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestUpdateVertexReturnOld creates a document, updates it checks the ReturnOld value.
func TestUpdateVertexReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_returnOld_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "Hello",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"Title": "Goodbye",
	}
	var old Book
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := vc.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestUpdateVertexReturnNew creates a document, updates it checks the ReturnNew value.
func TestUpdateVertexReturnNew(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_returnNew_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "person", t)

	doc := UserDoc{
		Name: "Bertha",
		Age:  31,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"age": 45,
	}
	var newDoc UserDoc
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := vc.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := doc
	expected.Age = 45
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestUpdateVertexKeepNullTrue creates a document, updates it with KeepNull(true) and then checks the update has succeeded.
func TestUpdateVertexKeepNullTrue(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	conn := c.Connection()
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_keepNullTrue_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "accounts", t)

	doc := Account{
		ID: "store1",
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"id":   "foo",
		"user": nil,
	}
	if _, err := vc.UpdateDocument(driver.WithKeepNull(ctx, true), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc map[string]interface{}
	var rawResponse []byte
	ctx = driver.WithRawResponse(ctx, &rawResponse)
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	// We parse to this type of map, since unmarshalling nil values to a map of type map[string]interface{}
	// will cause the entry to be deleted.
	var jsonMap map[string]*driver.RawObject
	if err := conn.Unmarshal(rawResponse, &jsonMap); err != nil {
		t.Fatalf("Failed to parse raw response: %s", describe(err))
	}
	// Get "vertex" field and unmarshal it
	if raw, found := jsonMap["vertex"]; !found {
		t.Errorf("Expected vertex to be found but got not found")
	} else {
		jsonMap = nil
		if err := conn.Unmarshal(*raw, &jsonMap); err != nil {
			t.Fatalf("Failed to parse raw vertex object: %s", describe(err))
		}
		if raw, found := jsonMap["user"]; !found {
			t.Errorf("Expected user to be found but got not found")
		} else if raw != nil {
			t.Errorf("Expected user to be found and nil, got %s", string(*raw))
		}
	}
}

// TestUpdateVertexKeepNullFalse creates a document, updates it with KeepNull(false) and then checks the update has succeeded.
func TestUpdateVertexKeepNullFalse(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_keepNullFalse_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "accounts", t)

	doc := Account{
		ID: "Nullify",
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"id":   "another",
		"user": nil,
	}
	if _, err := vc.UpdateDocument(driver.WithKeepNull(ctx, false), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	readDoc := doc
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if readDoc.User == nil {
		t.Errorf("Expected user to be untouched, got %v", readDoc.User)
	}
}

// TestUpdateVertexSilent creates a document, updates it with Silent() and then checks the meta is indeed empty.
func TestUpdateVertexSilent(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "moments", t)

	doc := Book{
		Title: "Enjoy the silence",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"Title": "No more noise",
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := vc.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestUpdateVertexRevision creates a document, updates it with a specific (correct) revision.
// Then it attempts an update with an incorrect revision which must fail.
func TestUpdateVertexRevision(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_revision_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "Rev1",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Update document with correct revision
	update := map[string]interface{}{
		"Title": "Rev2",
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var updatedRevCtx context.Context
	if meta2, err := vc.UpdateDocument(initialRevCtx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else {
		updatedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Update document with incorrect revision
	update["Title"] = "Rev3"
	if _, err := vc.UpdateDocument(initialRevCtx, meta.Key, update); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Update document  once more with correct revision
	update["Title"] = "Rev4"
	if _, err := vc.UpdateDocument(updatedRevCtx, meta.Key, update); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateVertexKeyEmpty updates a document it with an empty key.
func TestUpdateVertexKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_keyEmpty_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "tests", t)

	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := vc.UpdateDocument(nil, "", update); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateVertexUpdateNil updates a document it with a nil update.
func TestUpdateVertexUpdateNil(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	g := ensureGraph(ctx, db, "update_vertex_updateNil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "errors", t)

	if _, err := vc.UpdateDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
