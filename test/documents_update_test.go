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
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

// TestUpdateDocuments1 creates documents, updates them and then checks the updates have succeeded.
func TestUpdateDocuments1(t *testing.T) {
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
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
		{
			"Otto",
			43,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"name": "Updated1",
		},
		{
			"name": "Updated2",
		},
	}
	if _, _, err := col.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc UserDoc
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		doc := docs[i]
		doc.Name = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(doc, readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, readDoc)
		}
	}
}

// TestUpdateDocumentsReturnOld creates documents, updates them checks the ReturnOld values.
func TestUpdateDocumentsReturnOld(t *testing.T) {
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
			"Foo",
			70,
		},
		{
			"Mindy",
			70,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"name": "Updated1",
		},
		{
			"name": "Updated2",
		},
		{
			"name": "Updated3",
		},
	}
	oldDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := col.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}

	returnOld, exist := driver.HasReturnOld(ctx)
	require.True(t, exist, "ReturnOld not set")

	oldDocs2, ok := returnOld.([]UserDoc)
	require.True(t, ok, "ReturnOld not set correctly")
	require.Len(t, oldDocs2, len(oldDocs), "ReturnOld not set correctly")
	require.Equal(t, oldDocs[0].Age, oldDocs2[0].Age, "ReturnOld not set correctly")
	require.Equal(t, oldDocs[1].Name, oldDocs2[1].Name, "ReturnOld not set correctly")

	// Check old documents
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestUpdateDocumentsReturnNew creates documents, updates them checks the ReturnNew values.
func TestUpdateDocumentsReturnNew(t *testing.T) {
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
			"Duck",
			21,
		},
		{
			"Donald",
			53,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"name": "Updated1",
		},
		{
			"name": "Updated2",
		},
		{
			"name": "Updated3",
		},
	}
	newDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := col.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Check new documents
	for i, doc := range docs {
		expected := doc
		expected.Name = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(expected, newDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, expected, newDocs[i])
		}
	}
}

// TestUpdateDocumentsKeepNullTrue creates documents, updates them with KeepNull(true) and then checks the updates have succeeded.
func TestUpdateDocumentsKeepNullTrue(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	conn := c.Connection()
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []Account{
		{
			ID: "1234",
			User: &UserDoc{
				"Mathilda",
				45,
			},
		},
		{
			ID: "432",
			User: &UserDoc{
				"Clair",
				12,
			},
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Update documents
	updates := []map[string]interface{}{
		{
			"id":   "5678",
			"user": nil,
		},
		{
			"id":   "742",
			"user": nil,
		},
	}
	if _, _, err := col.UpdateDocuments(driver.WithKeepNull(ctx, true), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc map[string]interface{}
		var rawResponse []byte
		ctx = driver.WithRawResponse(ctx, &rawResponse)
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document %d '%s': %s", i, meta.Key, describe(err))
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
}

// TestUpdateDocumentsKeepNullFalse creates documents, updates them with KeepNull(false) and then checks the updates have succeeded.
func TestUpdateDocumentsKeepNullFalse(t *testing.T) {
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
	col := ensureCollection(ctx, db, "documents_test", nil, t)
	docs := []Account{
		{
			ID: "1234",
			User: &UserDoc{
				"Mathilda",
				45,
			},
		},
		{
			ID: "364",
			User: &UserDoc{
				"Jo",
				42,
			},
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update document
	updates := []map[string]interface{}{
		{
			"id":   "5678",
			"user": nil,
		},
		{
			"id":   "753",
			"user": nil,
		},
	}
	if _, _, err := col.UpdateDocuments(driver.WithKeepNull(ctx, false), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		readDoc := docs[i]
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if readDoc.User == nil {
			t.Errorf("Expected user to be untouched, got %v", readDoc.User)
		}
	}
}

// TestUpdateDocumentsSilent creates documents, updates them with Silent() and then checks the metas are indeed empty.
func TestUpdateDocumentsSilent(t *testing.T) {
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
			"Jo",
			19,
		},
	}
	metas, _, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"age": "61",
		},
		{
			"age": "16",
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := col.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	} else if strings.Join(metas.Keys(), "") != "" {
		t.Errorf("Expected empty meta, got %v", metas)
	}
}

// TestUpdateDocumentsRevision creates documents, updates them with a specific (correct) revisions.
// Then it attempts an update with an incorrect revisions which must fail.
func TestUpdateDocumentsRevision(t *testing.T) {
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
			"Revision",
			33,
		},
		{
			"Revision2",
			34,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Fatalf("Expected %d metas, got %d", len(docs), len(metas))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Update documents with correct revisions
	updates := []map[string]interface{}{
		{
			"age": 34,
		},
		{
			"age": 77,
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var updatedRevCtx context.Context
	if metas2, _, err := col.UpdateDocuments(initialRevCtx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else {
		updatedRevCtx = driver.WithRevisions(ctx, metas2.Revs())
		if strings.Join(metas2.Revs(), ",") == strings.Join(metas.Revs(), ",") {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", strings.Join(metas.Revs(), ","), strings.Join(metas2.Revs(), ","))
		}
	}

	// Update documents with incorrect revisions
	updates[0]["age"] = 35
	var rawResponse []byte
	if _, errs, err := col.UpdateDocuments(driver.WithRawResponse(initialRevCtx, &rawResponse), metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for _, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError, got %s (resp: %s", describe(err), string(rawResponse))
			}
		}
	}

	// Update documents once more with correct revisions
	updates[0]["age"] = 36
	if _, _, err := col.UpdateDocuments(updatedRevCtx, metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateDocumentsKeyEmpty updates documents with an empty key.
func TestUpdateDocumentsKeyEmpty(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "documents_test", nil, t)
	// Update document
	updates := []map[string]interface{}{
		{
			"name": "Updated",
		},
	}
	if _, _, err := col.UpdateDocuments(nil, []string{""}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateDocumentsUpdateNil updates documents it with a nil update.
func TestUpdateDocumentsUpdateNil(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "documents_test", nil, t)
	if _, _, err := col.UpdateDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateDocumentsUpdateLenDiff updates documents with a different number of updates, keys.
func TestUpdateDocumentsUpdateLenDiff(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "documents_test", nil, t)
	updates := []map[string]interface{}{
		{
			"name": "name1",
		},
		{
			"name": "name2",
		},
	}
	if _, _, err := col.UpdateDocuments(nil, []string{"only1"}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateDocumentsInWaitForSyncCollection creates documents in a collection with waitForSync enabled,
// updates them and then checks the updates have succeeded.
func TestUpdateDocumentsInWaitForSyncCollection(t *testing.T) {
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
	col := ensureCollection(ctx, db, "TestUpdateDocumentsInWaitForSyncCollection", &driver.CreateCollectionOptions{
		WaitForSync: true,
	}, t)
	docs := []UserDoc{
		{
			"Piere",
			23,
		},
		{
			"Otto",
			43,
		},
	}
	metas, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"name": "Updated1",
		},
		{
			"name": "Updated2",
		},
	}
	if _, _, err := col.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc UserDoc
		if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		doc := docs[i]
		doc.Name = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(doc, readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, readDoc)
		}
	}
}
