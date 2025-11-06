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

	driver "github.com/arangodb/go-driver"
)

// TestUpdateVertices creates documents, updates them and then checks the updates have succeeded.
func TestUpdateVertices(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test1", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "relations", t)

	docs := []UserDoc{
		{
			Name: "Bob",
		},
		{
			Name: "Anna",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
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
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc UserDoc
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		doc := docs[i]
		doc.Name = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(doc, readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, readDoc)
		}
	}
}

// TestUpdateVerticesReturnOld creates documents, updates them checks the ReturnOld values.
func TestUpdateVerticesReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertices_update_test2", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_returnOld_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		{
			Title: "Pinkeltje op de maan",
		},
		{
			Title: "Pinkeltje in het bos",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"Title": "Updated1",
		},
		{
			"Title": "Updated2",
		},
	}
	oldDocs := make([]Book, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Check old documents
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestUpdateVerticesReturnNew creates documents, updates them checks the ReturnNew values.
func TestUpdateVerticesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertices_update_test3", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_returnOld_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "users", t)

	docs := []UserDoc{
		{
			Name: "Tony",
		},
		{
			Name: "Parker",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
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
	newDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
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

// TestUpdateVerticesKeepNullTrue creates documents, updates them with KeepNull(true) and then checks the updates have succeeded.
func TestUpdateVerticesKeepNullTrue(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	conn := c.Connection()
	db := ensureDatabase(ctx, c, "vertices_update_test4", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_keepNullTrue_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "keepers", t)

	docs := []Account{
		{
			ID: "123",
			User: &UserDoc{
				"Greata",
				77,
			},
		},
		{
			ID: "456",
			User: &UserDoc{
				"Mathilda",
				45,
			},
		},
	}

	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Update documents
	updates := []map[string]interface{}{
		{
			"id":   "abc",
			"user": nil,
		},
		{
			"id":   "def",
			"user": nil,
		},
	}
	if _, _, err := ec.UpdateDocuments(driver.WithKeepNull(ctx, true), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc map[string]interface{}
		var rawResponse []byte
		ctxInner := driver.WithRawResponse(ctx, &rawResponse)
		if _, err := ec.ReadDocument(ctxInner, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document %d '%s': %s", i, meta.Key, describe(err))
		}
		// We parse to this type of map, since unmarshalling nil values to a map of type map[string]interface{}
		// will cause the entry to be deleted.
		var jsonMap map[string]*driver.RawObject
		if err := conn.Unmarshal(rawResponse, &jsonMap); err != nil {
			t.Fatalf("Failed to parse raw response: %s", describe(err))
		}
		// Get "vertex" field and unmarshal it
		if raw, found := jsonMap["vertex"]; !found {
			t.Errorf("Expected vertex to be found but got not found")
		} else {
			jsonMap = nil
			if err := conn.Unmarshal(*raw, &jsonMap); err != nil {
				t.Fatalf("Failed to parse raw vertex object: %s", describe(err))
			}
			if raw, found := jsonMap["user"]; !found {
				t.Errorf("Expected user to be found but got not found")
			} else if raw != nil {
				t.Errorf("Expected user to be found and nil, got %s", string(*raw))
			}
		}
	}
}

// TestUpdateVerticesKeepNullFalse creates documents, updates them with KeepNull(false) and then checks the updates have succeeded.
func TestUpdateVerticesKeepNullFalse(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test5", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_keepNullFalse_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "accounts", t)

	docs := []Account{
		{
			ID: "123",
			User: &UserDoc{
				"Greata",
				77,
			},
		},
		{
			ID: "456",
			User: &UserDoc{
				"Mathilda",
				45,
			},
		},
	}

	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update document
	updates := []map[string]interface{}{
		{
			"id":   "abc",
			"user": nil,
		},
		{
			"id":   "def",
			"user": nil,
		},
	}
	if _, _, err := ec.UpdateDocuments(driver.WithKeepNull(ctx, false), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		readDoc := docs[i]
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if readDoc.User == nil {
			t.Errorf("Expected user to be untouched, got %v", readDoc.User)
		}
	}
}

// TestUpdateVerticesSilent creates documents, updates them with Silent() and then checks the metas are indeed empty.
func TestUpdateVerticesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test6", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_silent_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "moments", t)

	docs := []Book{
		{
			Title: "Foo",
		},
		{
			Title: "Oops",
		},
	}
	metas, _, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"Title": 61,
		},
		{
			"Title": 16,
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	} else if strings.Join(metas.Keys(), "") != "" {
		t.Errorf("Expected empty meta, got %v", metas)
	}
}

// TestUpdateVerticesRevision creates documents, updates them with a specific (correct) revisions.
// Then it attempts an update with an incorrect revisions which must fail.
func TestUpdateVerticesRevision(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test7", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_revision_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "revisions", t)

	docs := []Book{
		{
			Title: "Roman age",
		},
		{
			Title: "New age",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
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
			"Title": 34,
		},
		{
			"Title": 77,
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var updatedRevCtx context.Context
	if metas2, _, err := ec.UpdateDocuments(initialRevCtx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else {
		updatedRevCtx = driver.WithRevisions(ctx, metas2.Revs())
		if strings.Join(metas2.Revs(), ",") == strings.Join(metas.Revs(), ",") {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", strings.Join(metas.Revs(), ","), strings.Join(metas2.Revs(), ","))
		}
	}

	// Update documents with incorrect revisions
	updates[0]["Title"] = 35
	var rawResponse []byte
	if _, errs, err := ec.UpdateDocuments(driver.WithRawResponse(initialRevCtx, &rawResponse), metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for _, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError, got %s (resp: %s", describe(err), string(rawResponse))
			}
		}
	}

	// Update documents once more with correct revisions
	updates[0]["Title"] = 36
	if _, _, err := ec.UpdateDocuments(updatedRevCtx, metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateVerticesKeyEmpty updates documents with an empty key.
func TestUpdateVerticesKeyEmpty(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test8", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_keyEmpty_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "lonely", t)

	// Update document
	updates := []map[string]interface{}{
		{
			"name": "Updated",
		},
	}
	if _, _, err := ec.UpdateDocuments(nil, []string{""}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateVerticesUpdateNil updates documents it with a nil update.
func TestUpdateVerticesUpdateNil(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test9", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_updateNil_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "nilAndSome", t)

	if _, _, err := ec.UpdateDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateVerticesUpdateLenDiff updates documents with a different number of updates, keys.
func TestUpdateVerticesUpdateLenDiff(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_update_test10", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "update_vertices_updateLenDiff_test", nil, t)
	ec := ensureVertexCollection(ctx, g, "diffs", t)

	updates := []map[string]interface{}{
		{
			"name": "name1",
		},
		{
			"name": "name2",
		},
	}
	if _, _, err := ec.UpdateDocuments(nil, []string{"only1"}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
