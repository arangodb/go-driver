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
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestReplaceEdge creates a document, replaces it and then checks the replacement has succeeded.
func TestReplaceEdge(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 123,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replacement doc
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: 567,
	}
	if _, err := ec.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Read replaces document
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(replacement, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", replacement, readDoc)
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeReturnOld creates a document, replaces it checks the ReturnOld value.
func TestReplaceEdgeReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_returnOld_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 123,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replace document
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: 246,
	}
	var old RouteEdge
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := ec.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeReturnNew creates a document, replaces it checks the ReturnNew value.
func TestReplaceEdgeReturnNew(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_returnNew_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 123,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: 246,
	}
	var newDoc RouteEdge
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := ec.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := replacement
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeSilent creates a document, replaces it with Silent() and then checks the meta is indeed empty.
func TestReplaceEdgeSilent(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_returnNew_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 0,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: -1,
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := ec.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeRevision creates a document, replaces it with a specific (correct) revision.
// Then it attempts a replacement with an incorrect revision which must fail.
func TestReplaceEdgeRevision(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_revision_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 0,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Replace document with correct revision
	replacement := RouteEdge{
		From:     to.ID.String(),
		To:       from.ID.String(),
		Distance: -1,
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var replacedRevCtx context.Context
	if meta2, err := ec.ReplaceDocument(initialRevCtx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	} else {
		replacedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', replaced revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Replace document with incorrect revision
	replacement.Distance = 999
	if _, err := ec.ReplaceDocument(initialRevCtx, meta.Key, replacement); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Replace document once more with correct revision
	replacement.Distance = 111
	if _, err := ec.ReplaceDocument(replacedRevCtx, meta.Key, replacement); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeKeyEmpty replaces a document it with an empty key.
func TestReplaceEdgeKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_keyEmpty_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	// Update document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := ec.ReplaceDocument(nil, "", replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestReplaceEdgeUpdateNil replaces a document it with a nil update.
func TestReplaceEdgeUpdateNil(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "replace_edge_updateNil_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	if _, err := ec.ReplaceDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	err := db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
