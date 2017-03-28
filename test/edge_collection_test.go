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

// ensureEdgeCollection returns the edge collection with given name, creating it if needed.
func ensureEdgeCollection(ctx context.Context, g driver.Graph, collection string, from, to []string, t *testing.T) driver.Collection {
	ec, _, err := g.EdgeCollection(ctx, collection)
	if driver.IsNotFound(err) {
		ec, err := g.CreateEdgeCollection(ctx, collection, driver.VertexConstraints{From: from, To: to})
		if err != nil {
			t.Fatalf("Failed to create edge collection: %s", describe(err))
		}
		return ec
	} else if err != nil {
		t.Fatalf("Failed to open edge collection: %s", describe(err))
	}
	return ec
}

// TestCreateEdgeCollection creates a graph and then adds an edge collection in it
func TestCreateEdgeCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "edge_collection_test", nil, t)
	name := "test_create_edge_collection"
	g, err := db.CreateGraph(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// List edge collections, must be empty
	if list, _, err := g.EdgeCollections(nil); err != nil {
		t.Errorf("EdgeCollections failed: %s", describe(err))
	} else if len(list) > 0 {
		t.Errorf("EdgeCollections return %d edge collections, expected 0", len(list))
	}

	// Now create an edge collection
	colName := "create_edge_collection_friends"
	if ec, err := g.CreateEdgeCollection(nil, colName, driver.VertexConstraints{From: []string{"person"}, To: []string{"person"}}); err != nil {
		t.Errorf("CreateEdgeCollection failed: %s", describe(err))
	} else if ec.Name() != colName {
		t.Errorf("Invalid name, expected '%s', got '%s'", colName, ec.Name())
	}

	assertCollection(nil, db, colName, t)
	assertCollection(nil, db, "person", t)

	// List edge collections, must be contain 'friends'
	if list, _, err := g.EdgeCollections(nil); err != nil {
		t.Errorf("EdgeCollections failed: %s", describe(err))
	} else if len(list) != 1 {
		t.Errorf("EdgeCollections return %d edge collections, expected 1", len(list))
	} else if list[0].Name() != colName {
		t.Errorf("Invalid list[0].name, expected '%s', got '%s'", colName, list[0].Name())
	}

	// Friends edge collection must exits
	if found, err := g.EdgeCollectionExists(nil, colName); err != nil {
		t.Errorf("EdgeCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("EdgeCollectionExists return false, expected true")
	}

	// Open friends edge collection must exits
	if ec, _, err := g.EdgeCollection(nil, colName); err != nil {
		t.Errorf("EdgeCollection failed: %s", describe(err))
	} else if ec.Name() != colName {
		t.Errorf("EdgeCollection return invalid collection, expected '%s', got '%s'", colName, ec.Name())
	}
}

// TestRemoveEdgeCollection creates a graph and then adds an edge collection in it and then removes the edge collection.
func TestRemoveEdgeCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "edge_collection_test", nil, t)
	name := "test_remove_edge_collection"
	g, err := db.CreateGraph(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// Now create an edge collection
	colName := "remove_edge_collection_friends"
	ec, err := g.CreateEdgeCollection(nil, colName, driver.VertexConstraints{From: []string{"person"}, To: []string{"person"}})
	if err != nil {
		t.Fatalf("CreateEdgeCollection failed: %s", describe(err))
	} else if ec.Name() != colName {
		t.Errorf("Invalid name, expected '%s', got '%s'", colName, ec.Name())
	}

	// Friends edge collection must exits
	if found, err := g.EdgeCollectionExists(nil, colName); err != nil {
		t.Errorf("EdgeCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("EdgeCollectionExists return false, expected true")
	}

	// Remove edge collection
	if err := ec.Remove(nil); err != nil {
		t.Errorf("Remove failed: %s", describe(err))
	}

	// Friends edge collection must NOT exits
	if found, err := g.EdgeCollectionExists(nil, colName); err != nil {
		t.Errorf("EdgeCollectionExists failed: %s", describe(err))
	} else if found {
		t.Errorf("EdgeCollectionExists return true, expected false")
	}
}

// TestReplaceEdgeCollection creates a graph and then adds an edge collection in it and then replaces the edge collection.
/*func TestReplaceEdgeCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "edge_collection_test", nil, t)
	name := "test_replace_edge_collection"
	g, err := db.CreateGraph(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// Now create an edge collection
	ec, err := g.CreateEdgeCollection(nil, "friends", []string{"person"}, []string{"person"})
	if err != nil {
		t.Errorf("CreateEdgeCollection failed: %s", describe(err))
	} else if ec.Name() != "friends" {
		t.Errorf("Invalid name, expected 'friends', got '%s'", ec.Name())
	}

	// Friends edge collection must exits
	if found, err := g.EdgeCollectionExists(nil, "friends"); err != nil {
		t.Errorf("EdgeCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("EdgeCollectionExists return false, expected true")
	}

	// Replace edge collection
	if err := ec.Replace(nil, []string{"city"}, []string{"state"}); err != nil {
		t.Errorf("Replace failed: %s", describe(err))
	}

	assertCollection(nil, db, "city", t)
	assertCollection(nil, db, "state", t)
}
*/
