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

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_GraphEdgeDefinitions(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					cols, err := graph.GetEdgeDefinitions(ctx)
					require.NoError(t, err)
					require.Len(t, cols, 0)

					colName := "test_edge_collection"
					colTo := "test_vertex_collection_to"
					colFrom := "test_vertex_collection_from"
					colToNew := "test_vertex_collection_to_new"

					createResp, err := graph.CreateEdgeDefinition(ctx, colName, []string{colFrom}, []string{colTo}, nil)
					require.NoError(t, err)
					require.Empty(t, createResp.GraphDefinition.OrphanCollections)
					require.Len(t, createResp.GraphDefinition.EdgeDefinitions, 1)
					require.Contains(t, createResp.GraphDefinition.EdgeDefinitions[0].Collection, colName)
					require.Contains(t, createResp.GraphDefinition.EdgeDefinitions[0].To, colTo)
					require.Contains(t, createResp.GraphDefinition.EdgeDefinitions[0].From, colFrom)

					exist, err := graph.EdgeDefinitionExists(ctx, colName)
					require.NoError(t, err)
					require.True(t, exist)

					colsAfterCreate, err := graph.GetEdgeDefinitions(ctx)
					require.NoError(t, err)
					require.Len(t, colsAfterCreate, 1)
					require.Equal(t, colName, colsAfterCreate[0].Name())

					colRead, err := graph.EdgeDefinition(ctx, colName)
					require.NoError(t, err)
					require.Equal(t, colName, colRead.Name())

					colToRead, err := graph.VertexCollection(ctx, colTo)
					require.NoError(t, err)
					require.Equal(t, colTo, colToRead.Name())

					colFromRead, err := graph.VertexCollection(ctx, colFrom)
					require.NoError(t, err)
					require.Equal(t, colFrom, colFromRead.Name())

					t.Run("Replacing Edge should not remove the collection", func(t *testing.T) {
						replaceResp, err := graph.ReplaceEdgeDefinition(ctx, colName, []string{colFrom}, []string{colToNew}, nil)
						require.NoError(t, err)
						require.Len(t, replaceResp.GraphDefinition.OrphanCollections, 1)
						require.Contains(t, replaceResp.GraphDefinition.OrphanCollections, colTo)

						require.Len(t, replaceResp.GraphDefinition.EdgeDefinitions, 1)
						require.Contains(t, replaceResp.GraphDefinition.EdgeDefinitions[0].To, colToNew)

						exist, err := graph.VertexCollectionExists(ctx, colTo)
						require.NoError(t, err)
						require.True(t, exist)
					})

					t.Run("Deleting Edge should not remove the collection", func(t *testing.T) {
						delResp, err := graph.DeleteEdgeDefinition(ctx, colName, nil)
						require.NoError(t, err)
						require.NotContains(t, delResp.GraphDefinition.EdgeDefinitions, colName)

						exist, err := graph.VertexCollectionExists(ctx, colToNew)
						require.NoError(t, err)
						require.True(t, exist)

						exist, err = graph.VertexCollectionExists(ctx, colFrom)
						require.NoError(t, err)
						require.True(t, exist)

						col, err := db.Collection(ctx, colName)
						require.NoError(t, err)

						prop, err := col.Properties(ctx)
						require.NoError(t, err)
						require.False(t, prop.IsSatellite())
					})
				})
			})
		})
	})
}

func TestGraphEdgeDefinitionsRemovalCollections(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		skipNoEnterprise(client, context.Background(), t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			gDef := sampleGraphWithEdges(db)
			WithGraph(t, db, gDef, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					t.Run("Deleting Edge should remove the collection when requested", func(t *testing.T) {
						edgeDef := gDef.EdgeDefinitions[0]

						opts := arangodb.DeleteEdgeDefinitionOptions{
							DropCollection: utils.NewType(true),
						}
						delResp, err := graph.DeleteEdgeDefinition(ctx, edgeDef.Collection, &opts)
						require.NoError(t, err)
						require.NotContains(t, delResp.GraphDefinition.OrphanCollections, edgeDef.Collection)
						require.Contains(t, delResp.GraphDefinition.OrphanCollections, edgeDef.To[0])
						require.Contains(t, delResp.GraphDefinition.OrphanCollections, edgeDef.From[0])

						exist, err := db.CollectionExists(ctx, edgeDef.Collection)
						require.NoError(t, err)
						require.False(t, exist, "collection should not exist")
					})
				})
			})
		})
	})
}

func TestGraphEdgeDefinitionsWithSatellites(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		skipNoEnterprise(client, context.Background(), t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

					colName := "create_sat_edge_collection"
					colFromName := "sat_edge_collectionFrom"
					colToName := "sat_edge_collectionTo"

					opts := arangodb.CreateEdgeDefinitionOptions{
						Satellites: []string{colFromName},
					}
					createResp, err := graph.CreateEdgeDefinition(ctx, colName, []string{colFromName}, []string{colToName}, &opts)
					require.NoError(t, err)
					require.Len(t, createResp.GraphDefinition.EdgeDefinitions, 1)

					col, err := db.Collection(ctx, colName)
					require.NoError(t, err)

					prop, err := col.Properties(ctx)
					require.NoError(t, err)
					require.True(t, prop.IsSatellite())

					colFrom, err := db.Collection(ctx, colFromName)
					require.NoError(t, err)

					propFrom, err := colFrom.Properties(ctx)
					require.NoError(t, err)
					require.True(t, propFrom.IsSatellite())

					t.Run("Replace Satellite", func(t *testing.T) {
						newColName := "new_sat_edge_collection_new"
						opts := arangodb.ReplaceEdgeOptions{
							Satellites: []string{newColName},
						}
						delResp, err := graph.ReplaceEdgeDefinition(ctx, colName, []string{newColName}, []string{colToName}, &opts)
						require.NoError(t, err)
						require.Contains(t, delResp.GraphDefinition.EdgeDefinitions[0].From, newColName)
						require.Contains(t, delResp.GraphDefinition.OrphanCollections, colFromName)

						colNew, err := db.Collection(ctx, newColName)
						require.NoError(t, err)

						propNew, err := colNew.Properties(ctx)
						require.NoError(t, err)
						require.True(t, propNew.IsSatellite())
					})
				})
			})
		})
	})
}
