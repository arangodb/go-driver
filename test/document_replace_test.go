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

// TestReplaceDocument creates a document, replaces it and then checks the replacement has succeeded.
func TestReplaceDocument(t *testing.T) {
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
	// Replacement doc
	replacement := Account{
		ID:   "foo",
		User: &UserDoc{},
	}
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Read replaces document
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(replacement, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", replacement, readDoc)
	}
}

// TestReplaceDocumentReturnOld creates a document, replaces it checks the ReturnOld value.
func TestReplaceDocumentReturnOld(t *testing.T) {
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
	// Replace document
	replacement := Book{
		Title: "Golang 1.8",
	}
	var old UserDoc
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestReplaceDocumentReturnNew creates a document, replaces it checks the ReturnNew value.
func TestReplaceDocumentReturnNew(t *testing.T) {
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
	replacement := Book{
		Title: "Golang 1.8",
	}
	var newDoc Book
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := replacement
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestReplaceDocumentSilent creates a document, replaces it with Silent() and then checks the meta is indeed empty.
func TestReplaceDocumentSilent(t *testing.T) {
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
	replacement := Book{
		Title: "Jungle book",
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestReplaceDocumentRevision creates a document, replaces it with a specific (correct) revision.
// Then it attempts a replacement with an incorrect revision which must fail.
func TestReplaceDocumentRevision(t *testing.T) {
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

	// Replace document with correct revision
	replacement := Book{
		Title: "Jungle book",
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var replacedRevCtx context.Context
	if meta2, err := col.ReplaceDocument(initialRevCtx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else {
		replacedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', replaced revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Replace document with incorrect revision
	replacement.Title = "Wrong deal"
	if _, err := col.ReplaceDocument(initialRevCtx, meta.Key, replacement); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Replace document once more with correct revision
	replacement.Title = "Good deal"
	if _, err := col.ReplaceDocument(replacedRevCtx, meta.Key, replacement); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestReplaceDocumentKeyEmpty replaces a document it with an empty key.
func TestReplaceDocumentKeyEmpty(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	// Update document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := col.ReplaceDocument(nil, "", replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentUpdateNil replaces a document it with a nil update.
func TestReplaceDocumentUpdateNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.ReplaceDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
