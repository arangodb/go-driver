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

	"github.com/stretchr/testify/require"

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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
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
	if found, err := col.DocumentExists(nil, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if !found {
		t.Errorf("DocumentExists returned false for '%s', expected true", meta.Key)
	}
	// Read document
	var readDoc UserDoc
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateDocumentWithKey creates a document with given key and then checks that it exists.
func TestCreateDocumentWithKey(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
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

	// Retry creating the document with same key. This must fail.
	if _, err := col.CreateDocument(nil, doc); !driver.IsConflict(err) {
		t.Fatalf("Expected ConflictError, got %s", describe(err))
	}
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateDocumentReturnNew creates a document and checks the document returned in in ReturnNew.
func TestCreateDocumentReturnNew(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"JanNew",
		1,
	}
	var newDoc UserDoc

	withNewDocCtx := driver.WithReturnNew(ctx, &newDoc)
	meta, err := col.CreateDocument(withNewDocCtx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// NewDoc must equal doc
	if !reflect.DeepEqual(doc, newDoc) {
		t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", doc, newDoc)
	}

	returnNew, exist := driver.HasReturnNew(withNewDocCtx)
	require.True(t, exist, "ReturnNew not set")

	newDoc2, ok := returnNew.(*UserDoc)
	require.True(t, ok, "ReturnNew not set correctly")
	require.Equal(t, newDoc.Age, newDoc2.Age, "ReturnNew not set correctly")
	require.Equal(t, newDoc.Name, newDoc2.Name, "ReturnNew not set correctly")

	// Document must exists now
	var readDoc UserDoc
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
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

// TestCreateDocumentSilent creates a document with WithSilent.
func TestCreateDocumentSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
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
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateDocumentNil creates a document with a nil document.
func TestCreateDocumentNil(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.CreateDocument(nil, nil); !driver.IsInvalidArgument(err) {
		t.Fatalf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateDocumentInWaitForSyncCollection creates a document in a collection with waitForSync enabled,
// and then checks that it exists.
func TestCreateDocumentInWaitForSyncCollection(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "TestCreateDocumentInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Document must exists now
	if found, err := col.DocumentExists(nil, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if !found {
		t.Errorf("DocumentExists returned false for '%s', expected true", meta.Key)
	}
	// Read document
	var readDoc UserDoc
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
