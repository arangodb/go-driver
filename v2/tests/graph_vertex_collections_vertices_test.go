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

func Test_VerticesSimple(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, nil, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					colName := "test_vertices_simple_collection"
					colVertex, err := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, err)

					peter := UserDoc{
						Name: "Peter",
						Age:  40,
					}
					var key string

					t.Run("Create Vertex", func(t *testing.T) {
						peterResp, err := colVertex.CreateVertex(ctx, peter, nil)
						require.NoError(t, err)
						require.NotEmpty(t, peterResp.Key)
						require.Empty(t, peterResp.New)
						key = peterResp.Key
					})

					t.Run("Get Vertex", func(t *testing.T) {
						result := UserDoc{}
						err := colVertex.GetVertex(ctx, key, &result, nil)
						require.NoError(t, err)
						require.Equal(t, peter, result)
					})

					t.Run("Update Vertex with options", func(t *testing.T) {
						peterUpdate := UserDoc{
							Age: 42,
						}

						response, err := colVertex.UpdateVertex(ctx, key, peterUpdate, nil)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)

						latest := UserDoc{}
						err = colVertex.GetVertex(ctx, key, &latest, nil)
						require.NoError(t, err)
						require.Equal(t, peterUpdate, latest)
					})

					t.Run("Replace Vertex", func(t *testing.T) {
						peterUpdate := UserDoc{
							Age: 52,
						}

						response, err := colVertex.ReplaceVertex(ctx, key, peterUpdate, nil)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)

						latest := UserDoc{}
						err = colVertex.GetVertex(ctx, key, &latest, nil)
						require.NoError(t, err)
						require.Equal(t, peterUpdate, latest)
					})

					t.Run("Remove Vertex", func(t *testing.T) {
						response, err := colVertex.DeleteVertex(ctx, key, nil)
						require.NoError(t, err)
						require.Empty(t, response.Old)
					})
				})

			})
		})
	})
}

func Test_VerticesExtended(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, nil, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					colName := "test_vertices_extended_collection"
					colVertex, colErr := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, colErr)

					john := UserDocWithMeta{
						UserDoc: UserDoc{
							Name: "John",
							Age:  30,
						},
					}

					t.Run("Create Vertex wit returnNew option", func(t *testing.T) {
						require.Empty(t, john.Key)
						opts := arangodb.CreateVertexOptions{
							NewObject: &john,
						}
						johnResp, err := colVertex.CreateVertex(ctx, john.UserDoc, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, johnResp.Key)
						require.NotEmpty(t, johnResp.New)
						require.NotEmpty(t, john.Key)
					})

					t.Run("Update Vertex with options", func(t *testing.T) {
						johnUpdated := UserDocWithMeta{
							UserDoc: UserDoc{
								Age: 32,
							},
						}
						var johnOld UserDocWithMeta

						opts := arangodb.VertexUpdateOptions{
							NewObject: &johnUpdated,
							OldObject: &johnOld,
						}

						response, err := colVertex.UpdateVertex(ctx, john.Key, johnUpdated, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)
						require.NotEmpty(t, response.New)
						require.NotEmpty(t, response.Old)
						require.Equal(t, john, johnOld)
						require.NotEqual(t, john.Age, johnUpdated.Age)
						require.Equal(t, 32, johnUpdated.Age)

						johnLatest := UserDocWithMeta{}
						err = colVertex.GetVertex(ctx, john.Key, &johnLatest, nil)
						require.NoError(t, err)
						require.Equal(t, johnUpdated, johnLatest)
					})

					t.Run("Replace Vertex with options", func(t *testing.T) {
						johnUpdated := UserDocWithMeta{
							UserDoc: UserDoc{
								Age: 55,
							},
						}
						var johnOld UserDocWithMeta

						opts := arangodb.VertexReplaceOptions{
							NewObject: &johnUpdated,
							OldObject: &johnOld,
						}

						response, err := colVertex.ReplaceVertex(ctx, john.Key, johnUpdated, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)
						require.NotEmpty(t, response.New)
						require.NotEmpty(t, response.Old)
						require.NotEqual(t, john.Age, johnUpdated.Age)
						require.Equal(t, 55, johnUpdated.Age)

						johnLatest := UserDocWithMeta{}
						err = colVertex.GetVertex(ctx, john.Key, &johnLatest, nil)
						require.NoError(t, err)
						require.Equal(t, johnUpdated, johnLatest)
					})

					t.Run("Get previous version of Vertex should fail", func(t *testing.T) {
						johnResult := UserDocWithMeta{}
						opts := arangodb.GetVertexOptions{
							Rev: john.Rev,
						}
						err := colVertex.GetVertex(ctx, john.Key, &johnResult, &opts)
						require.Error(t, err)
					})

					t.Run("Delete Vertex", func(t *testing.T) {
						johnLatest := UserDocWithMeta{}
						err := colVertex.GetVertex(ctx, john.Key, &johnLatest, nil)

						var johnOld UserDocWithMeta
						opts := arangodb.DeleteVertexOptions{
							OldObject: &johnOld,
						}
						response, err := colVertex.DeleteVertex(ctx, john.Key, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, response.Old)
						require.Equal(t, johnLatest, johnOld)

					})
				})

			})
		})
	})
}

func Test_VerticesUpdate(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, nil, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					colName := "test_vertices_update_collection"
					colVertex, colErr := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, colErr)

					type Developer struct {
						Name    string `json:"name"`
						IsAdmin *bool  `json:"isAdmin"`
					}

					john := Developer{
						Name:    "John",
						IsAdmin: newBool(true),
					}

					johnResp, err := colVertex.CreateVertex(ctx, john, nil)
					require.NoError(t, err)

					t.Run("Update Vertex with KeepNull=true", func(t *testing.T) {
						johnUpdated := Developer{
							Name: "JohnUpdated",
						}

						opts := arangodb.VertexUpdateOptions{
							NewObject: &johnUpdated,
							KeepNull:  newBool(true),
						}

						response, err := colVertex.UpdateVertex(ctx, johnResp.Key, johnUpdated, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)

						t.Run("Updated vertex should keep the 'isAdmin' nil field", func(t *testing.T) {
							var docRawAfterUpdate map[string]interface{}
							err = colVertex.GetVertex(ctx, johnResp.Key, &docRawAfterUpdate, nil)
							require.NoError(t, err)
							require.Contains(t, docRawAfterUpdate, "isAdmin")
							require.Equal(t, docRawAfterUpdate["isAdmin"], nil)
						})
					})

					t.Run("Update Vertex with KeepNull=false", func(t *testing.T) {
						johnUpdated := Developer{
							Name: "JohnUpdated",
						}

						opts := arangodb.VertexUpdateOptions{
							NewObject: &johnUpdated,
							KeepNull:  newBool(false),
						}

						response, err := colVertex.UpdateVertex(ctx, johnResp.Key, johnUpdated, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, response.Key)

						t.Run("Updated vertex should no have the 'isAdmin' anymore", func(t *testing.T) {
							var docRawAfterUpdate map[string]interface{}
							err = colVertex.GetVertex(ctx, johnResp.Key, &docRawAfterUpdate, nil)
							require.NoError(t, err)
							require.NotContains(t, docRawAfterUpdate, "isAdmin")
						})
					})
				})

			})
		})
	})
}
