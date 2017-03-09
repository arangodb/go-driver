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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestImportDocumentsWithKeys imports documents and then checks that it exists.
func TestImportDocumentsWithKeys(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_withKeys_test", nil, t)
	docs := []UserDocWithKey{
		UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		UserDocWithKey{
			"foo",
			"Foo",
			41,
		},
		UserDocWithKey{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
	}
}

// TestImportDocumentsWithoutKeys imports documents and then checks that it exists.
func TestImportDocumentsWithoutKeys(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_withoutKeys_test", nil, t)
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

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
	}
}

// TestImportDocumentsEmptyEntries imports documents and then checks that it exists.
func TestImportDocumentsEmptyEntries(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_emptyEntries_test", nil, t)
	docs := []*UserDocWithKey{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		&UserDocWithKey{
			"foo",
			"Foo",
			41,
		},
		nil,
		&UserDocWithKey{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs))-1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs)-1, stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 1 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 1, stats.Empty, string(raw))
		}
	}
}

// TestImportDocumentsInvalidEntries imports documents and then checks that it exists.
func TestImportDocumentsInvalidEntries(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_invalidEntries_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		[]string{"array", "is", "invalid"},
		&UserDocWithKey{
			"foo",
			"Foo",
			41,
		},
		"string is not valid",
		nil,
		&UserDocWithKey{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs))-3 {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs)-3, stats.Created, string(raw))
		}
		if stats.Errors != 2 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 2, stats.Errors, string(raw))
		}
		if stats.Empty != 1 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 1, stats.Empty, string(raw))
		}
	}
}

// TestImportDocumentsDuplicateEntries imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntries(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_duplicateEntries_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != 1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, string(raw))
		}
		if stats.Errors != 1 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 1, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
		if stats.Updated != 0 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 0, stats.Updated, string(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, string(raw))
		}
	}
}

// TestImportDocumentsDuplicateEntriesComplete imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntriesComplete(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_duplicateEntriesComplete_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	if _, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
		Complete: true,
	}); !driver.IsConflict(err) {
		t.Errorf("Expected ConflictError, got %s", describe(err))
	}
}

// TestImportDocumentsDuplicateEntriesUpdate imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntriesUpdate(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_duplicateEntriesUpdate_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		map[string]interface{}{
			"_key": "jan",
			"name": "Jan2",
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
		OnDuplicate: driver.ImportOnDuplicateUpdate,
	})
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != 1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
		if stats.Updated != 1 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 1, stats.Updated, string(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, string(raw))
		}

		var user UserDocWithKey
		if _, err := col.ReadDocument(nil, "jan", &user); err != nil {
			t.Errorf("ReadDocument failed: %s", describe(err))
		} else {
			if user.Name != "Jan2" {
				t.Errorf("Expected Name to be 'Jan2', got '%s'", user.Name)
			}
			if user.Age != 40 {
				t.Errorf("Expected Age to be 40, got %d", user.Age)
			}
		}
	}
}

// TestImportDocumentsDuplicateEntriesReplace imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntriesReplace(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_duplicateEntriesReplace_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		map[string]interface{}{
			"_key": "jan",
			"name": "Jan2",
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
		OnDuplicate: driver.ImportOnDuplicateReplace,
	})
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != 1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
		if stats.Updated != 1 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 1, stats.Updated, string(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, string(raw))
		}

		var user UserDocWithKey
		if _, err := col.ReadDocument(nil, "jan", &user); err != nil {
			t.Errorf("ReadDocument failed: %s", describe(err))
		} else {
			if user.Name != "Jan2" {
				t.Errorf("Expected Name to be 'Jan2', got '%s'", user.Name)
			}
			if user.Age != 0 {
				t.Errorf("Expected Age to be 0, got %d", user.Age)
			}
		}
	}
}

// TestImportDocumentsDuplicateEntriesIgnore imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntriesIgnore(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_duplicateEntriesIgnore_test", nil, t)
	docs := []interface{}{
		&UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		map[string]interface{}{
			"_key": "jan",
			"name": "Jan2",
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
		OnDuplicate: driver.ImportOnDuplicateIgnore,
	})
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != 1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, string(raw))
		}
		if stats.Updated != 0 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 0, stats.Updated, string(raw))
		}
		if stats.Ignored != 1 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 1, stats.Ignored, string(raw))
		}

		var user UserDocWithKey
		if _, err := col.ReadDocument(nil, "jan", &user); err != nil {
			t.Errorf("ReadDocument failed: %s", describe(err))
		} else {
			if user.Name != "Jan" {
				t.Errorf("Expected Name to be 'Jan', got '%s'", user.Name)
			}
			if user.Age != 40 {
				t.Errorf("Expected Age to be 40, got %d", user.Age)
			}
		}
	}
}
