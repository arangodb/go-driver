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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/util/connection/wrappers/async"
)

func TestAsyncJobListDone(t *testing.T) {
	client := createAsyncClientFromEnv(t)
	ctx := context.Background()
	ctxAsync := driver.WithAsync(context.Background())

	// Trigger two async requests
	info, err := client.Version(ctxAsync)
	require.Error(t, err)
	require.Empty(t, info.Version)

	id, isAsyncId := async.IsAsyncJobInProgress(err)
	require.True(t, isAsyncId)
	require.NotEmpty(t, id)

	info2, err2 := client.Version(ctxAsync)
	require.Error(t, err2)
	require.Empty(t, info2.Version)

	id2, isAsyncId2 := async.IsAsyncJobInProgress(err2)
	require.True(t, isAsyncId2)
	require.NotEmpty(t, id2)

	// wait for the jobs to be done
	time.Sleep(3 * time.Second)

	t.Run("AsyncJobs List Done jobs", func(t *testing.T) {
		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 2)
	})

	t.Run("AsyncJobs List with Count param", func(t *testing.T) {
		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, &driver.AsyncJobListOptions{Count: 1})
		require.NoError(t, err)
		require.Len(t, jobs, 1)
	})

	t.Run("async request final result", func(t *testing.T) {
		info, err = client.Version(driver.WithAsyncId(ctx, id))
		require.NoError(t, err)
		require.NotEmpty(t, info.Version)
	})

	t.Run("List of Done jobs should decrease", func(t *testing.T) {
		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		// finish the second job
		info2, err2 = client.Version(driver.WithAsyncId(ctx, id2))
		require.NoError(t, err2)
		require.NotEmpty(t, info2.Version)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func TestAsyncJobListPending(t *testing.T) {
	client := createAsyncClientFromEnv(t)
	ctx := context.Background()
	ctxAsync := driver.WithAsync(context.Background())

	db := ensureDatabase(ctx, client, databaseName("db", "async"), nil, t)
	defer db.Remove(ctx)
	col := ensureCollection(ctx, db, "frontend", nil, t)

	idTransaction := runLongRequest(t, ctxAsync, db, 2, col.Name())
	require.NotEmpty(t, idTransaction)

	t.Run("AsyncJobs List Pending jobs", func(t *testing.T) {
		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)
	})

	t.Run("wait fot the async jobs to be done", func(t *testing.T) {
		time.Sleep(4 * time.Second)

		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

	})

	t.Run("read async result", func(t *testing.T) {
		idTransaction := runLongRequest(t, driver.WithAsyncId(ctx, idTransaction), db, 2, col.Name())
		require.Empty(t, idTransaction)

		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func TestAsyncJobCancel(t *testing.T) {
	client := createAsyncClientFromEnv(t)
	ctx := context.Background()
	ctxAsync := driver.WithAsync(context.Background())

	db := ensureDatabase(ctx, client, databaseName("db", "async", "cancel"), nil, t)
	defer db.Remove(ctx)

	aqlQuery := "FOR i IN 1..10 FOR j IN 1..10 LET x = sleep(1.0) FILTER i == 5 && j == 5 RETURN 42"
	_, err := db.Query(ctxAsync, aqlQuery, nil)
	require.Error(t, err)
	require.IsType(t, async.ErrorAsyncJobInProgress{}, err)

	id, isAsyncId := async.IsAsyncJobInProgress(err)
	require.True(t, isAsyncId)

	t.Run("cancel Pending job", func(t *testing.T) {
		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := client.AsyncJob().Cancel(ctx, jobs[0])
		require.NoError(t, err)
		require.True(t, success)

	})

	t.Run("cancelled job should move from pending to done state", func(t *testing.T) {
		time.Sleep(5 * time.Second)

		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Equal(t, id, jobs[0])

		_, err = db.Query(driver.WithAsyncId(ctx, id), aqlQuery, nil)
		require.Error(t, err)
		require.Equal(t, "canceled request", err.Error())
	})
}

func TestAsyncJobDelete(t *testing.T) {
	client := createAsyncClientFromEnv(t)
	ctx := context.Background()
	ctxAsync := driver.WithAsync(context.Background())

	db := ensureDatabase(ctx, client, databaseName("db", "async", "cancel"), nil, t)
	defer db.Remove(ctx)
	col := ensureCollection(ctx, db, "backend", nil, t)

	t.Run("delete all jobs", func(t *testing.T) {
		// Trigger async request
		_, err := client.Version(ctxAsync)
		require.Error(t, err)

		_, err2 := client.Version(ctxAsync)
		require.Error(t, err2)

		time.Sleep(2 * time.Second)

		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 2)

		success, err := client.AsyncJob().Delete(ctx, driver.DeleteAllJobs, nil)
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete specific job which is done", func(t *testing.T) {
		// Trigger async request
		_, err := client.Version(ctxAsync)
		require.Error(t, err)

		time.Sleep(2 * time.Second)

		jobs, err := client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := client.AsyncJob().Delete(ctx, driver.DeleteSingleJob, &driver.AsyncJobDeleteOptions{JobID: jobs[0]})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete pending job", func(t *testing.T) {
		idTransaction := runLongRequest(t, ctxAsync, db, 10, col.Name())
		require.NotEmpty(t, idTransaction)

		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := client.AsyncJob().Delete(ctx, driver.DeleteSingleJob, &driver.AsyncJobDeleteOptions{JobID: jobs[0]})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete expired jobs", func(t *testing.T) {
		idTransaction := runLongRequest(t, ctxAsync, db, 10, col.Name())
		require.NotEmpty(t, idTransaction)

		jobs, err := client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := client.AsyncJob().Delete(ctx, driver.DeleteExpiredJobs,
			&driver.AsyncJobDeleteOptions{Stamp: time.Now().Add(24 * time.Hour)})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = client.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = client.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func runLongRequest(t *testing.T, ctx context.Context, db driver.Database, lenOfTimeInSec int, colName string) string {
	_, err := db.Transaction(ctx, fmt.Sprintf("function () {require('internal').sleep(%d);}", lenOfTimeInSec),
		&driver.TransactionOptions{ReadCollections: []string{colName}})
	if err != nil {
		require.IsType(t, async.ErrorAsyncJobInProgress{}, err)

		idTransaction, isAsyncId := async.IsAsyncJobInProgress(err)
		require.True(t, isAsyncId)

		return idTransaction
	}
	return ""
}
