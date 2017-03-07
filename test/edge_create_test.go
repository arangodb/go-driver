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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestCreateEdge creates an edge and then checks that it exists.
func TestCreateEdge(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	g := ensureGraph(ctx, db, "create_edge_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	meta, err := ec.CreateDocument(ctx, driver.EdgeDocument{From: from.ID, To: to.ID})
	if err != nil {
		t.Fatalf("Failed to create new edge: %s", describe(err))
	}
	// Document must exists now
	var readDoc driver.EdgeDocument
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read edge '%s': %s", meta.Key, describe(err))
	} else {
		if readDoc.From != from.ID {
			t.Errorf("Got invalid _from. Expected '%s', got '%s'", from.ID, readDoc.From)
		}
		if readDoc.To != to.ID {
			t.Errorf("Got invalid _to. Expected '%s', got '%s'", to.ID, readDoc.To)
		}
	}
}

// TestCreateCustomEdge creates an edge with a custom type and then checks that it exists.
func TestCreateCustomEdge(t *testing.T) {
	var ctx context.Context
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	g := ensureGraph(ctx, db, "create_custom_edge_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 7,
	}
	meta, err := ec.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new edge: %s", describe(err))
	}
	// Document must exists now
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read edge '%s': %s", meta.Key, describe(err))
	} else if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got invalid return document. Expected '%+v', got '%+v'", doc, readDoc)
	}
}

// TestCreateEdgeReturnNew creates a document and checks the document returned in in ReturnNew.
func TestCreateEdgeReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	g := ensureGraph(ctx, db, "create_edge_return_new_est", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 7,
	}
	var newDoc RouteEdge
	meta, err := ec.CreateDocument(driver.WithReturnNew(ctx, &newDoc), doc)
	if err != nil {
		t.Fatalf("Failed to create new edge: %s", describe(err))
	}
	// NewDoc must equal doc
	if !reflect.DeepEqual(doc, newDoc) {
		t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", doc, newDoc)
	}
	// Document must exists now
	var readDoc RouteEdge
	if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestCreateEdgeSilent creates a document with WithSilent.
func TestCreateEdgeSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	g := ensureGraph(ctx, db, "create_edge_silent_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	doc := RouteEdge{
		From:     from.ID.String(),
		To:       to.ID.String(),
		Distance: 7,
	}
	if meta, err := ec.CreateDocument(driver.WithSilent(ctx), doc); err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestCreateEdgeNil creates a document with a nil document.
func TestCreateEdgeNil(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edge_test", nil, t)
	g := ensureGraph(ctx, db, "create_edge_nil_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)

	if _, err := ec.CreateDocument(nil, nil); !driver.IsInvalidArgument(err) {
		t.Fatalf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
