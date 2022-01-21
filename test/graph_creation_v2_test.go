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

package test

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

// Test_Graph_AdvancedCreateV2 will check if graph created have properly set replication factor and write concern
func Test_Graph_AdvancedCreateV2(t *testing.T) {
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

	_, err = db.CreateGraphV2(ctx, graphID, &options)
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

// Test_Graph_AdvancedCreateV2_Defaults will check if graph created have properly set replication factor and write concern by default
func Test_Graph_AdvancedCreateV2_Defaults(t *testing.T) {
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

	_, err = db.CreateGraphV2(ctx, graphID, &options)
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

func TestGraphCreationV2(t *testing.T) {
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

		g, err := db.CreateGraphV2(ctx, graphID, &options)
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

func TestHybridSmartGraphCreationV2(t *testing.T) {
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
	g, err := db.CreateGraphV2(ctx, name, &options)
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
