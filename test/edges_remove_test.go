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

// TestRemoveEdges creates documents, removes them and then checks the removal has succeeded.
func TestRemoveEdges(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "remove_edges_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 68,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	if _, _, err := ec.RemoveDocuments(ctx, metas.Keys()); err != nil {
		t.Fatalf("Failed to remove documents: %s", describe(err))
	}
	// Should not longer exist
	for i, meta := range metas {
		var readDoc Account
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Fatalf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveEdgesReturnOld creates documents, removes them checks the ReturnOld value.
func TestRemoveEdgesReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	prefix := "remove_edges_returnOld_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 68,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	oldDocs := make([]RouteEdge, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	_, errs, err = ec.RemoveDocuments(ctx, metas.Keys())
	require.Nil(t, err)
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

// TestRemoveEdgesSilent creates documents, removes them with Silent() and then checks the meta is indeed empty.
func TestRemoveEdgesSilent(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "remove_edges_silent_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmetas, rerrs, err := ec.RemoveDocuments(ctx, metas.Keys()); err != nil {
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
		var readDoc RouteEdge
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveEdgesRevision creates documents, removes them with an incorrect revisions.
func TestRemoveEdgesRevision(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "remove_edges_revision_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Replace the documents to get another revision
	replacements := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 880,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 210,
		},
	}
	metas2, errs2, err := ec.ReplaceDocuments(ctx, metas.Keys(), replacements)
	if err != nil {
		t.Fatalf("Failed to replace documents: %s", describe(err))
	} else if err := errs2.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Try to remove documents with initial revision (must fail)
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	if _, errs, err := ec.RemoveDocuments(initialRevCtx, metas.Keys()); err != nil {
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
	if _, errs, err := ec.RemoveDocuments(replacedRevCtx, metas.Keys()); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Should not longer exist
	for i, meta := range metas {
		var readDoc RouteEdge
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
			t.Errorf("Expected NotFoundError at %d, got  %s", i, describe(err))
		}
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveEdgesKeyEmpty removes a document it with an empty key.
func TestRemoveEdgesKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "remove_edges_keyEmpty_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	if _, _, err := ec.RemoveDocuments(nil, []string{""}); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
