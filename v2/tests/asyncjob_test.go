//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

var asyncTestOpt = WrapOptions{
	Async:    utils.NewType(true),
	Parallel: utils.NewType(false),
}

func TestAsyncJobListDone(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipBelowVersion(client, ctx, "3.11.1", t)
			skipResilientSingleMode(t)

			ctxAsync := connection.WithAsync(context.Background())

			// Trigger two async requests
			info, err := client.Version(ctxAsync)
			require.Empty(t, info.Version)
			require.Error(t, err)

			id, isAsyncId := connection.IsAsyncJobInProgress(err)
			require.True(t, isAsyncId)
			require.NotEmpty(t, id)

			info2, err2 := client.Version(ctxAsync)
			require.Error(t, err2)
			require.Empty(t, info2.Version)

			id2, isAsyncId2 := connection.IsAsyncJobInProgress(err2)
			require.True(t, isAsyncId2)
			require.NotEmpty(t, id2)

			// wait for the jobs to be done
			time.Sleep(3 * time.Second)

			t.Run("AsyncJobs List Done jobs", func(t *testing.T) {
				jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, nil)
				require.NoError(t, err)
				require.Len(t, jobs, 2)
			})

			t.Run("AsyncJobs List with Count param", func(t *testing.T) {
				jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, &arangodb.AsyncJobListOptions{Count: 1})
				require.NoError(t, err)
				require.Len(t, jobs, 1)
			})

			t.Run("async request final result", func(t *testing.T) {
				info, err = client.Version(connection.WithAsyncID(ctx, id))
				require.NoError(t, err)
				require.NotEmpty(t, info.Version)
			})

			t.Run("List of Done jobs should decrease", func(t *testing.T) {
				jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, nil)
				require.NoError(t, err)
				require.Len(t, jobs, 1)

				// finish the second job
				info2, err2 = client.Version(connection.WithAsyncID(ctx, id2))
				require.NoError(t, err2)
				require.NotEmpty(t, info2.Version)

				jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
				require.NoError(t, err)
				require.Len(t, jobs, 0)
			})
		})
	}, asyncTestOpt)
}

func TestAsyncJobListPending(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.11.1", t)
					skipResilientSingleMode(t)

					ctxAsync := connection.WithAsync(context.Background())

					idTransaction := runLongRequest(t, ctxAsync, db, 2, col.Name())
					require.NotEmpty(t, idTransaction)

					t.Run("AsyncJobs List Pending jobs", func(t *testing.T) {
						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)
					})

					t.Run("wait fot the async jobs to be done", func(t *testing.T) {
						time.Sleep(4 * time.Second)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)

					})

					t.Run("read async result", func(t *testing.T) {
						idTransaction := runLongRequest(t, connection.WithAsyncID(ctx, idTransaction), db, 2, col.Name())
						require.Empty(t, idTransaction)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)
					})

				})
			})
		})
	}, asyncTestOpt)
}

func TestAsyncJobCancel(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.11.1", t)
					skipResilientSingleMode(t)

					ctxAsync := connection.WithAsync(context.Background())

					aqlQuery := "FOR i IN 1..10 FOR j IN 1..10 LET x = sleep(1.0) FILTER i == 5 && j == 5 RETURN 42"
					_, err := db.Query(ctxAsync, aqlQuery, nil)
					require.Error(t, err)

					id, isAsyncId := connection.IsAsyncJobInProgress(err)
					require.True(t, isAsyncId)

					t.Run("cancel Pending job", func(t *testing.T) {
						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)

						success, err := client.AsyncJobCancel(ctx, jobs[0])
						require.NoError(t, err)
						require.True(t, success)

					})

					t.Run("cancelled job should move from pending to done state", func(t *testing.T) {
						time.Sleep(5 * time.Second)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)
						require.Equal(t, id, jobs[0])

						_, err = db.Query(connection.WithAsyncID(ctx, id), aqlQuery, nil)
						require.Error(t, err)
						require.Contains(t, err.Error(), "canceled")
					})
				})
			})
		})
	}, asyncTestOpt)
}

func TestAsyncJobDelete(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipBelowVersion(client, ctx, "3.11.1", t)
					skipResilientSingleMode(t)

					ctxAsync := connection.WithAsync(context.Background())

					t.Run("delete all jobs", func(t *testing.T) {
						// Trigger async request
						_, err := client.Version(ctxAsync)
						require.Error(t, err)

						_, err2 := client.Version(ctxAsync)
						require.Error(t, err2)

						time.Sleep(2 * time.Second)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 2)

						success, err := client.AsyncJobDelete(ctx, arangodb.DeleteAllJobs, nil)
						require.NoError(t, err)
						require.True(t, success)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)
					})

					t.Run("delete specific job which is done", func(t *testing.T) {
						// Trigger async request
						_, err := client.Version(ctxAsync)
						require.Error(t, err)

						time.Sleep(2 * time.Second)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)

						success, err := client.AsyncJobDelete(ctx, arangodb.DeleteSingleJob, &arangodb.AsyncJobDeleteOptions{JobID: jobs[0]})
						require.NoError(t, err)
						require.True(t, success)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)
					})

					t.Run("delete pending job", func(t *testing.T) {
						idTransaction := runLongRequest(t, ctxAsync, db, 10, col.Name())
						require.NotEmpty(t, idTransaction)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)

						success, err := client.AsyncJobDelete(ctx, arangodb.DeleteSingleJob, &arangodb.AsyncJobDeleteOptions{JobID: jobs[0]})
						require.NoError(t, err)
						require.True(t, success)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)
					})

					t.Run("delete expired jobs", func(t *testing.T) {
						idTransaction := runLongRequest(t, ctxAsync, db, 10, col.Name())
						require.NotEmpty(t, idTransaction)

						jobs, err := client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 1)

						success, err := client.AsyncJobDelete(ctx, arangodb.DeleteExpiredJobs,
							&arangodb.AsyncJobDeleteOptions{Stamp: time.Now().Add(24 * time.Hour)})
						require.NoError(t, err)
						require.True(t, success)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobPending, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)

						jobs, err = client.AsyncJobList(ctx, arangodb.JobDone, nil)
						require.NoError(t, err)
						require.Len(t, jobs, 0)
					})

				})
			})
		})
	}, asyncTestOpt)
}

func runLongRequest(t *testing.T, ctx context.Context, db arangodb.Database, lenOfTimeInSec int, colName string) string {
	txOpt := arangodb.TransactionJSOptions{
		Action: fmt.Sprintf("function () {require('internal').sleep(%d);}", lenOfTimeInSec),
		Collections: arangodb.TransactionCollections{
			Read: []string{colName},
		},
	}

	_, err := db.TransactionJS(ctx, txOpt)
	if err != nil {
		idTransaction, isAsyncId := connection.IsAsyncJobInProgress(err)
		require.True(t, isAsyncId)

		return idTransaction
	}
	return ""
}
