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

// TestCreateEdges creates documents and then checks that it exists.
func TestCreateEdges(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "create_edges_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 68,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}

			// Document must exists now
			var readDoc RouteEdge
			if _, err := ec.ReadDocument(nil, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateEdgesReturnNew creates documents and checks the document returned in in ReturnNew.
func TestCreateEdgesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "create_edges_returnNew_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 68,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	newDocs := make([]RouteEdge, len(docs))
	metas, errs, err := ec.CreateDocuments(driver.WithReturnNew(ctx, newDocs), docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		for i := 0; i < len(docs); i++ {
			if err := errs[i]; err != nil {
				t.Errorf("Expected no error at index %d, got %s", i, describe(err))
			}
			// NewDoc must equal doc
			if !reflect.DeepEqual(docs[i], newDocs[i]) {
				t.Errorf("Got wrong ReturnNew document. Expected %+v, got %+v", docs[i], newDocs[i])
			}
			// Document must exists now
			var readDoc RouteEdge
			if _, err := ec.ReadDocument(ctx, metas[i].Key, &readDoc); err != nil {
				t.Fatalf("Failed to read document '%s': %s", metas[i].Key, describe(err))
			}
			if !reflect.DeepEqual(docs[i], readDoc) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], readDoc)
			}
		}
	}
}

// TestCreateEdgesSilent creates documents with WithSilent.
func TestCreateEdgesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "create_edges_silent_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities := ensureCollection(ctx, db, "city", nil, t)
	states := ensureCollection(ctx, db, "state", nil, t)
	from := createDocument(ctx, cities, map[string]interface{}{"name": "Venlo"}, t)
	to := createDocument(ctx, states, map[string]interface{}{"name": "Limburg"}, t)

	docs := []RouteEdge{
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 40,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 68,
		},
		RouteEdge{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 21,
		},
	}
	if metas, errs, err := ec.CreateDocuments(driver.WithSilent(ctx), docs); err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else {
		if len(metas) != 0 {
			t.Errorf("Expected 0 metas, got %d", len(metas))
		}
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(errs))
		}
	}
}

// TestCreateEdgesNil creates multiple documents with a nil documents input.
func TestCreateEdgesNil(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "create_edges_nil_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	if _, _, err := ec.CreateDocuments(nil, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestCreateEdgesNonSlice creates multiple documents with a non-slice documents input.
func TestCreateEdgesNonSlice(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	g := ensureGraph(ctx, db, "create_edges_nonSlice_test", nil, t)
	ec := ensureEdgeCollection(ctx, g, "citiesPerState", []string{"city"}, []string{"state"}, t)

	var obj UserDoc
	if _, _, err := ec.CreateDocuments(nil, &obj); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	var m map[string]interface{}
	if _, _, err := ec.CreateDocuments(nil, &m); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
