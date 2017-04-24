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

	driver "github.com/arangodb/go-driver"
)

// TestReplaceVertices creates documents, replaces them and then checks the replacements have succeeded.
func TestReplaceVertices(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "male", t)

	docs := []UserDoc{
		UserDoc{
			Name: "Bob",
		},
		UserDoc{
			Name: "Joe",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replacement docs
	replacements := []Book{
		Book{
			Title: "For bob",
		},
		Book{
			Title: "For joe",
		},
	}
	if _, _, err := vc.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Read replaced documents
	for i, meta := range metas {
		var readDoc Book
		if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if !reflect.DeepEqual(replacements[i], readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, replacements[i], readDoc)
		}
	}
}

// TestReplaceVerticesReturnOld creates documents, replaces them checks the ReturnOld values.
func TestReplaceVerticesReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.3", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_returnOld_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "pensions", t)

	docs := []UserDoc{
		UserDoc{
			Name: "Bob",
		},
		UserDoc{
			Name: "Joe",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []Book{
		Book{
			Title: "For bob",
		},
		Book{
			Title: "For joe",
		},
	}
	oldDocs := make([]UserDoc, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := vc.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Check old document
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestReplaceVerticesReturnNew creates documents, replaces them checks the ReturnNew values.
func TestReplaceVerticesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.3", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_returnNew_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		Book{
			Title: "For bob",
		},
		Book{
			Title: "For joe",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []Book{
		Book{
			Title: "For the new bob",
		},
		Book{
			Title: "For the new joe",
		},
	}
	newDocs := make([]Book, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := vc.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
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

// TestReplaceVerticesSilent creates documents, replaces them with Silent() and then checks the meta is indeed empty.
func TestReplaceVerticesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "moments", t)

	docs := []Book{
		Book{
			Title: "Fly me to the moon",
		},
		Book{
			Title: "Fly me to the earth",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []UserDoc{
		UserDoc{
			Name: "Bob",
		},
		UserDoc{
			Name: "Christal",
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := vc.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
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

// TestReplaceVerticesRevision creates documents, replaces then with a specific (correct) revisions.
// Then it attempts replacements with incorrect revisions which must fail.
func TestReplaceVerticesRevision(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_revision_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "planets", t)

	docs := []Book{
		Book{
			Title: "Pluto",
		},
		Book{
			Title: "Mars",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace documents with correct revisions
	replacements := []UserDoc{
		UserDoc{
			Name: "Bob",
		},
		UserDoc{
			Name: "Christal",
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var replacedRevCtx context.Context
	if metas2, errs, err := vc.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
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
	replacements[0].Name = "Wrong deal 1"
	replacements[1].Name = "Wrong deal 2"
	if _, errs, err := vc.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for i, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError at %d, got %s", i, describe(err))
			}
		}
	}

	// Replace document once more with correct revision
	replacements[0].Name = "Good deal 1"
	replacements[1].Name = "Good deal 2"
	if _, errs, err := vc.ReplaceDocuments(replacedRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
}

// TestReplaceVerticesKeyEmpty replaces a document it with an empty key.
func TestReplaceVerticesKeyEmpty(t *testing.T) {
	ctx := context.TODO()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_keyEmpty_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "planets", t)

	// Replacement document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, _, err := vc.ReplaceDocuments(nil, []string{""}, replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceVerticesUpdateNil replaces a document it with a nil update.
func TestReplaceVerticesUpdateNil(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_updateNil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "relations", t)

	if _, _, err := vc.ReplaceDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceVerticesUpdateLenDiff replacements documents with a different number of documents, keys.
func TestReplaceVerticesUpdateLenDiff(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "replace_vertices_updateNil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "failures", t)

	replacements := []map[string]interface{}{
		map[string]interface{}{
			"name": "name1",
		},
		map[string]interface{}{
			"name": "name2",
		},
	}
	if _, _, err := vc.ReplaceDocuments(nil, []string{"only1"}, replacements); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
