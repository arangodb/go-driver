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

// createDocument creates a document in the given collection, failing the test on error.
func createDocument(ctx context.Context, col driver.Collection, document interface{}, t *testing.T) driver.DocumentMeta {
	meta, err := col.CreateDocument(ctx, document)
	if err != nil {
		t.Fatalf("Failed to create document: %s", describe(err))
	}
	return meta
}

// TestCreateDocument creates a document and then checks that it exists.
func TestCreateDocument(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Document must exists now
	var readDoc UserDoc
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestCreateDocumentWithKey creates a document with given key and then checks that it exists.
func TestCreateDocumentWithKey(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_withKey_test", nil, t)
	doc := UserDocWithKey{
		"jan",
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Key must be given key
	if meta.Key != doc.Key {
		t.Errorf("Expected key to be '%s', got '%s'", doc.Key, meta.Key)
	}
	// Document must exists now
	var readDoc UserDocWithKey
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestCreateDocumentReturnNew creates a document and checks the document returned in in ReturnNew.
func TestCreateDocumentReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"JanNew",
		1,
	}
	var newDoc UserDoc
	meta, err := col.CreateDocument(driver.WithReturnNew(ctx, &newDoc), doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// NewDoc must equal doc
	if !reflect.DeepEqual(doc, newDoc) {
		t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", doc, newDoc)
	}
	// Document must exists now
	var readDoc UserDoc
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestCreateDocumentSilent creates a document with WithSilent.
func TestCreateDocumentSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Sjjjj",
		1,
	}
	if meta, err := col.CreateDocument(driver.WithSilent(ctx), doc); err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestCreateDocumentNil creates a document with a nil document.
func TestCreateDocumentNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.CreateDocument(nil, nil); !driver.IsInvalidArgument(err) {
		t.Fatalf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
