//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package test

import (
	"context"
	"testing"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

func newGraphOpts(db driver.Database) (driver.CreateGraphOptions, []string) {
	// Create
	edge1 := db.Name() + "_e1"
	edge2 := db.Name() + "_e2"
	edge3 := db.Name() + "_e3"
	coll1 := db.Name() + "_1"
	coll2 := db.Name() + "_2"
	coll3 := db.Name() + "_2"

	collections := []string{
		coll1, coll2, coll3,
	}

	var ed1 driver.EdgeDefinition
	ed1.Collection = edge1
	ed1.From = []string{coll1}
	ed1.To = []string{coll1, coll2}

	var ed2 driver.EdgeDefinition
	ed2.Collection = edge2
	ed2.From = []string{coll1}
	ed2.To = []string{coll1}

	var ed3 driver.EdgeDefinition
	ed3.Collection = edge3
	ed3.From = []string{coll2}
	ed3.To = []string{coll2}

	var options driver.CreateGraphOptions
	options.OrphanVertexCollections = []string{coll3}
	options.EdgeDefinitions = []driver.EdgeDefinition{ed1, ed2, ed3}
	options.IsSmart = true
	options.SmartGraphAttribute = "key"
	options.NumberOfShards = 3

	return options, collections
}

func waitForCollections(t *testing.T, db driver.Database, collections []string) {
	ctx := context.Background()
	err := retry(125*time.Millisecond, 5*time.Second, func() error {
		for _, collName := range collections {
			coll, err := db.Collection(ctx, collName)
			if err != nil {
				if driver.IsNotFound(err) {
					t.Logf("Collection missing %s", collName)
					return nil
				}
				return err
			}

			props := driver.SetCollectionPropertiesOptions{}

			err = coll.SetProperties(ctx, props)
			require.NoError(t, err)
		}

		return interrupt{}
	})
	require.NoError(t, err)
}

// Test_Graph_AdvancedCreate will check if graph created have properly set replication factor
// and write concern
func Test_Graph_AdvancedCreate(t *testing.T) {
	// Arrange
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	v, err := c.Version(nil)
	require.NoError(t, err)

	skipNoCluster(c, t)

	db := ensureDatabase(ctx, c, databaseName("graph", "create", "replication"), nil, t)

	// Create
	graphID := db.Name() + "_graph"

	options, collections := newGraphOpts(db)
	options.ReplicationFactor = 3
	options.WriteConcern = 2

	_, err = db.CreateGraph(ctx, graphID, &options)
	require.NoError(t, err)

	// Wait for collections to be created
	waitForCollections(t, db, collections)

	t.Run("Ensure all properties are set properly", func(t *testing.T) {
		for _, collName := range collections {
			collection, err := db.Collection(ctx, collName)
			require.NoError(t, err)

			prop, err := collection.Properties(ctx)
			require.NoError(t, err)

			require.Equalf(t, 3, prop.NumberOfShards, "NumberOfShards mismatch for %s", collName)

			require.Equalf(t, 3, prop.ReplicationFactor, "ReplicationFactor mismatch for %s", collName)
			if v.Version.CompareTo("3.6") >= 0 {
				require.Equalf(t, 2, prop.WriteConcern, "WriteConcern mismatch for %s", collName)
			}
		}
	})
}

// Test_Graph_AdvancedCreate_Defaults will check if graph created have properly set replication factor
// and write concern by default
func Test_Graph_AdvancedCreate_Defaults(t *testing.T) {
	// Arrange
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	v, err := c.Version(nil)
	require.NoError(t, err)

	skipNoCluster(c, t)

	db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

	// Create
	graphID := db.Name() + "_graph"

	options, collections := newGraphOpts(db)

	_, err = db.CreateGraph(ctx, graphID, &options)
	require.NoError(t, err)

	// Wait for collections to be created
	waitForCollections(t, db, collections)

	t.Run("Ensure all properties are set properly by default", func(t *testing.T) {
		for _, collName := range collections {
			collection, err := db.Collection(ctx, collName)
			require.NoError(t, err)

			prop, err := collection.Properties(ctx)
			require.NoError(t, err)

			require.Equalf(t, 1, prop.ReplicationFactor, "ReplicationFactor mismatch for %s", collName)
			if v.Version.CompareTo("3.6") >= 0 {
				require.Equalf(t, 1, prop.WriteConcern, "WriteConcern mismatch for %s", collName)
			}
		}
	})
}

func TestGraphCreation(t *testing.T) {
	// Arrange
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.7.0")).Cluster().Enterprise()

	t.Run("Satellite", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		options.ReplicationFactor = driver.SatelliteGraph
		options.IsSmart = false
		options.SmartGraphAttribute = ""

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		require.True(t, g.IsSatellite())
	})

	t.Run("Satellite - list", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		options.ReplicationFactor = driver.SatelliteGraph
		options.IsSmart = false
		options.SmartGraphAttribute = ""

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		graphs, err := db.Graphs(ctx)
		require.NoError(t, err)
		require.Len(t, graphs, 1)

		require.Equal(t, g.Name(), graphs[0].Name())
		require.True(t, graphs[0].IsSatellite())
	})

	t.Run("Standard", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		require.False(t, g.IsSatellite())
	})

	t.Run("Standard - list", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		graphs, err := db.Graphs(ctx)
		require.NoError(t, err)
		require.Len(t, graphs, 1)

		require.Equal(t, g.Name(), graphs[0].Name())
		require.False(t, graphs[0].IsSatellite())
	})

	t.Run("Disjoint", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		options.IsDisjoint = true

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		require.True(t, g.IsDisjoint())
	})

	t.Run("Disjoint - list", func(t *testing.T) {
		db := ensureDatabase(ctx, c, databaseName("graph", "create", "defaults"), nil, t)

		// Create
		graphID := db.Name() + "_graph"

		options, collections := newGraphOpts(db)

		options.IsDisjoint = true

		g, err := db.CreateGraph(ctx, graphID, &options)
		require.NoError(t, err)

		// Wait for collections to be created
		waitForCollections(t, db, collections)

		graphs, err := db.Graphs(ctx)
		require.NoError(t, err)
		require.Len(t, graphs, 1)

		require.Equal(t, g.Name(), graphs[0].Name())
		require.True(t, graphs[0].IsDisjoint())
	})
}

func TestHybridSmartGraphCreation(t *testing.T) {
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.9.0")).Cluster().Enterprise()

	db := ensureDatabase(ctx, c, databaseName("graph", "create", "hybrid"), nil, t)

	name := db.Name() + "_test_create_hybrid_graph"
	colName := db.Name() + "_create_hybrid_edge_col"
	col1Name := db.Name() + "_sat_edge_col"
	col2Name := db.Name() + "_non_sat_edge_col"

	options := driver.CreateGraphOptions{
		IsSmart:             true,
		SmartGraphAttribute: "test",
		ReplicationFactor:   2,
		NumberOfShards:      2,
		Satellites:          []string{colName, col1Name},
		EdgeDefinitions: []driver.EdgeDefinition{{
			Collection: colName,
			From:       []string{col1Name},
			To:         []string{col2Name},
		}},
	}
	g, err := db.CreateGraph(ctx, name, &options)
	if err != nil {
		t.Fatalf("Failed to create graph '%s': %s", name, describe(err))
	}

	graphs, err := db.Graphs(ctx)
	require.NoError(t, err)
	require.Len(t, graphs, 1)

	require.Equal(t, g.Name(), graphs[0].Name())
	require.True(t, graphs[0].IsSmart())

	for _, collName := range []string{colName, col1Name, col2Name} {
		collection, err := db.Collection(ctx, collName)
		require.NoError(t, err)

		prop, err := collection.Properties(ctx)
		require.NoError(t, err)

		if collName == col2Name {
			require.Equalf(t, 2, prop.ReplicationFactor, "ReplicationFactor mismatch for %s", collName)
			require.Equalf(t, 2, prop.NumberOfShards, "NumberOfShards mismatch for %s", collName)
		} else {
			require.True(t, prop.IsSatellite())
		}
	}
}
