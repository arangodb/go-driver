//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_GraphSimple(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			gDef := sampleGraphWithEdges(db)
			gDef.ReplicationFactor = 2
			gDef.WriteConcern = newInt(2)
			WithGraph(t, db, gDef, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					exist, err := db.GraphExists(ctx, graph.Name())
					require.NoError(t, err)
					require.True(t, exist, "graph should exist")

					graphs, err := db.Graphs(ctx)
					require.NoError(t, err)
					g1, err := graphs.Read()
					require.NoError(t, err)
					require.Equal(t, graph.Name(), g1.Name())
					_, err = graphs.Read()
					require.Error(t, err, "should be no more graphs")

					g, err := db.Graph(ctx, graph.Name(), nil)
					require.NoError(t, err)

					require.Equal(t, graph.Name(), g.Name())
					require.Equal(t, graph.EdgeDefinitions(), g.EdgeDefinitions())
					require.Equal(t, graph.OrphanCollections(), g.OrphanCollections())
					require.Equal(t, graph.SmartGraphAttribute(), g.SmartGraphAttribute())
					require.Equal(t, graph.NumberOfShards(), g.NumberOfShards())
					require.Equal(t, graph.ReplicationFactor(), g.ReplicationFactor())
					require.Equal(t, graph.WriteConcern(), g.WriteConcern())
					require.Equal(t, graph.IsSmart(), g.IsSmart())
					require.False(t, g.IsSatellite())

					t.Run("Test created collections", func(t *testing.T) {
						for _, c := range append(g.EdgeDefinitions()[0].To, g.EdgeDefinitions()[0].From...) {
							col, err := db.Collection(ctx, c)
							require.NoError(t, err)
							require.NotNil(t, col)

							prop, err := col.Properties(ctx)
							require.NoError(t, err)
							require.Equal(t, *g.NumberOfShards(), prop.NumberOfShards)
							require.Equal(t, g.ReplicationFactor(), int(prop.ReplicationFactor))
							require.Equal(t, *g.WriteConcern(), prop.WriteConcern)
						}
					})
					require.NoError(t, g.Remove(ctx, nil))

					exist, err = db.GraphExists(ctx, graph.Name())
					require.NoError(t, err)
					require.False(t, exist, "graph should not exist")
				})
			})
		})
	})
}

func Test_GraphCreation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		skipNoEnterprise(client, context.Background(), t)
		WithDatabase(t, client, nil, func(db arangodb.Database) {

			t.Run("Satellite", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					gDef := sampleSmartGraph()
					gDef.ReplicationFactor = arangodb.SatelliteGraph
					gDef.IsSmart = false
					gDef.SmartGraphAttribute = ""
					gDef.NumberOfShards = newInt(1)

					g, err := db.CreateGraph(ctx, db.Name()+"_sat", gDef, nil)
					require.NoError(t, err)
					require.NotNil(t, g)
					require.Equal(t, arangodb.SatelliteGraph, g.ReplicationFactor())
					require.Equal(t, gDef.IsSmart, g.IsSmart())
					require.Equal(t, gDef.SmartGraphAttribute, g.SmartGraphAttribute())
					require.True(t, g.IsSatellite())
					require.NoError(t, g.Remove(ctx, nil))
				})
			})

			t.Run("Disjoint", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					gDef := sampleSmartGraph()
					gDef.IsDisjoint = true

					g, err := db.CreateGraph(ctx, db.Name()+"_disjoint", gDef, nil)
					require.NoError(t, err)
					require.NotNil(t, g)
					require.True(t, g.IsDisjoint())
					require.Equal(t, gDef.IsSmart, g.IsSmart())
					require.Equal(t, gDef.SmartGraphAttribute, g.SmartGraphAttribute())
					require.NoError(t, g.Remove(ctx, nil))
				})
			})

			t.Run("Hybrid", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipNoEnterprise(client, ctx, t)

					colHybrid := db.Name() + "_create_hybrid_edge_col"
					colSat := db.Name() + "_sat_edge_col"
					colNonSat := db.Name() + "_non_sat_edge_col"

					gDef := &arangodb.GraphDefinition{
						OrphanCollections: []string{"orphan1", "orphan2"},
						EdgeDefinitions: []arangodb.EdgeDefinition{
							{
								Collection: colHybrid,
								To:         []string{colSat},
								From:       []string{colNonSat},
							},
						},
						NumberOfShards:      newInt(2),
						SmartGraphAttribute: "test",
						IsSmart:             true,
						ReplicationFactor:   2,
					}
					opts := &arangodb.CreateGraphOptions{
						Satellites: []string{colHybrid, colSat},
					}

					g, err := db.CreateGraph(ctx, db.Name()+"_hybrid", gDef, opts)
					require.NoError(t, err)
					require.NotNil(t, g)
					require.True(t, g.IsSmart())

					for _, c := range []string{colHybrid, colSat, colNonSat} {
						col, err := db.Collection(ctx, c)
						require.NoError(t, err)
						require.NotNil(t, col)

						prop, err := col.Properties(ctx)
						require.NoError(t, err)

						if c == colNonSat {
							require.Equal(t, 2, int(prop.ReplicationFactor))
							require.Equal(t, 2, prop.NumberOfShards)
							require.False(t, prop.IsSatellite())
						} else {
							require.True(t, prop.IsSatellite())
						}
					}

					require.NoError(t, g.Remove(ctx, nil))
				})
			})
		})
	})
}

func Test_GraphHybridSmartGraphCreationConditions(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		skipNoEnterprise(client, context.Background(), t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			testCases := []struct {
				Name                string
				IsSmart             bool
				SmartGraphAttribute string
			}{
				{
					Name:                db.Name() + "_graph_smart_no_conditions",
					IsSmart:             false,
					SmartGraphAttribute: "",
				},
				{
					Name:                db.Name() + "_graph_smart_conditions",
					IsSmart:             true,
					SmartGraphAttribute: "test",
				},
				{
					Name:                db.Name() + "_graph_smart_conditions_no_attribute",
					IsSmart:             true,
					SmartGraphAttribute: "",
				},
			}

			for _, tc := range testCases {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					gDef := &arangodb.GraphDefinition{
						IsSmart:             tc.IsSmart,
						SmartGraphAttribute: tc.SmartGraphAttribute,
					}

					g, err := db.CreateGraph(ctx, tc.Name, gDef, nil)
					require.NoError(t, err)
					require.NotNil(t, g)
					require.Equal(t, tc.IsSmart, g.IsSmart())
					require.Equal(t, tc.SmartGraphAttribute, g.SmartGraphAttribute())
				})
			}
		})
	})
}

func Test_GraphRemoval(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {

			t.Run("Deleting graph should not remove the collection", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					orphanColName := "col-not-remove"

					gDef := sampleSmartGraph()
					gDef.OrphanCollections = []string{orphanColName}

					g, err := db.CreateGraph(ctx, db.Name()+"_del", gDef, nil)
					require.NoError(t, err)

					err = g.Remove(ctx, nil)
					require.NoError(t, err)

					exist, err := db.CollectionExists(ctx, orphanColName)
					require.NoError(t, err)
					require.True(t, exist, "orphan collection should exist")
				})
			})

			t.Run("Deleting graph should remove the collection", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					orphanColName := "col-remove"

					gDef := sampleSmartGraph()
					gDef.OrphanCollections = []string{orphanColName}

					g, err := db.CreateGraph(ctx, db.Name()+"_del", gDef, nil)
					require.NoError(t, err)

					opts := arangodb.RemoveGraphOptions{
						DropCollections: true,
					}
					err = g.Remove(ctx, &opts)
					require.NoError(t, err)

					exist, err := db.CollectionExists(ctx, orphanColName)
					require.NoError(t, err)
					require.False(t, exist, "orphan collection should not exist")
				})
			})
		})
	})
}
