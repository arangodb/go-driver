//
// DISCLAIMER
//
// Copyright 2024-2025 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"testing"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_GraphVertexCollections(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					cols, err := graph.VertexCollections(ctx)
					require.NoError(t, err)
					require.Len(t, cols, 0)

					colName := "test_vertex_collection"

					createResp, err := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, err)
					require.Contains(t, createResp.GraphDefinition.OrphanCollections, colName)

					exist, err := graph.VertexCollectionExists(ctx, colName)
					require.NoError(t, err)
					require.True(t, exist, "vertex collection should exist")

					colsAfterCreate, err := graph.VertexCollections(ctx)
					require.NoError(t, err)
					require.Len(t, colsAfterCreate, 1)
					require.Equal(t, colName, colsAfterCreate[0].Name())

					colRead, err := graph.VertexCollection(ctx, colName)
					require.NoError(t, err)
					require.Equal(t, colName, colRead.Name())

					t.Run("Deleting Vertex should not remove the collection", func(t *testing.T) {
						delResp, err := graph.DeleteVertexCollection(ctx, colName, nil)
						require.NoError(t, err)
						require.NotContains(t, delResp.GraphDefinition.OrphanCollections, colName)

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

func Test_AddBulkVerticesToCollection(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// skipNoEnterprise(client, ctx, t)

					type DocVertex struct {
						Key   string  `json:"_key,omitempty"`
						Value string  `json:"value"`
						Lat   float32 `json:"latitude"`
						Lon   float32 `json:"longitude"`
					}

					docs := []DocVertex{
						{
							Key:   "111",
							Value: "Value1",
							Lat:   1,
							Lon:   0,
						},
						{
							Key:   "222",
							Value: "Value2",
							Lat:   50,
							Lon:   0,
						},
						{
							Key:   "333",
							Value: "Value3",
							Lat:   10,
							Lon:   0,
						},
					}

					colName := "test_vertex_collection_add_many"
					createResp, err := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, err)
					require.Contains(t, createResp.GraphDefinition.OrphanCollections, colName)

					idxOpts := arangodb.CreateGeoIndexOptions{GeoJSON: utils.NewType(false)}
					col := createResp.VertexCollection
					col.AsCollection().EnsureGeoIndex(ctx, []string{"latitude", "longitude"}, &idxOpts)
					_, err = col.AsCollection().CreateDocuments(ctx, docs)
					require.NoError(t, err)

					QUERY := fmt.Sprintf("FOR x IN %v FILTER DISTANCE(0, 0, x.latitude, x.longitude) <= 1120000 RETURN x", colName)
					cursor, err := db.Query(ctx, QUERY, nil)
					require.NoError(t, err)

					var vertRead1, vertRead2 DocVertex
					_, err = cursor.ReadDocument(ctx, &vertRead1)
					require.NoError(t, err)
					_, err = cursor.ReadDocument(ctx, &vertRead2)
					require.NoError(t, err)
					require.ElementsMatch(t, []string{"Value1", "Value3"}, []string{vertRead1.Value, vertRead2.Value})
					cursor.Close()

					err = col.GetVertex(ctx, "111", &vertRead1, nil)
					require.NoError(t, err)
					require.Equal(t, "Value1", vertRead1.Value)

				})
			})
		})

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// skipNoEnterprise(client, ctx, t)

					type DocVertex struct {
						Key   string `json:"_key,omitempty"`
						Value string `json:"value"`
					}

					docs := []DocVertex{
						{
							Key:   "111",
							Value: "Value1",
						},
						{
							Key:   "222",
							Value: "Value2",
						},
						{
							Key:   "333",
							Value: "Value3",
						},
						{
							Key:   "444",
							Value: "Value4",
						},
					}
					vColName := "test_vertex_collection_add_many"
					createVertResp, err := graph.CreateVertexCollection(ctx, vColName, nil)
					require.NoError(t, err)
					vCol := createVertResp.VertexCollection
					_, err = vCol.AsCollection().CreateDocuments(ctx, docs)
					require.NoError(t, err)
					require.Contains(t, createVertResp.GraphDefinition.OrphanCollections, vColName)

					type DocEdge struct {
						From string `json:"_from"`
						To   string `json:"_to"`
					}

					px := vColName + "/"
					edges := []DocEdge{
						{
							From: px + "111",
							To:   px + "222",
						},
						{
							From: px + "222",
							To:   px + "333",
						},
						{
							From: px + "333",
							To:   px + "444",
						},
						{
							From: px + "444",
							To:   px + "111",
						},
						{
							From: px + "222",
							To:   px + "111",
						},
						{
							From: px + "333",
							To:   px + "222",
						},
						{
							From: px + "444",
							To:   px + "333",
						},
					}

					eColName := "test_edge_collection_add_many"
					createEdgeResp, err := graph.CreateEdgeDefinition(ctx, eColName, []string{vColName}, []string{vColName}, nil)
					require.NoError(t, err)
					require.NotContains(t, createEdgeResp.GraphDefinition.OrphanCollections, vColName)
					eCol := createEdgeResp.Edge
					_, err = eCol.AsCollection().CreateDocuments(ctx, edges)
					require.NoError(t, err)

					var meta arangodb.DocumentMeta
					_ = vCol.GetVertex(ctx, "111", &meta, nil)
					QUERY := fmt.Sprintf(
						"FOR v, e, p IN 1..1 OUTBOUND \"%v\" GRAPH \"%v\" RETURN CONCAT_SEPARATOR(\"--\", p.vertices[*].value)",
						meta.ID,
						graph.Name(),
					)
					cursor, err := db.Query(ctx, QUERY, nil)
					require.NoError(t, err)

					var pathRead string
					_, err = cursor.ReadDocument(ctx, &pathRead)
					require.NoError(t, err)
					require.Equal(t, "Value1--Value2", pathRead)

					_ = vCol.GetVertex(ctx, "444", &meta, nil)
					QUERY = fmt.Sprintf(
						"FOR v, e, p IN 1..1 OUTBOUND \"%v\" GRAPH \"%v\" RETURN CONCAT_SEPARATOR(\"--\", p.vertices[*].value)",
						meta.ID,
						graph.Name(),
					)
					cursor, err = db.Query(ctx, QUERY, nil)
					require.NoError(t, err)

					var pathRead2 string
					_, err = cursor.ReadDocument(ctx, &pathRead)
					require.NoError(t, err)
					_, err = cursor.ReadDocument(ctx, &pathRead2)
					require.NoError(t, err)
					require.ElementsMatch(t, []string{"Value4--Value1", "Value4--Value3"}, []string{pathRead, pathRead2})
				})
			})
		})
	})
}

func TestCreateSatelliteVertexCollection(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		requireClusterMode(t)
		skipNoEnterprise(client, context.Background(), t)

		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleSmartGraph(), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipNoEnterprise(client, ctx, t)
					cols, err := graph.VertexCollections(ctx)
					require.NoError(t, err)
					require.Len(t, cols, 0)

					colName := "test_vertex_-sat-collection"
					opts := arangodb.CreateVertexCollectionOptions{
						Satellites: []string{colName},
					}

					createResp, err := graph.CreateVertexCollection(ctx, colName, &opts)
					require.NoError(t, err)
					require.Contains(t, createResp.GraphDefinition.OrphanCollections, colName)

					exist, err := graph.VertexCollectionExists(ctx, colName)
					require.NoError(t, err)
					require.True(t, exist, "vertex collection should exist")

					col, err := db.Collection(ctx, colName)
					require.NoError(t, err)

					prop, err := col.Properties(ctx)
					require.NoError(t, err)
					require.True(t, prop.IsSatellite())

					t.Run("Deleting Vertex should remove the collection when requested", func(t *testing.T) {
						opts := arangodb.DeleteVertexCollectionOptions{
							DropCollection: utils.NewType(true),
						}
						delResp, err := graph.DeleteVertexCollection(ctx, colName, &opts)
						require.NoError(t, err)
						require.NotContains(t, delResp.GraphDefinition.OrphanCollections, colName)

						exist, err := db.CollectionExists(ctx, colName)
						require.NoError(t, err)
						require.False(t, exist, "collection should not exist")
					})
				})
			})
		})
	})
}
