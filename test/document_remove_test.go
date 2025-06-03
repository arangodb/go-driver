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

// TestReplaceDocument creates a document, remove it and then checks the removal has succeeded.
func TestRemoveDocument(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Piere",
		23,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	if _, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
	// Document must exists now
	if found, err := col.DocumentExists(ctx, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if found {
		t.Errorf("DocumentExists returned true for '%s', expected false", meta.Key)
	}
}

// TestRemoveDocumentReturnOld creates a document, removes it checks the ReturnOld value.
func TestRemoveDocumentReturnOld(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Tim",
		27,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	var old UserDoc
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentSilent creates a document, removes it with Silent() and then checks the meta is indeed empty.
func TestRemoveDocumentSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Angela",
		91,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmeta, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	} else if rmeta.Key != "" {
		t.Errorf("Expected empty meta, got %v", rmeta)
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentRevision creates a document, removes it with an incorrect revision.
func TestRemoveDocumentRevision(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"DryLake",
		91,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Replace the document to get another revision
	replacement := Book{
		Title: "Jungle book",
	}
	meta2, err := col.ReplaceDocument(ctx, meta.Key, replacement)
	if err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}

	// Try to remove document with initial revision (must fail)
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	if _, err := col.RemoveDocument(initialRevCtx, meta.Key); !driver.IsPreconditionFailed(err) {
		t.Fatalf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Try to remove document with correct revision (must succeed)
	replacedRevCtx := driver.WithRevision(ctx, meta2.Rev)
	if _, err := col.RemoveDocument(replacedRevCtx, meta.Key); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentKeyEmpty removes a document it with an empty key.
func TestRemoveDocumentKeyEmpty(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.RemoveDocument(nil, ""); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentInWaitForSyncCollection creates a document in a collection with waitForSync enabled,
// removes it and then checks the removal has succeeded.
func TestRemoveDocumentInWaitForSyncCollection(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "TestRemoveDocumentInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	doc := UserDoc{
		"Piere",
		23,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	if _, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
	// Document must exists now
	if found, err := col.DocumentExists(ctx, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if found {
		t.Errorf("DocumentExists returned true for '%s', expected false", meta.Key)
	}
}
