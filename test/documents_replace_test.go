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
	"strings"
	"testing"

	"github.com/arangodb/go-driver"
)

// TestReplaceDocuments creates documents, replaces them and then checks the replacements have succeeded.
func TestReplaceDocuments(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
		{
			"Pioter",
			45,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replacement docs
	replacements := []Account{
		{
			ID:   "foo",
			User: &UserDoc{},
		},
		{
			ID:   "foo2",
			User: &UserDoc{},
		},
	}
	if _, _, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Read replaced documents
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if !reflect.DeepEqual(replacements[i], readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, replacements[i], readDoc)
		}
	}
}

// TestReplaceDocumentsReturnOld creates documents, replaces them checks the ReturnOld values.
func TestReplaceDocumentsReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Tim",
			27,
		},
		{
			"George",
			32,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []Book{
		{
			Title: "Golang 1.8",
		},
		{
			Title: "Dart 1.0",
		},
	}
	oldDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Check old document
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestReplaceDocumentsReturnNew creates documents, replaces them checks the ReturnNew values.
func TestReplaceDocumentsReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Tim",
			27,
		},
		{
			"Anna",
			27,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []Book{
		{
			Title: "Golang 1.8",
		},
		{
			Title: "C++ made easy",
		},
	}
	newDocs := make([]Book, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Check new documents
	for i, replacement := range replacements {
		expected := replacement
		if !reflect.DeepEqual(expected, newDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, expected, newDocs[i])
		}
	}
}

// TestReplaceDocumentsSilent creates documents, replaces them with Silent() and then checks the meta is indeed empty.
func TestReplaceDocumentsSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Angela",
			91,
		},
		{
			"Fiona",
			12,
		},
		{
			"Roos",
			54,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []Book{
		{
			Title: "Jungle book",
		},
		{
			Title: "Database book",
		},
		{
			Title: "Raft book",
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	} else {
		if len(errs) > 0 {
			t.Errorf("Expected 0 errors, got %d", len(errs))
		}
		if len(metas) > 0 {
			t.Errorf("Expected 0 metas, got %d", len(metas))
		}
	}
}

// TestReplaceDocumentsRevision creates documents, replaces then with a specific (correct) revisions.
// Then it attempts replacements with incorrect revisions which must fail.
func TestReplaceDocumentsRevision(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Revision",
			33,
		},
		{
			"Other revision",
			33,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace documents with correct revisions
	replacements := []Book{
		{
			Title: "Jungle book",
		},
		{
			Title: "Portable book",
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var replacedRevCtx context.Context
	if metas2, errs, err := col.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	} else {
		replacedRevCtx = driver.WithRevisions(ctx, metas2.Revs())
		if strings.Join(metas2.Revs(), ",") == strings.Join(metas.Revs(), ",") {
			t.Errorf("Expected revisions to change, got initial revisions '%s', replaced revisions '%s'", strings.Join(metas.Revs(), ","), strings.Join(metas2.Revs(), ","))
		}
	}

	// Replace documents with incorrect revision
	replacements[0].Title = "Wrong deal 1"
	replacements[1].Title = "Wrong deal 2"
	if _, errs, err := col.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for i, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError at %d, got %s", i, describe(err))
			}
		}
	}

	// Replace document once more with correct revision
	replacements[0].Title = "Good deal 1"
	replacements[1].Title = "Good deal 2"
	if _, errs, err := col.ReplaceDocuments(replacedRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
}

// TestReplaceDocumentsKeyEmpty replaces a document it with an empty key.
func TestReplaceDocumentsKeyEmpty(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	// Replacement document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, _, err := col.ReplaceDocuments(nil, []string{""}, replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentsUpdateNil replaces a document it with a nil update.
func TestReplaceDocumentsUpdateNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	if _, _, err := col.ReplaceDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentsUpdateLenDiff replacements documents with a different number of documents, keys.
func TestReplaceDocumentsUpdateLenDiff(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	replacements := []map[string]interface{}{
		{
			"name": "name1",
		},
		{
			"name": "name2",
		},
	}
	if _, _, err := col.ReplaceDocuments(nil, []string{"only1"}, replacements); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentsInWaitForSyncCollection creates documents into a collection with waitForSync enabled,
// replaces them and then checks the replacements have succeeded.
func TestReplaceDocumentsInWaitForSyncCollection(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "TestReplaceDocumentsInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
		{
			"Pioter",
			45,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replacement docs
	replacements := []Account{
		{
			ID:   "foo",
			User: &UserDoc{},
		},
		{
			ID:   "foo2",
			User: &UserDoc{},
		},
	}
	if _, _, err := col.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Read replaced documents
	for i, meta := range metas {
		var readDoc Account
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if !reflect.DeepEqual(replacements[i], readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, replacements[i], readDoc)
		}
	}
}
