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
	asyncClient := createAsyncClientFromEnv(t)
	syncClient := createClientFromEnv(t, false)
	ctx := context.Background()

	// Trigger async request
	info, err := asyncClient.Version(ctx)
	require.Error(t, err)
	require.Empty(t, info.Version)

	id, isAsyncId := async.IsAsyncJobInProgress(err)
	require.True(t, isAsyncId)
	require.NotEmpty(t, id)

	info2, err2 := asyncClient.Version(ctx)
	require.Error(t, err2)
	require.Empty(t, info2.Version)

	id2, isAsyncId2 := async.IsAsyncJobInProgress(err2)
	require.True(t, isAsyncId2)
	require.NotEmpty(t, id2)

	// wait fot the jobs to be done
	time.Sleep(3 * time.Second)

	t.Run("AsyncJobs List Done jobs", func(t *testing.T) {
		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 2)
	})

	t.Run("AsyncJobs List with Count param", func(t *testing.T) {
		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, &driver.AsyncJobListOptions{Count: 1})
		require.NoError(t, err)
		require.Len(t, jobs, 1)
	})

	t.Run("async request final result", func(t *testing.T) {
		info, err = asyncClient.Version(driver.WithAsyncId(ctx, id))
		require.NoError(t, err)
		require.NotEmpty(t, info.Version)
	})

	t.Run("List of Done jobs should decrease", func(t *testing.T) {
		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		// finish the second job
		info2, err2 = asyncClient.Version(driver.WithAsyncId(ctx, id2))
		require.NoError(t, err2)
		require.NotEmpty(t, info2.Version)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func TestAsyncJobListPending(t *testing.T) {
	asyncClient := createAsyncClientFromEnv(t)
	syncClient := createClientFromEnv(t, false)
	ctx := context.Background()

	db := ensureDatabase(ctx, syncClient, databaseName("db", "async"), nil, t)
	defer db.Remove(ctx)
	col := ensureCollection(ctx, db, "frontend", nil, t)

	dbAsync := getDatabaseWithAsyncClient(t, ctx, asyncClient, db.Name())

	idTransaction := runAsyncRequest(t, ctx, dbAsync, 2, col.Name())
	require.NotEmpty(t, idTransaction)

	t.Run("AsyncJobs List Pending jobs", func(t *testing.T) {
		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)
	})

	t.Run("wait fot the async jobs to be done", func(t *testing.T) {
		time.Sleep(4 * time.Second)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

	})

	t.Run("read async result", func(t *testing.T) {
		idTransaction := runAsyncRequest(t, driver.WithAsyncId(ctx, idTransaction), dbAsync, 2, col.Name())
		require.Empty(t, idTransaction)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func TestAsyncJobCancel(t *testing.T) {
	asyncClient := createAsyncClientFromEnv(t)
	syncClient := createClientFromEnv(t, false)
	ctx := context.Background()

	db := ensureDatabase(ctx, syncClient, databaseName("db", "async", "cancel"), nil, t)
	defer db.Remove(ctx)

	dbAsync := getDatabaseWithAsyncClient(t, ctx, asyncClient, db.Name())

	aqlQuery := "FOR i IN 1..10 FOR j IN 1..10 LET x = sleep(1.0) FILTER i == 5 && j == 5 RETURN 42"
	_, err := dbAsync.Query(ctx, aqlQuery, nil)
	require.Error(t, err)
	require.IsType(t, async.ErrorAsyncJobInProgress{}, err)

	id, isAsyncId := async.IsAsyncJobInProgress(err)
	require.True(t, isAsyncId)

	t.Run("cancel Pending job", func(t *testing.T) {
		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := syncClient.AsyncJob().Cancel(ctx, jobs[0])
		require.NoError(t, err)
		require.True(t, success)

	})

	t.Run("cancelled job should move from pending to done state", func(t *testing.T) {
		time.Sleep(5 * time.Second)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Equal(t, id, jobs[0])

		_, err = dbAsync.Query(driver.WithAsyncId(ctx, id), aqlQuery, nil)
		require.Error(t, err)
		require.Equal(t, "canceled request", err.Error())
	})
}

func TestAsyncJobDelete(t *testing.T) {
	asyncClient := createAsyncClientFromEnv(t)
	syncClient := createClientFromEnv(t, false)
	ctx := context.Background()

	db := ensureDatabase(ctx, syncClient, databaseName("db", "async", "cancel"), nil, t)
	defer db.Remove(ctx)
	col := ensureCollection(ctx, db, "backend", nil, t)

	dbAsync := getDatabaseWithAsyncClient(t, ctx, asyncClient, db.Name())

	t.Run("delete all jobs", func(t *testing.T) {
		// Trigger async request
		_, err := asyncClient.Version(ctx)
		require.Error(t, err)

		_, err2 := asyncClient.Version(ctx)
		require.Error(t, err2)

		time.Sleep(2 * time.Second)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 2)

		success, err := syncClient.AsyncJob().Delete(ctx, driver.DeleteAllJobs, nil)
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete specific job which is done", func(t *testing.T) {
		// Trigger async request
		_, err := asyncClient.Version(ctx)
		require.Error(t, err)

		time.Sleep(2 * time.Second)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := syncClient.AsyncJob().Delete(ctx, driver.DeleteSingleJob, &driver.AsyncJobDeleteOptions{JobID: jobs[0]})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete pending job", func(t *testing.T) {
		idTransaction := runAsyncRequest(t, ctx, dbAsync, 10, col.Name())
		require.NotEmpty(t, idTransaction)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := syncClient.AsyncJob().Delete(ctx, driver.DeleteSingleJob, &driver.AsyncJobDeleteOptions{JobID: jobs[0]})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})

	t.Run("delete expired jobs", func(t *testing.T) {
		idTransaction := runAsyncRequest(t, ctx, dbAsync, 10, col.Name())
		require.NotEmpty(t, idTransaction)

		jobs, err := syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		success, err := syncClient.AsyncJob().Delete(ctx, driver.DeleteExpiredJobs,
			&driver.AsyncJobDeleteOptions{Stamp: time.Now().Add(24 * time.Hour)})
		require.NoError(t, err)
		require.True(t, success)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobPending, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)

		jobs, err = syncClient.AsyncJob().List(ctx, driver.JobDone, nil)
		require.NoError(t, err)
		require.Len(t, jobs, 0)
	})
}

func getDatabaseWithAsyncClient(t *testing.T, ctx context.Context, asyncClient driver.Client, dbName string) driver.Database {
	dbAsync, err := asyncClient.Database(ctx, dbName)
	require.Error(t, err)

	idDB, isAsyncId := async.IsAsyncJobInProgress(err)
	require.IsType(t, async.ErrorAsyncJobInProgress{}, err)
	require.True(t, isAsyncId)

	dbAsync, err = asyncClient.Database(driver.WithAsyncId(ctx, idDB), dbName)
	require.NoError(t, err)

	return dbAsync
}

func runAsyncRequest(t *testing.T, ctx context.Context, dbAsync driver.Database, lenOfTimeInSec int, colName string) string {
	_, err := dbAsync.Transaction(ctx, fmt.Sprintf("function () {require('internal').sleep(%d);}", lenOfTimeInSec),
		&driver.TransactionOptions{ReadCollections: []string{colName}})
	if err != nil {
		require.IsType(t, async.ErrorAsyncJobInProgress{}, err)

		idTransaction, isAsyncId := async.IsAsyncJobInProgress(err)
		require.True(t, isAsyncId)

		return idTransaction
	}
	return ""
}
