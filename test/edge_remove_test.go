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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestRemoveEdge creates a document, remove it and then checks the removal has succeeded.
func TestRemoveEdge(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "remove_edge_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 32,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	if _, err := ec.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Should not longer exist
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveEdgeReturnOld creates a document, removes it with ReturnOld, which is an invalid argument.
func TestRemoveEdgeReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "remove_edge_returnOld_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 32,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	var old RouteEdge
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := ec.RemoveDocument(ctx, meta.Key); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestRemoveEdgeSilent creates a document, removes it with Silent() and then checks the meta is indeed empty.
func TestRemoveEdgeSilent(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "remove_edge_silent_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 77,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmeta, err := ec.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	} else if rmeta.Key != "" {
		t.Errorf("Expected empty meta, got %v", rmeta)
	}
	// Should not longer exist
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveEdgeRevision creates a document, removes it with an incorrect revision.
func TestRemoveEdgeRevision(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "remove_edge_revision_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 77,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Replace the document to get another revision
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: 88,
	}
	meta2, err := ec.ReplaceDocument(ctx, meta.Key, replacement)
	if err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}

	// Try to remove document with initial revision (must fail)
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	if _, err := ec.RemoveDocument(initialRevCtx, meta.Key); !driver.IsPreconditionFailed(err) {
		t.Fatalf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Try to remove document with correct revision (must succeed)
	replacedRevCtx := driver.WithRevision(ctx, meta2.Rev)
	if _, err := ec.RemoveDocument(replacedRevCtx, meta.Key); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Should not longer exist
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}

	// Document must not exists now
	if found, err := ec.DocumentExists(nil, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if found {
		t.Errorf("DocumentExists returned true for '%s', expected false", meta.Key)
	}
}

// TestRemoveEdgeKeyEmpty removes a document it with an empty key.
func TestRemoveEdgeKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "remove_edge_nil_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	if _, err := ec.RemoveDocument(nil, ""); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
