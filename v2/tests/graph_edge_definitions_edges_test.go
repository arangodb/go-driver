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

func Test_EdgeSimple(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, nil, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					edgeColName := "citiesPerState"
					toColName := "states"
					fromColName := "cities"

					toVertex := ensureVertex(t, ctx, graph, toColName, Place{Name: "Texas"})
					fromVertex := ensureVertex(t, ctx, graph, fromColName, Place{Name: "Houston"})

					edgeDefResp, err := graph.CreateEdgeDefinition(ctx, edgeColName, []string{fromColName}, []string{toColName}, nil)
					require.NoError(t, err)
					require.Len(t, edgeDefResp.GraphDefinition.EdgeDefinitions, 1)

					doc := RouteEdge{
						From:     string(fromVertex.ID),
						To:       string(toVertex.ID),
						Distance: 7,
					}
					var docKey string

					t.Run("Create Edge", func(t *testing.T) {
						opts := arangodb.CreateEdgeOptions{
							WaitForSync: newBool(true),
						}
						createEdgeResp, err := edgeDefResp.Edge.CreateEdge(ctx, doc, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, createEdgeResp.Key)

						docKey = createEdgeResp.Key
						docRead := RouteEdge{}
						err = edgeDefResp.Edge.GetEdge(ctx, docKey, &docRead, nil)
						require.NoError(t, err)
						require.Equal(t, doc, docRead)

					})

					t.Run("Update Edge", func(t *testing.T) {
						opts := arangodb.EdgeUpdateOptions{
							WaitForSync: newBool(true),
						}
						updateEdgeResp, err := edgeDefResp.Edge.UpdateEdge(ctx, docKey, map[string]int{"distance": 10}, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, updateEdgeResp.Key)

						docRead := RouteEdge{}
						optsRead := arangodb.GetEdgeOptions{
							Rev: updateEdgeResp.Rev,
						}
						err = edgeDefResp.Edge.GetEdge(ctx, docKey, &docRead, &optsRead)
						require.NoError(t, err)
						require.Equal(t, doc.From, docRead.From)
						require.Equal(t, doc.To, docRead.To)
						require.Equal(t, 10, docRead.Distance)

					})

					t.Run("Replace Edge", func(t *testing.T) {
						opts := arangodb.EdgeReplaceOptions{
							WaitForSync: newBool(true),
						}

						docReplace := RouteEdge{
							From:     string(fromVertex.ID),
							To:       string(toVertex.ID),
							Distance: 12,
						}

						replaceEdgeResp, err := edgeDefResp.Edge.ReplaceEdge(ctx, docKey, docReplace, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, replaceEdgeResp.Key)

						docRead := RouteEdge{}
						optsRead := arangodb.GetEdgeOptions{
							Rev: replaceEdgeResp.Rev,
						}
						err = edgeDefResp.Edge.GetEdge(ctx, docKey, &docRead, &optsRead)
						require.NoError(t, err)
						require.Equal(t, doc.From, docRead.From)
						require.Equal(t, doc.To, docRead.To)
						require.Equal(t, 12, docRead.Distance)

					})

					t.Run("Delete Edge", func(t *testing.T) {
						resp, err := edgeDefResp.Edge.DeleteEdge(ctx, docKey, nil)
						require.NoError(t, err)
						require.Empty(t, resp.Old)
					})
				})
			})
		})
	})
}

func Test_EdgeExtended(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, nil, nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					edgeColName := "citiesPerState"
					toColName := "states"
					fromColName := "cities"

					toVertex := ensureVertex(t, ctx, graph, toColName, Place{Name: "Texas"})
					fromVertex := ensureVertex(t, ctx, graph, fromColName, Place{Name: "Houston"})

					edgeDefResp, err := graph.CreateEdgeDefinition(ctx, edgeColName, []string{fromColName}, []string{toColName}, nil)
					require.NoError(t, err)
					require.Len(t, edgeDefResp.GraphDefinition.EdgeDefinitions, 1)

					doc := RouteEdge{
						From:     string(fromVertex.ID),
						To:       string(toVertex.ID),
						Distance: 7,
					}
					var docKey string

					t.Run("Create Edge", func(t *testing.T) {
						var newObject RouteEdge
						opts := arangodb.CreateEdgeOptions{
							WaitForSync: newBool(true),
							NewObject:   &newObject,
						}

						createEdgeResp, err := edgeDefResp.Edge.CreateEdge(ctx, doc, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, createEdgeResp.Key)
						require.NotEmpty(t, newObject)
						require.Equal(t, doc, newObject)
						docKey = createEdgeResp.Key
					})

					t.Run("Update Edge", func(t *testing.T) {
						var oldObject RouteEdge
						opts := arangodb.EdgeUpdateOptions{
							WaitForSync: newBool(true),
							OldObject:   &oldObject,
						}
						updateEdgeResp, err := edgeDefResp.Edge.UpdateEdge(ctx, docKey, map[string]int{"distance": 10}, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, updateEdgeResp.Key)
						require.NotEmpty(t, oldObject)
						require.Equal(t, doc, oldObject)

						docRead := RouteEdge{}
						optsRead := arangodb.GetEdgeOptions{
							Rev: updateEdgeResp.Rev,
						}
						err = edgeDefResp.Edge.GetEdge(ctx, docKey, &docRead, &optsRead)
						require.NoError(t, err)
						require.Equal(t, doc.From, docRead.From)
						require.Equal(t, doc.To, docRead.To)
						require.Equal(t, 10, docRead.Distance)

					})

					t.Run("Replace Edge", func(t *testing.T) {
						var oldObject RouteEdge
						var newObject RouteEdge
						opts := arangodb.EdgeReplaceOptions{
							OldObject:   &oldObject,
							NewObject:   &newObject,
							WaitForSync: newBool(true),
						}

						docReplace := RouteEdge{
							From:     string(fromVertex.ID),
							To:       string(toVertex.ID),
							Distance: 12,
						}

						replaceEdgeResp, err := edgeDefResp.Edge.ReplaceEdge(ctx, docKey, docReplace, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, replaceEdgeResp.Key)
						require.NotEmpty(t, oldObject)
						require.NotEmpty(t, newObject)
						require.Equal(t, 10, oldObject.Distance)
						require.Equal(t, 12, newObject.Distance)
					})

					t.Run("Delete Edge", func(t *testing.T) {
						var oldObject RouteEdge
						opts := arangodb.DeleteEdgeOptions{
							OldObject: &oldObject,
						}
						resp, err := edgeDefResp.Edge.DeleteEdge(ctx, docKey, &opts)
						require.NoError(t, err)
						require.NotEmpty(t, resp.Old)
						require.Equal(t, 12, oldObject.Distance)
					})
				})
			})
		})
	})
}
