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
	"github.com/stretchr/testify/require"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// ensureGraph is a helper to check if a graph exists and create if if needed.
// It will fail the test when an error occurs.
func ensureGraph(ctx context.Context, db driver.Database, name string, options *driver.CreateGraphOptions, t *testing.T) driver.Graph {
	g, err := db.Graph(ctx, name)
	if driver.IsNotFound(err) {
		g, err = db.CreateGraphV2(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open graph '%s': %s", name, describe(err))
	}
	return g
}

// TestCreateGraph creates a graph and then checks that it exists.
func TestCreateGraph(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "graph_test", nil, t)
	name := "test_create_graph"

	if _, err := db.CreateGraphV2(ctx, name, nil); err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}
	// Graph must exist now
	if found, err := db.GraphExists(ctx, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("GraphExists('%s') return false, expected true", name)
	}
	// Graph must be listed
	if list, err := db.Graphs(ctx); err != nil {
		t.Errorf("Graphs failed: %s", describe(err))
	} else {
		found := false
		for _, g := range list {
			if g.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Graph '%s' not found in list", name)
		}
	}
	// Open graph
	if g, err := db.Graph(ctx, name); err != nil {
		t.Errorf("Graph('%s') failed: %s", name, describe(err))
	} else if g.Name() != name {
		t.Errorf("Graph.Name wrong. Expected '%s', got '%s'", name, g.Name())
	}
}

// TestCreateGraphWithOptions creates a graph with options then checks if each options is set correctly.
func TestCreateGraphWithOptions(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)

	db := ensureDatabase(ctx, c, "graph_test", nil, t)
	name := "test_create_graph_2"

	options := &driver.CreateGraphOptions{
		OrphanVertexCollections: []string{"orphan1", "orphan2"},
		EdgeDefinitions: []driver.EdgeDefinition{
			{
				Collection: "coll",
				To:         []string{"to-coll1"},
				From:       []string{"from-coll1"},
			},
		},
		NumberOfShards:      2,
		ReplicationFactor:   3,
		WriteConcern:        2,
		SmartGraphAttribute: "orphan1",
	}

	if _, err := db.CreateGraphV2(ctx, name, options); err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}
	// Graph must exist now
	if found, err := db.GraphExists(ctx, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("GraphExists('%s') return false, expected true", name)
	}
	// Graph must be listed
	if list, err := db.Graphs(ctx); err != nil {
		t.Errorf("Graphs failed: %s", describe(err))
	} else {
		found := false
		for _, g := range list {
			if g.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Graph '%s' not found in list", name)
		}
	}

	// Open graph
	g, err := db.Graph(ctx, name)
	if err != nil {
		t.Errorf("Graph('%s') failed: %s", name, describe(err))
	} else if g.Name() != name {
		t.Errorf("Graph.Name wrong. Expected '%s', got '%s'", name, g.Name())
	}

	if g.NumberOfShards() != options.NumberOfShards {
		t.Errorf("Graph.NumberOfShards wrong. Expected '%d', got '%d'", options.NumberOfShards, g.NumberOfShards())
	}
	if g.ReplicationFactor() != options.ReplicationFactor {
		t.Errorf("Graph.ReplicationFactor wrong. Expected '%d', got '%d'", options.ReplicationFactor, g.ReplicationFactor())
	}
	if g.WriteConcern() != options.WriteConcern {
		t.Errorf("Graph.WriteConcern wrong. Expected '%d', got '%d'", options.WriteConcern, g.WriteConcern())
	}
	if g.EdgeDefinitions()[0].Collection != options.EdgeDefinitions[0].Collection {
		t.Errorf("Graph.EdgeDefinitions.collection wrong. Expected '%s', got '%s'", options.EdgeDefinitions[0].Collection, g.EdgeDefinitions()[0].Collection)
	}
	if g.EdgeDefinitions()[0].From[0] != options.EdgeDefinitions[0].From[0] {
		t.Errorf("Graph.EdgeDefinitions.from wrong. Expected '%s', got '%s'", options.EdgeDefinitions[0].From[0], g.EdgeDefinitions()[0].From[0])
	}
	if g.EdgeDefinitions()[0].To[0] != options.EdgeDefinitions[0].To[0] {
		t.Errorf("Graph.EdgeDefinitions.to wrong. Expected '%s', got '%s'", options.EdgeDefinitions[0].To[0], g.EdgeDefinitions()[0].To[0])
	}
	if g.OrphanCollections()[0] != options.OrphanVertexCollections[0] && g.OrphanCollections()[1] != options.OrphanVertexCollections[1] {
		t.Errorf("Graph.IsSmart wrong. Expected '%v', got '%v'", options.OrphanVertexCollections, g.OrphanCollections())
	}
}

// TestRemoveGraph creates a graph and then removes it.
func TestRemoveGraph(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "graph_test", nil, t)
	name := "test_remove_graph"
	g, err := db.CreateGraphV2(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}
	// Graph must exist now
	if found, err := db.GraphExists(ctx, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("GraphExists('%s') return false, expected true", name)
	}
	// Now remove it
	if err := g.Remove(ctx); err != nil {
		t.Fatalf("Failed to remove graph '%s': %s", name, describe(err))
	}
	// Graph must not exist now
	if found, err := db.GraphExists(ctx, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("GraphExists('%s') return true, expected false", name)
	}
}

// TestRemoveGraphWithOpts creates a graph with collection and then removes it.
func TestRemoveGraphWithOpts(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "graph_test_remove", nil, t)
	name := "test_remove_graph_opts"
	colName := "remove_graph_col"

	g, err := db.CreateGraphV2(ctx, name, nil)
	require.NoError(t, err)

	found, err := db.GraphExists(ctx, name)
	require.NoError(t, err)
	require.True(t, found)

	// Now create a vertex collection
	vc, err := g.CreateVertexCollection(nil, colName)
	require.NoError(t, err)
	require.Equal(t, colName, vc.Name())

	// Now remove the graph with collections
	err = g.RemoveWithOpts(ctx, &driver.RemoveGraphOptions{DropCollections: true})
	require.NoError(t, err)

	// Collection must not exist in a database
	colExist, err := db.CollectionExists(ctx, name)
	require.NoError(t, err)
	require.False(t, colExist)

	found, err = db.GraphExists(ctx, name)
	require.NoError(t, err)
	require.False(t, found)
}
