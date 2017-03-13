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
	"encoding/json"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestUpdateEdge creates a document, updates it and then checks the update has succeeded.
func TestUpdateEdge(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_"
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
	update := map[string]interface{}{
		"distance": 555,
	}
	if _, err := ec.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	doc.Distance = 555
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestUpdateEdgeReturnOld creates a document, updates it checks the ReturnOld value.
func TestUpdateEdgeReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_returnOld_"
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
	update := map[string]interface{}{
		"distance": 333,
	}
	var old RouteEdge
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := ec.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestUpdateEdgeReturnNew creates a document, updates it checks the ReturnNew value.
func TestUpdateEdgeReturnNew(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_returnNew_"
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
	update := map[string]interface{}{
		"_from": to.ID.String(),
	}
	var newDoc RouteEdge
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := ec.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := doc
	expected.From = to.ID.String()
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestUpdateEdgeKeepNullTrue creates a document, updates it with KeepNull(true) and then checks the update has succeeded.
func TestUpdateEdgeKeepNullTrue(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_keepNullTrue_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := AccountEdge{
		From: from.ID.String(),
		To:   to.ID.String(),
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"_to":  from.ID.String(),
		"user": nil,
	}
	if _, err := ec.UpdateDocument(driver.WithKeepNull(ctx, true), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc map[string]interface{}
	var rawResponse []byte
	ctx = driver.WithRawResponse(ctx, &rawResponse)
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	// We parse to this type of map, since unmarshalling nil values to a map of type map[string]interface{}
	// will cause the entry to be deleted.
	var jsonMap map[string]*json.RawMessage
	if err := json.Unmarshal(rawResponse, &jsonMap); err != nil {
		t.Fatalf("Failed to parse raw response: %s", describe(err))
	}
	// Get "edge" field and unmarshal it
	if raw, found := jsonMap["edge"]; !found {
		t.Errorf("Expected edge to be found but got not found")
	} else {
		jsonMap = nil
		if err := json.Unmarshal(*raw, &jsonMap); err != nil {
			t.Fatalf("Failed to parse raw edge object: %s", describe(err))
		}
		if raw, found := jsonMap["user"]; !found {
			t.Errorf("Expected user to be found but got not found")
		} else if raw != nil {
			t.Errorf("Expected user to be found and nil, got %s", string(*raw))
		}
	}
}

// TestUpdateEdgeKeepNullFalse creates a document, updates it with KeepNull(false) and then checks the update has succeeded.
func TestUpdateEdgeKeepNullFalse(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_keepNullFalse_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := AccountEdge{
		From: from.ID.String(),
		To:   to.ID.String(),
		User: &UserDoc{
			"Mathilda",
			45,
		},
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"_to":  from.ID.String(),
		"user": nil,
	}
	if _, err := ec.UpdateDocument(driver.WithKeepNull(ctx, false), meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	readDoc := doc
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if readDoc.User == nil {
		t.Errorf("Expected user to be untouched, got %v", readDoc.User)
	}
}

// TestUpdateEdgeSilent creates a document, updates it with Silent() and then checks the meta is indeed empty.
func TestUpdateEdgeSilent(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_silent_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 7,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"distance": 61,
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := ec.UpdateDocument(ctx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestUpdateEdgeRevision creates a document, updates it with a specific (correct) revision.
// Then it attempts an update with an incorrect revision which must fail.
func TestUpdateEdgeRevision(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_revision_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	cities := ensureCollection(ctx, db, prefix+"city", nil, t)
	states := ensureCollection(ctx, db, prefix+"state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 7,
	}
	meta, err := ec.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Update document with correct revision
	update := map[string]interface{}{
		"distance": 34,
	}
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	var updatedRevCtx context.Context
	if meta2, err := ec.UpdateDocument(initialRevCtx, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else {
		updatedRevCtx = driver.WithRevision(ctx, meta2.Rev)
		if meta2.Rev == meta.Rev {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", meta.Rev, meta2.Rev)
		}
	}

	// Update document with incorrect revision
	update["distance"] = 35
	if _, err := ec.UpdateDocument(initialRevCtx, meta.Key, update); !driver.IsPreconditionFailed(err) {
		t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Update document  once more with correct revision
	update["distance"] = 36
	if _, err := ec.UpdateDocument(updatedRevCtx, meta.Key, update); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateEdgeKeyEmpty updates a document it with an empty key.
func TestUpdateEdgeKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_keyEmpty_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := ec.UpdateDocument(nil, "", update); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateEdgeUpdateNil updates a document it with a nil update.
func TestUpdateEdgeUpdateNil(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	prefix := "update_edge_updateNil_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	if _, err := ec.UpdateDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
