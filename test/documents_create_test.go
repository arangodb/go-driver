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

// TestCreateDocuments creates a document and then checks that it exists.
func TestCreateDocuments(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	docs := []UserDoc{
		UserDoc{
			"Jan",
			40,
		},
		UserDoc{
			"Foo",
			41,
		},
		UserDoc{
			"Frank",
			42,
		},
	}
	metas, errs, err := col.CreateDocuments(nil, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		// Read back using ReadDocuments
		keys := make([]string, len(docs))
		for i, m := range metas {
			keys[i] = m.Key
		}
		readDocs := make([]UserDoc, len(docs))
		if _, _, err := col.ReadDocuments(nil, keys, readDocs); err != nil {
			t.Fatalf("Failed to read documents: %s", describe(err))
		}
		for i, d := range readDocs {
			if !reflect.DeepEqual(docs[i], d) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], d)
			}
		}
		// Read back using individual ReadDocument requests
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}

			// Document must exists now
			var readDoc UserDoc
			if _, err := col.ReadDocument(nil, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateDocumentsReturnNew creates a document and checks the document returned in in ReturnNew.
func TestCreateDocumentsReturnNew(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		UserDoc{
			"Sjjjj",
			1,
		},
		UserDoc{
			"Mies",
			2,
		},
	}
	newDocs := make([]UserDoc, len(docs))
	metas, errs, err := col.CreateDocuments(driver.WithReturnNew(ctx, newDocs), docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}
			// NewDoc must equal doc
			if !reflect.DeepEqual(docs[i], newDocs[i]) {
				t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", docs[i], newDocs[i])
			}
			// Document must exists now
			var readDoc UserDoc
			if _, err := col.ReadDocument(ctx, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateDocumentsSilent creates a document with WithSilent.
func TestCreateDocumentsSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		UserDoc{
			"Sjjjj",
			1,
		},
		UserDoc{
			"Mies",
			2,
		},
	}
	if metas, errs, err := col.CreateDocuments(driver.WithSilent(ctx), docs); err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else {
		if len(metas) != 0 {
			t.Errorf("Expected 0 metas, got %d", len(metas))
		}
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(errs))
		}
	}
}

// TestCreateDocumentsNil creates multiple documents with a nil documents input.
func TestCreateDocumentsNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	if _, _, err := col.CreateDocuments(nil, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestCreateDocumentsNonSlice creates multiple documents with a non-slice documents input.
func TestCreateDocumentsNonSlice(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "documents_test", nil, t)
	var obj UserDoc
	if _, _, err := col.CreateDocuments(nil, &obj); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	var m map[string]interface{}
	if _, _, err := col.CreateDocuments(nil, &m); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestCreateDocumentsInWaitForSyncCollection creates a few documents in a collection with waitForSync enabled and then checks that it exists.
func TestCreateDocumentsInWaitForSyncCollection(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "TestCreateDocumentsInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	docs := []UserDoc{
		UserDoc{
			"Jan",
			40,
		},
		UserDoc{
			"Foo",
			41,
		},
		UserDoc{
			"Frank",
			42,
		},
	}
	metas, errs, err := col.CreateDocuments(nil, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}

			// Document must exists now
			var readDoc UserDoc
			if _, err := col.ReadDocument(nil, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}
