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
	"testing"

	"github.com/arangodb/go-driver"
)

// TestImportDocumentsWithKeys imports documents and then checks that it exists.
func TestImportDocumentsWithKeys(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_withKeys_test", nil, t)
	docs := []UserDocWithKey{
		{
			"jan",
			"Jan",
			40,
		},
		{
			"foo",
			"Foo",
			41,
		},
		{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s %#v", describe(err), err)
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
	}
}

// TestImportDocumentsWithoutKeys imports documents and then checks that it exists.
func TestImportDocumentsWithoutKeys(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_withoutKeys_test", nil, t)
	docs := []UserDoc{
		{
			"Jan",
			40,
		},
		{
			"Foo",
			41,
		},
		{
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
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
	}
}

// TestImportDocumentsEmptyEntries imports documents and then checks that it exists.
func TestImportDocumentsEmptyEntries(t *testing.T) {
	if getContentTypeFromEnv(t) == driver.ContentTypeVelocypack {
		t.Skip("Not supported on vpack")
	}
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_emptyEntries_test", nil, t)
	docs := []*UserDocWithKey{
		{
			"jan",
			"Jan",
			40,
		},
		{
			"foo",
			"Foo",
			41,
		},
		nil,
		{
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
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs)-1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 1 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 1, stats.Empty, formatRawResponse(raw))
		}
	}
}

// TestImportDocumentsInvalidEntries imports documents and then checks that it exists.
func TestImportDocumentsInvalidEntries(t *testing.T) {
	if getContentTypeFromEnv(t) == driver.ContentTypeVelocypack {
		t.Skip("Not supported on vpack")
	}
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs)-3, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 2 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 2, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 1 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 1, stats.Empty, formatRawResponse(raw))
		}
	}
}

// TestImportDocumentsDuplicateEntries imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntries(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 1 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 1, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
		if stats.Updated != 0 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 0, stats.Updated, formatRawResponse(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, formatRawResponse(raw))
		}
	}
}

// TestImportDocumentsDuplicateEntriesComplete imports documents and then checks that it exists.
func TestImportDocumentsDuplicateEntriesComplete(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
		if stats.Updated != 1 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 1, stats.Updated, formatRawResponse(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, formatRawResponse(raw))
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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
		if stats.Updated != 1 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 1, stats.Updated, formatRawResponse(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, formatRawResponse(raw))
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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
		if stats.Updated != 0 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 0, stats.Updated, formatRawResponse(raw))
		}
		if stats.Ignored != 1 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 1, stats.Ignored, formatRawResponse(raw))
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

// TestImportDocumentsDetails imports documents and then checks that it exists.
func TestImportDocumentsDetails(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_details_test", nil, t)
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
	var details []string
	ctx := driver.WithImportDetails(driver.WithRawResponse(nil, &raw), &details)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != 1 {
			t.Errorf("Expected %d created documents, got %d (json %s)", 1, stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 1 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 1, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
		if stats.Updated != 0 {
			t.Errorf("Expected %d updated documents, got %d (json %s)", 0, stats.Updated, formatRawResponse(raw))
		}
		if stats.Ignored != 0 {
			t.Errorf("Expected %d ignored documents, got %d (json %s)", 0, stats.Ignored, formatRawResponse(raw))
		}

		detailsExpected := `at position 1: creating document failed with error 'unique constraint violated', offending document: {"_key":"jan","name":"Jan2"}`
		if len(details) != 1 {
			t.Errorf("Expected 1 details, to %d", len(details))
		} else if details[0] != detailsExpected {
			t.Errorf("Expected details[0] to be '%s', got '%s'", detailsExpected, details[0])
		}
	}
}

// TestImportDocumentsOverwriteYes imports documents and then checks that it exists.
func TestImportDocumentsOverwriteYes(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_overwriteYes_test", nil, t)
	docs := []interface{}{
		&UserDoc{
			"Jan",
			40,
		},
		map[string]interface{}{
			"name": "Jan2",
		},
	}

	for i := 0; i < 3; i++ {
		var raw []byte
		var details []string
		ctx := driver.WithImportDetails(driver.WithRawResponse(nil, &raw), &details)
		stats, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
			Overwrite: true,
		})
		if err != nil {
			t.Fatalf("Failed to import documents: %s", describe(err))
		} else {
			if stats.Created != 2 {
				t.Errorf("Expected %d created documents, got %d (json %s)", 2, stats.Created, formatRawResponse(raw))
			}
		}

		countExpected := int64(2)
		if count, err := col.Count(nil); err != nil {
			t.Errorf("Failed to count documents: %s", describe(err))
		} else if count != countExpected {
			t.Errorf("Expected count to be %d in round %d, got %d", countExpected, i, count)
		}
	}
}

// TestImportDocumentsOverwriteNo imports documents and then checks that it exists.
func TestImportDocumentsOverwriteNo(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "import_overwriteNo_test", nil, t)
	docs := []interface{}{
		&UserDoc{
			"Jan",
			40,
		},
		map[string]interface{}{
			"name": "Jan2",
		},
	}

	for i := 0; i < 3; i++ {
		var raw []byte
		var details []string
		ctx := driver.WithImportDetails(driver.WithRawResponse(nil, &raw), &details)
		stats, err := col.ImportDocuments(ctx, docs, &driver.ImportDocumentOptions{
			Overwrite: false,
		})
		if err != nil {
			t.Fatalf("Failed to import documents: %s", describe(err))
		} else {
			if stats.Created != 2 {
				t.Errorf("Expected %d created documents, got %d (json %s)", 2, stats.Created, formatRawResponse(raw))
			}
		}

		countExpected := int64(2 * (i + 1))
		if count, err := col.Count(nil); err != nil {
			t.Errorf("Failed to count documents: %s", describe(err))
		} else if count != countExpected {
			t.Errorf("Expected count to be %d in round %d, got %d", countExpected, i, count)
		}
	}
}

// TestImportDocumentsWithKeysInWaitForSyncCollection imports documents into a collection with waitForSync enabled
// and then checks that it exists.
func TestImportDocumentsWithKeysInWaitForSyncCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "TestImportDocumentsWithKeysInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	docs := []UserDocWithKey{
		{
			"jan",
			"Jan",
			40,
		},
		{
			"foo",
			"Foo",
			41,
		},
		{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s %#v", describe(err), err)
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, formatRawResponse(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, formatRawResponse(raw))
		}
		if stats.Empty != 0 {
			t.Errorf("Expected %d empty documents, got %d (json %s)", 0, stats.Empty, formatRawResponse(raw))
		}
	}
}
