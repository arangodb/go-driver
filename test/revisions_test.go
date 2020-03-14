package test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO more unit tests ?
func TestRevisionTree(t *testing.T) {
	c := createClientFromEnv(t, true)
	if getTestMode() != testModeSingle {
		t.Skipf("Not a single")
	}
	//skipBelowVersion(c, "3.7", t) // TODO turn on at the end of task

	db := ensureDatabase(nil, c, "revision_tree", nil, t)
	col := ensureCollection(nil, db, "revision_tree", nil, t)

	var noOfDocuments int = 100
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
	require.NoError(t, err)
	require.NotEmpty(t, tree.Version)
	require.NotEmpty(t, tree.RangeMin)
	require.NotEmpty(t, tree.RangeMax)
	require.NotEmpty(t, tree.Nodes)

	getRevisions := func() ([]driver.Revisions, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		rangeRevisions := []driver.RevisionMinMax{{tree.RangeMin, tree.RangeMax}}
		return c.Replication().GetRevisionsByRanges(timeoutCtx, db, batch.BatchID(), col.Name(), rangeRevisions, nil)
	}

	revisions, err := getRevisions()
	require.NoError(t, err)
	require.NotEmpty(t, revisions)
	require.Len(t, revisions, 1)
	require.Len(t, revisions[0], noOfDocuments)

	getDocuments := func() ([]map[string]interface{}, error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		return c.Replication().GetRevisionDocuments(timeoutCtx, db, batch.BatchID(), col.Name(), revisions[0])
	}

	documents, err := getDocuments()
	require.NoError(t, err)
	require.Len(t, documents, noOfDocuments)

	for i, d := range documents {
		user := UserDoc{}
		bytes, _ := json.Marshal(d)
		json.Unmarshal(bytes, &user)
		require.Equal(t, user, expectedDocuments[i])
	}
}
