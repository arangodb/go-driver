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

// ensureGraph is a helper to check if a graph exists and create if if needed.
// It will fail the test when an error occurs.
func ensureGraph(ctx context.Context, db driver.Database, name string, options *driver.CreateGraphOptions, t *testing.T) driver.Graph {
	g, err := db.Graph(ctx, name)
	if driver.IsNotFound(err) {
		g, err = db.CreateGraph(ctx, name, options)
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
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "graph_test", nil, t)
	name := "test_create_graph"
	if _, err := db.CreateGraph(nil, name, nil); err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}
	// Graph must exist now
	if found, err := db.GraphExists(nil, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("GraphExists('%s') return false, expected true", name)
	}
	// Graph must be listed
	if list, err := db.Graphs(nil); err != nil {
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
	if g, err := db.Graph(nil, name); err != nil {
		t.Errorf("Graph('%s') failed: %s", name, describe(err))
	} else if g.Name() != name {
		t.Errorf("Graph.Name wrong. Expected '%s', got '%s'", name, g.Name())
	}
}

// TestRemoveGraph creates a graph and then removes it.
func TestRemoveGraph(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "graph_test", nil, t)
	name := "test_remove_graph"
	g, err := db.CreateGraph(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}
	// Graph must exist now
	if found, err := db.GraphExists(nil, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("GraphExists('%s') return false, expected true", name)
	}
	// Now remove it
	if err := g.Remove(nil); err != nil {
		t.Fatalf("Failed to remove graph '%s': %s", name, describe(err))
	}
	// Graph must not exist now
	if found, err := db.GraphExists(nil, name); err != nil {
		t.Errorf("GraphExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("GraphExists('%s') return true, expected false", name)
	}
}
