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

	"github.com/arangodb/go-driver"
)

// TestReplaceDocuments creates documents, removes them and then checks the removal has succeeded.
func TestRemoveDocuments(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	if _, _, err := col.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	}
	// Should not longer exist
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Fatalf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
}

// TestRemoveDocumentsReturnOld creates documents, removes them checks the ReturnOld value.
func TestRemoveDocumentsReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Tim",
			27,
		},
		{
			"Tom",
			27,
		},
		{
			"Tam",
			27,
		},
		{
			"Tum",
			27,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	oldDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := col.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	}
	// Check old documents
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
		// Should not longer exist
		var readDoc Account
		if _, err := col.ReadDocument(ctx, metas[i].Key, &readDoc); !driver.IsNotFound(err) {
			t.Fatalf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
}

// TestRemoveDocumentsSilent creates documents, removes them with Silent() and then checks the meta is indeed empty.
func TestRemoveDocumentsSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Angela",
			91,
		},
		{
			"Tommy",
			19,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmetas, rerrs, err := col.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	} else {
		if len(rmetas) > 0 {
			t.Errorf("Expected empty metas, got %d", len(rmetas))
		}
		if len(rerrs) > 0 {
			t.Errorf("Expected empty errors, got %d", len(rerrs))
		}
	}
	// Should not longer exist
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
}

// TestRemoveDocumentsRevision creates documents, removes them with an incorrect revisions.
func TestRemoveDocumentsRevision(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"DryLake",
			91,
		},
		{
			"DryBed",
			91,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace the documents to get another revision
	replacements := []Book{
		{
			Title: "Jungle book",
		},
		{
			Title: "Another book",
		},
	}
	metas2, errs2, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements)
	if err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	} else if err := errs2.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Try to remove documents with initial revision (must fail)
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	if _, errs, err := col.RemoveDocuments(initialRevCtx, metas.Keys()); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	} else {
		for i, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError at %d, got %s", i, describe(err))
			}
		}
	}

	// Try to remove documents with correct revision (must succeed)
	replacedRevCtx := driver.WithRevisions(ctx, metas2.Revs())
	if _, errs, err := col.RemoveDocuments(replacedRevCtx, metas.Keys()); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Should not longer exist
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
}

// TestRemoveDocumentsKeyEmpty removes a document it with an empty key.
func TestRemoveDocumentsKeyEmpty(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "documents_test", nil, t)
	if _, _, err := col.RemoveDocuments(nil, []string{""}); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestRemoveDocumentsInWaitForSyncCollection creates documents in a collection with waitForSync enabled,
// removes them and then checks the removal has succeeded.
func TestRemoveDocumentsInWaitForSyncCollection(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "TestRemoveDocumentsInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	if _, _, err := col.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	}
	// Should not longer exist
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Fatalf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
}
