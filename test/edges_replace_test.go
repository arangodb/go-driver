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

// TestReplaceEdges creates documents, replaces them and then checks the replacements have succeeded.
func TestReplaceEdges(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)
	male := ensureCollection(ctx, db, "male", nil, t)
	female := ensureCollection(ctx, db, "female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replacement docs
	replacements := []driver.EdgeDocument{
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
	}
	if _, _, err := ec.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Read replaced documents
	for i, meta := range metas {
		var readDoc driver.EdgeDocument
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if !reflect.DeepEqual(replacements[i], readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, replacements[i], readDoc)
		}
	}
}

// TestReplaceEdgesReturnOld creates documents, replaces them checks the ReturnOld values.
func TestReplaceEdgesReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_returnOld_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)
	male := ensureCollection(ctx, db, "male", nil, t)
	female := ensureCollection(ctx, db, "female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "married",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []driver.EdgeDocument{
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
	}
	oldDocs := make([]RelationEdge, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := ec.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	}
	// Check old document
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestReplaceEdgesReturnNew creates documents, replaces them checks the ReturnNew values.
func TestReplaceEdgesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_returnNew_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)
	male := ensureCollection(ctx, db, "male", nil, t)
	female := ensureCollection(ctx, db, "female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "married",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []driver.EdgeDocument{
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
	}
	newDocs := make([]driver.EdgeDocument, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := ec.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
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

// TestReplaceEdgesSilent creates documents, replaces them with Silent() and then checks the meta is indeed empty.
func TestReplaceEdgesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_silent_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)
	male := ensureCollection(ctx, db, "male", nil, t)
	female := ensureCollection(ctx, db, "female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "married",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Replace documents
	replacements := []driver.EdgeDocument{
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
		driver.EdgeDocument{
			From: to.ID,
			To:   from.ID,
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := ec.ReplaceDocuments(ctx, metas.Keys(), replacements); err != nil {
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

// TestReplaceEdgesRevision creates documents, replaces then with a specific (correct) revisions.
// Then it attempts replacements with incorrect revisions which must fail.
func TestReplaceEdgesRevision(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_revision_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)
	male := ensureCollection(ctx, db, "male", nil, t)
	female := ensureCollection(ctx, db, "female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "married",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace documents with correct revisions
	replacements := []RelationEdge{
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "old-friend",
		},
		RelationEdge{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "just-married",
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var replacedRevCtx context.Context
	if metas2, errs, err := ec.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
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
	replacements[0].Type = "Wrong deal 1"
	replacements[1].Type = "Wrong deal 2"
	if _, errs, err := ec.ReplaceDocuments(initialRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for i, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError at %d, got %s", i, describe(err))
			}
		}
	}

	// Replace document once more with correct revision
	replacements[0].Type = "Good deal 1"
	replacements[1].Type = "Good deal 2"
	if _, errs, err := ec.ReplaceDocuments(replacedRevCtx, metas.Keys(), replacements); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
}

// TestReplaceEdgesKeyEmpty replaces a document it with an empty key.
func TestReplaceEdgesKeyEmpty(t *testing.T) {
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

// TestReplaceEdgesUpdateNil replaces a document it with a nil update.
func TestReplaceEdgesUpdateNil(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_updateNil_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)

	if _, _, err := ec.ReplaceDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceEdgesUpdateLenDiff replacements documents with a different number of documents, keys.
func TestReplaceEdgesUpdateLenDiff(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "replace_edges_updateNil_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "relation", []string{"male", "female"}, []string{"male", "female"}, t)

	replacements := []map[string]interface{}{
		map[string]interface{}{
			"name": "name1",
		},
		map[string]interface{}{
			"name": "name2",
		},
	}
	if _, _, err := ec.ReplaceDocuments(nil, []string{"only1"}, replacements); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
