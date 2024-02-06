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

func Test_GraphVertexCollections(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithGraph(t, db, sampleGraph(db), nil, func(graph arangodb.Graph) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					cols, err := graph.VertexCollections(ctx)
					require.NoError(t, err)
					// We have 2 default orphaned vertex collections
					require.Len(t, cols, 2)

					colName := "test_vertex_collection"

					createResp, err := graph.CreateVertexCollection(ctx, colName, nil)
					require.NoError(t, err)
					require.Contains(t, createResp.GraphDefinition.OrphanCollections, colName)

					exist, err := graph.VertexCollectionExists(ctx, colName)
					require.NoError(t, err)
					require.True(t, exist, "vertex collection should exist")

					colsAfterCreate, err := graph.VertexCollections(ctx)
					require.NoError(t, err)
					require.Len(t, colsAfterCreate, 3)

					colRead, err := graph.VertexCollection(ctx, colName)
					require.NoError(t, err)
					require.Equal(t, colName, colRead.Name())

					delResp, err := graph.DeleteVertexCollection(ctx, colName, nil)
					require.NoError(t, err)
					require.NotContains(t, delResp.GraphDefinition.OrphanCollections, colName)
				})
			})
		})
	})
}
