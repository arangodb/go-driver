//
// DISCLAIMER
//
// Copyright 2022 ArangoDB GmbH, Cologne, Germany
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

func TestCreatePregelJob(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.10", t)
	skipNoCluster(c, t)

	db := ensureDatabase(ctx, c, "pregel_job_test3", nil, t)
	g := ensureGraph(ctx, db, "pregel_graph_test3", nil, t)

	nameVertex := "test_pregel_vertex"
	ensureCollection(nil, db, nameVertex, &driver.CreateCollectionOptions{
		NumberOfShards: 4,
	}, t)
	ensureVertexCollection(ctx, g, nameVertex, t)

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
	})
	require.Nilf(t, err, "Failed to start Pregel job: %s", describe(err))
	require.NotEmpty(t, jobId, "JobId is empty")

	job, err := db.GetJob(ctx, jobId)
	require.Nilf(t, err, "Failed to get job: %s", describe(err))
	require.Equal(t, jobId, job.ID, "JobId mismatch")
	require.NotEmpty(t, job.Detail, "Detail is empty")

	jobs, err := db.GetJobs(ctx)
	require.Nilf(t, err, "Failed to get running jobs: %s", describe(err))
	require.Len(t, jobs, 1, "Expected 1 job, got %d", len(jobs))

	err = db.CancelJob(ctx, jobId)
	require.Nilf(t, err, "Failed to cancel job: %s", describe(err))
}
