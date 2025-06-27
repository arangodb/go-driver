//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

func TestRevisionTree(t *testing.T) {
	if getTestMode() != testModeSingle {
		t.Skipf("Not a single")
	}
	c := createClient(t, nil)
	skipBelowVersion(c, "3.8", t)

	db := ensureDatabase(nil, c, "revision_tree", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "revision_tree", nil, t)

	var noOfDocuments int = 80000
	expectedDocuments := make([]interface{}, 0, noOfDocuments)
	for i := 0; i < noOfDocuments; i++ {
		expectedDocuments = append(expectedDocuments, UserDoc{
			Name: fmt.Sprintf("User%d", 1),
			Age:  i,
		})
	}

	_, _, err := col.CreateDocuments(context.Background(), expectedDocuments)
	require.NoErrorf(t, err, "Failed to create new documents: %s", describe(err))

	batch, err := c.Replication().CreateBatch(context.Background(), db, 123, time.Hour)
	require.NoError(t, err)
	defer batch.Delete(context.Background())

	getTree := func() (driver.RevisionTree, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		return c.Replication().GetRevisionTree(timeoutCtx, db, batch.BatchID(), col.Name())
	}

	tree, err := getTree()
	if err != nil {
		if driver.IsArangoErrorWithCode(err, http.StatusNotImplemented) {
			t.Skip("Collection '" + col.Name() + "' does not support revision-based replication")
		}

		require.NoError(t, err)
	}

	require.NotEmpty(t, tree.Version)
	require.NotEmpty(t, tree.RangeMin)
	require.NotEmpty(t, tree.RangeMax)
	require.NotEmpty(t, tree.InitialRangeMin)
	require.NotEmpty(t, tree.Nodes)

	branchFactor := 8
	noOfLeavesOnLevel := 1
	noOfLeaves := noOfLeavesOnLevel
	for i := 1; i <= tree.MaxDepth; i++ {
		noOfLeavesOnLevel *= branchFactor
		if i == tree.MaxDepth {
			noOfLeaves = noOfLeavesOnLevel
		}
	}
	require.Equalf(t, noOfDocuments, int(tree.Count), "Count value of tree is not correct")
	require.Equalf(t, noOfLeaves, len(tree.Nodes), "Number of leaves in the revision tree is not correct")

	getRanges := func() driver.Revisions {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		rangeRevisions := []driver.RevisionMinMax{{tree.RangeMin, tree.RangeMax}}
		var resume driver.RevisionUInt64
		revisions := make(driver.Revisions, 0)

		for {
			ranges, err := c.Replication().GetRevisionsByRanges(timeoutCtx, db, batch.BatchID(), col.Name(),
				rangeRevisions, resume)
			require.NoError(t, err)

			if len(ranges.Ranges[0]) == 0 {
				// let's try again because we should get ranges at the end. There is a one minute timeout for it
				continue
			}

			revisions = append(revisions, ranges.Ranges[0]...)

			if ranges.Resume == 0 {
				break
			}
			resume = ranges.Resume
		}
		return revisions
	}

	revisions := getRanges()
	require.NotEmpty(t, revisions)
	require.Lenf(t, revisions, noOfDocuments, "Number of revisions ranges is not correct")

	getDocuments := func() ([]map[string]interface{}, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		return c.Replication().GetRevisionDocuments(timeoutCtx, db, batch.BatchID(), col.Name(), revisions)
	}

	documents, err := getDocuments()
	require.NoError(t, err)
	require.Lenf(t, documents, noOfDocuments, "Number of documents is not equal")

	for i, d := range documents {
		user := UserDoc{}
		bytes, _ := json.Marshal(d)
		json.Unmarshal(bytes, &user)
		require.Equalf(t, user, expectedDocuments[i], "Documents should be the same")
	}
}
