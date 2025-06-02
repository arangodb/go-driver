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
	"testing"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

// TestRemoveVertices creates documents, removes them and then checks the removal has succeeded.
func TestRemoveVertices(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "remove_vertices_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "places", t)

	docs := []Book{
		{
			Title: "For reading",
		},
		{
			Title: "For sleeping",
		},
		{
			Title: "For carrying monitors",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	if _, _, err := vc.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	}
	// Should not longer exist
	for i, meta := range metas {
		var readDoc Book
		if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Fatalf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveVerticesReturnOld creates documents, removes them checks the ReturnOld value.
func TestRemoveVerticesReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	g := ensureGraph(ctx, db, "remove_vertices_returnOld_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		{
			Title: "For reading",
		},
		{
			Title: "For sleeping",
		},
		{
			Title: "For carrying monitors",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	require.NoError(t, err)
	require.NoError(t, errs.FirstNonNil())

	oldDocs := make([]Book, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	_, errs, err = vc.RemoveDocuments(ctx, metas.Keys())
	require.NoError(t, err)
	require.NoError(t, errs.FirstNonNil())

	// Check old documents
	for i, doc := range docs {
		require.Equal(t, doc, oldDocs[i])
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveVerticesSilent creates documents, removes them with Silent() and then checks the meta is indeed empty.
func TestRemoveVerticesSilent(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "remove_vertices_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "silence", t)

	docs := []Book{
		{
			Title: "Sleepy",
		},
		{
			Title: "Sleeping",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmetas, rerrs, err := vc.RemoveDocuments(ctx, metas.Keys()); err != nil {
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
		var readDoc Book
		if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveVerticesRevision creates documents, removes them with an incorrect revisions.
func TestRemoveVerticesRevision(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "remove_vertices_revision_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	docs := []Book{
		{
			Title: "Old",
		},
		{
			Title: "New",
		},
	}
	metas, errs, err := vc.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace the documents to get another revision
	replacements := []UserDoc{
		{
			Name: "Anna",
		},
		{
			Name: "Nicole",
		},
	}
	metas2, errs2, err := vc.ReplaceDocuments(ctx, metas.Keys(), replacements)
	if err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	} else if err := errs2.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Try to remove documents with initial revision (must fail)
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	if _, errs, err := vc.RemoveDocuments(initialRevCtx, metas.Keys()); err != nil {
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
	if _, errs, err := vc.RemoveDocuments(replacedRevCtx, metas.Keys()); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Should not longer exist
	for i, meta := range metas {
		var readDoc Book
		if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveVerticesKeyEmpty removes a document it with an empty key.
func TestRemoveVerticesKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertices_test", nil, t)
	g := ensureGraph(ctx, db, "remove_vertices_keyEmpty_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "failures", t)

	if _, _, err := vc.RemoveDocuments(nil, []string{""}); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
