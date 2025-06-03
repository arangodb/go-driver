//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

func TestCreatePregelJob(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.10", t)
	skipFromVersion(c, "3.12", t)
	skipNoCluster(c, t)

	db := ensureDatabase(ctx, c, "pregel_job_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "pregel_graph_test", nil, t)

	nameVertex := "test_pregel_vertex"
	ensureCollection(nil, db, nameVertex, &driver.CreateCollectionOptions{
		NumberOfShards:    4,
		ReplicationFactor: 1,
	}, t)

	colVertex := ensureVertexCollection(ctx, g, nameVertex, t)

	doc := UserDoc{
		Name: "Jan",
		Age:  12,
	}
	meta, err := colVertex.CreateDocument(ctx, doc)
	require.NoError(t, err)

	nameEdge := "test_pregel_edge"
	ensureCollection(ctx, db, nameEdge, &driver.CreateCollectionOptions{
		Type:                 driver.CollectionTypeEdge,
		NumberOfShards:       4,
		ReplicationFactor:    1,
		ShardKeys:            []string{"vertex"},
		DistributeShardsLike: nameVertex,
	}, t)
	ensureEdgeCollection(ctx, g, nameEdge, []string{nameVertex}, []string{nameVertex}, t)

	jobId, err := db.StartJob(ctx, driver.PregelJobOptions{
		Algorithm: driver.PregelAlgorithmPageRank,
		GraphName: g.Name(),
		Params: map[string]interface{}{
			"store":       true,
			"resultField": "resultField",
		},
	})
	require.Nilf(t, err, "Failed to start Pregel job: %s", describe(err))
	require.NotEmpty(t, jobId, "JobId is empty")

	waitForDataPropagation()

	type UserDocPregelResult struct {
		UserDoc
		ResultField float64 `json:"resultField"`
	}

	docResult := UserDocPregelResult{}
	_, err = colVertex.ReadDocument(ctx, meta.Key, &docResult)
	require.NoError(t, err)
	require.Equal(t, doc.Name, docResult.Name)
	require.Equal(t, doc.Age, docResult.Age)
	require.NotEmpty(t, docResult.ResultField)
	require.Greater(t, docResult.ResultField, 0.0)

	t.Logf("resultField value: %f", docResult.ResultField)
}
