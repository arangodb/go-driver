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

// TestUpdateDocument1 creates a document, updates it and then checks the update has succeeded.
func TestUpdateDocument1(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Piere",
		23,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := col.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc UserDoc
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	doc.Name = "Updated"
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestUpdateDocumentReturnOld creates a document, updates it checks the ReturnOld value.
func TestUpdateDocumentReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Tim",
		27,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	var old UserDoc
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := col.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestUpdateDocumentReturnNew creates a document, updates it checks the ReturnNew value.
func TestUpdateDocumentReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Tim",
		27,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	var newDoc UserDoc
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := col.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := doc
	expected.Name = "Updated"
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestUpdateDocumentKeepNullTrue creates a document, updates it with KeepNull(true) and then checks the update has succeeded.
func TestUpdateDocumentKeepNullTrue(t *testing.T) {
	ctx := context.Background()
	var conn driver.Connection
	c := createClientFromEnv(t, true, &conn)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := Account{
		ID: "1234",
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"id":   "5678",
		"user": nil,
	}
	if _, err := col.UpdateDocument(driver.WithKeepNull(ctx, true), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc map[string]interface{}
	var rawResponse []byte
	ctx = driver.WithRawResponse(ctx, &rawResponse)
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	// We parse to this type of map, since unmarshalling nil values to a map of type map[string]interface{}
	// will cause the entry to be deleted.
	var jsonMap map[string]*driver.RawObject
	if err := conn.Unmarshal(rawResponse, &jsonMap); err != nil {
		t.Fatalf("Failed to parse raw response: %s", describe(err))
	}
	if raw, found := jsonMap["user"]; !found {
		t.Errorf("Expected user to be found but got not found")
	} else if raw != nil {
		t.Errorf("Expected user to be found and nil, got %s", string(*raw))
	}
}

// TestUpdateDocumentKeepNullFalse creates a document, updates it with KeepNull(false) and then checks the update has succeeded.
func TestUpdateDocumentKeepNullFalse(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := Account{
		ID: "1234",
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"id":   "5678",
		"user": nil,
	}
	if _, err := col.UpdateDocument(driver.WithKeepNull(ctx, false), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	readDoc := doc
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if readDoc.User == nil {
		t.Errorf("Expected user to be untouched, got %v", readDoc.User)
	}
}

// TestUpdateDocumentSilent creates a document, updates it with Silent() and then checks the meta is indeed empty.
func TestUpdateDocumentSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Angela",
		91,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"age": "61",
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := col.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestUpdateDocumentRevision creates a document, updates it with a specific (correct) revision.
// Then it attempts an update with an incorrect revision which must fail.
func TestUpdateDocumentRevision(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Revision",
		33,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Update document with correct revision
	update := map[string]interface{}{
		"age": 34,
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var updatedRevCtx context.Context
	if meta2, err := col.UpdateDocument(initialRevCtx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else {
		updatedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Update document with incorrect revision
	update["age"] = 35
	if _, err := col.UpdateDocument(initialRevCtx, meta.Key, update); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Update document  once more with correct revision
	update["age"] = 36
	if _, err := col.UpdateDocument(updatedRevCtx, meta.Key, update); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateDocumentKeyEmpty updates a document it with an empty key.
func TestUpdateDocumentKeyEmpty(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := col.UpdateDocument(nil, "", update); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateDocumentUpdateNil updates a document it with a nil update.
func TestUpdateDocumentUpdateNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.UpdateDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
