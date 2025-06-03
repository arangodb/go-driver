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

	"github.com/arangodb/go-driver"
)

// TestCreateEdges creates documents and then checks that it exists.
func TestCreateEdges(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	prefix := "create_edges_"
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
	} else if len(metas) != len(docs) {
		t.Errorf("Expected %d metas, got %d", len(docs), len(metas))
	} else {
		// Read back using ReadDocuments
		keys := make([]string, len(docs))
		for i, m := range metas {
			keys[i] = m.Key
		}
		readDocs := make([]RouteEdge, len(docs))
		if _, _, err := ec.ReadDocuments(nil, keys, readDocs); err != nil {
			t.Fatalf("Failed to read documents: %s", describe(err))
		}
		for i, d := range readDocs {
			if !reflect.DeepEqual(docs[i], d) {
				t.Errorf("Got wrong document. Expected %+v, got %+v", docs[i], d)
			}
		}
		// Read back using individual ReadDocument requests
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
	c := createClient(t, nil)
	// TODO refactor ME
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	prefix := "create_edges_returnNew_"
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
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	prefix := "create_edges_silent_"
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
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	prefix := "create_edges_nil_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)
	if _, _, err := ec.CreateDocuments(nil, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestCreateEdgesNonSlice creates multiple documents with a non-slice documents input.
func TestCreateEdgesNonSlice(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	prefix := "create_edges_nonSlice_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"citiesPerState", []string{prefix + "city"}, []string{prefix + "state"}, t)

	var obj UserDoc
	if _, _, err := ec.CreateDocuments(nil, &obj); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
	var m map[string]interface{}
	if _, _, err := ec.CreateDocuments(nil, &m); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
