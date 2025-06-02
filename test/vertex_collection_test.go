//
// DISCLAIMER
//
// Copyright 2017-2024 ArangoDB GmbH, Cologne, Germany
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

// ensureVertexCollection returns the vertex collection with given name, creating it if needed.
func ensureVertexCollection(ctx context.Context, g driver.Graph, collection string, t *testing.T) driver.Collection {
	ec, err := g.VertexCollection(ctx, collection)
	if driver.IsNotFound(err) {
		ec, err := g.CreateVertexCollection(ctx, collection)
		if err != nil {
			t.Fatalf("Failed to create vertex collection: %s", describe(err))
		}
		return ec
	} else if err != nil {
		t.Fatalf("Failed to open vertex collection: %s", describe(err))
	}
	return ec
}

// TestCreateVertexCollection creates a graph and then adds a vertex collection in it
func TestCreateVertexCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "vertex_collection_test", nil, t)
	name := "test_create_vertex_collection"
	g, err := db.CreateGraphV2(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// List vertex collections, must be empty
	if list, err := g.VertexCollections(nil); err != nil {
		t.Errorf("VertexCollections failed: %s", describe(err))
	} else if len(list) > 0 {
		t.Errorf("VertexCollections return %d vertex collections, expected 0", len(list))
	}

	// Now create a vertex collection
	if vc, err := g.CreateVertexCollection(nil, "person"); err != nil {
		t.Errorf("CreateVertexCollection failed: %s", describe(err))
	} else if vc.Name() != "person" {
		t.Errorf("Invalid name, expected 'person', got '%s'", vc.Name())
	}

	// List vertex collections, must be contain 'person'
	if list, err := g.VertexCollections(nil); err != nil {
		t.Errorf("VertexCollections failed: %s", describe(err))
	} else if len(list) != 1 {
		t.Errorf("VertexCollections return %d vertex collections, expected 1", len(list))
	} else if list[0].Name() != "person" {
		t.Errorf("Invalid list[0].name, expected 'person', got '%s'", list[0].Name())
	}

	// Person vertex collection must exits
	if found, err := g.VertexCollectionExists(nil, "person"); err != nil {
		t.Errorf("VertexCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("VertexCollectionExists return false, expected true")
	}

	// Open person vertex collection must exits
	if vc, err := g.VertexCollection(nil, "person"); err != nil {
		t.Errorf("VertexCollection failed: %s", describe(err))
	} else if vc.Name() != "person" {
		t.Errorf("VertexCollection return invalid collection, expected 'person', got '%s'", vc.Name())
	}
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestCreateSatelliteVertexCollection creates a graph and then adds a Satellite vertex collection in it
func TestCreateSatelliteVertexCollection(t *testing.T) {
	ctx := context.Background()

	c := createClient(t, nil)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.9.0")).Cluster().Enterprise()

	db := ensureDatabase(ctx, c, "vertex_collection_test", nil, t)

	name := "test_create_satellite_vertex_collection"
	options := driver.CreateGraphOptions{
		IsSmart:             true,
		SmartGraphAttribute: "key",
	}
	g, err := db.CreateGraphV2(ctx, name, &options)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// List vertex collections, must be empty
	if list, err := g.VertexCollections(ctx); err != nil {
		t.Errorf("VertexCollections failed: %s", describe(err))
	} else if len(list) > 0 {
		t.Errorf("VertexCollections return %d vertex collections, expected 0", len(list))
	}

	satelliteName := "vertex-sat-test"
	opt := driver.CreateVertexCollectionOptions{Satellites: []string{satelliteName}}

	// Now create a vertex collection
	if vc, err := g.CreateVertexCollectionWithOptions(ctx, satelliteName, opt); err != nil {
		t.Errorf("CreateVertexCollection failed: %s", describe(err))
	} else if vc.Name() != satelliteName {
		t.Errorf("Invalid name, expected 'vertex-sat-test', got '%s'", vc.Name())
	}

	// List vertex collections, must be contain 'person'
	if list, err := g.VertexCollections(ctx); err != nil {
		t.Errorf("VertexCollections failed: %s", describe(err))
	} else if len(list) != 1 {
		t.Errorf("VertexCollections return %d vertex collections, expected 1", len(list))
	} else if list[0].Name() != satelliteName {
		t.Errorf("Invalid list[0].name, expected 'vertex-sat-test', got '%s'", list[0].Name())
	} else {
		prop, err := list[0].Properties(ctx)
		if err != nil {
			t.Errorf("VertexCollections Properties failed: %s", describe(err))
		}
		if !prop.IsSatellite() {
			t.Errorf("Collection %s is not satellite", satelliteName)
		}
	}

	// revert
	err = g.Remove(ctx)
	require.NoError(t, err)
	err = db.Remove(ctx)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRemoveVertexCollection creates a graph and then adds an vertex collection in it and then removes the vertex collection.
func TestRemoveVertexCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "vertex_collection_test", nil, t)
	name := "test_remove_vertex_collection"
	g, err := db.CreateGraphV2(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// Now create an vertex collection
	vc, err := g.CreateVertexCollection(nil, "friends")
	if err != nil {
		t.Errorf("CreateVertexCollection failed: %s", describe(err))
	} else if vc.Name() != "friends" {
		t.Errorf("Invalid name, expected 'friends', got '%s'", vc.Name())
	}

	// Friends vertex collection must exits
	if found, err := g.VertexCollectionExists(nil, "friends"); err != nil {
		t.Errorf("VertexCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("VertexCollectionExists return false, expected true")
	}

	// Remove vertex collection
	if err := vc.Remove(nil); err != nil {
		t.Errorf("Remove failed: %s", describe(err))
	}

	// Friends vertex collection must NOT exits
	if found, err := g.VertexCollectionExists(nil, "friends"); err != nil {
		t.Errorf("VertexCollectionExists failed: %s", describe(err))
	} else if found {
		t.Errorf("VertexCollectionExists return true, expected false")
	}

	// Collection must still exist in database
	assertCollection(nil, db, "friends", t)
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}

// TestRenameVertexCollection creates a graph and then adds an vertex collection in it and then renames the vertex collection.
func TestRenameVertexCollection(t *testing.T) {
	c := createClient(t, nil)

	//Run only in single server
	skipNoSingle(c, t)

	db := ensureDatabase(nil, c, "vertex_collection_test", nil, t)
	name := "test_rename_vertex_collection"
	g, err := db.CreateGraphV2(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	// Now create an vertex collection
	colName := "friends"
	vc, err := g.CreateVertexCollection(nil, colName)
	if err != nil {
		t.Errorf("CreateVertexCollection failed: %s", describe(err))
	} else if vc.Name() != "friends" {
		t.Errorf("Invalid name, expected 'friends', got '%s'", vc.Name())
	}

	// Friends vertex collection must exits
	if found, err := g.VertexCollectionExists(nil, colName); err != nil {
		t.Errorf("VertexCollectionExists failed: %s", describe(err))
	} else if !found {
		t.Errorf("VertexCollectionExists return false, expected true")
	}

	// Rename vertex collection
	newName := "old_friends"
	if err := vc.Rename(nil, newName); err != nil {
		t.Errorf("Rename failed: %s", describe(err))
	}

	// Friends vertex collection must NOT exits
	if found, err := g.VertexCollectionExists(nil, colName); err != nil {
		t.Errorf("VertexCollectionExists failed: %s", describe(err))
	} else if found {
		t.Errorf("VertexCollectionExists return true, expected false")
	}

	// Collection must still exist in database
	assertCollection(nil, db, newName, t)
	err = db.Remove(nil)
	if err != nil {
		t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
	}
}
