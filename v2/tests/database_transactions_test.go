//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_DatabaseCreateReplicationV2(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		databaseReplication2Required(t, client, context.Background())

		opts := arangodb.CreateDatabaseOptions{
			Users: nil,
			Options: arangodb.CreateDatabaseDefaultOptions{
				ReplicationVersion: arangodb.DatabaseReplicationVersionTwo,
			},
		}
		WithDatabase(t, client, &opts, func(db arangodb.Database) {
			t.Run("Transaction", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
					info, err := db.Info(ctx)
					require.NoErrorf(t, err, "failed to get database info")
					require.Equal(t, arangodb.DatabaseReplicationVersionTwo, info.ReplicationVersion)
				})
			})
		})
	})
}

func Test_DatabaseTransactions_DataIsolation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			t.Run("Transaction", func(t *testing.T) {
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						d := document{
							basicDocument: basicDocument{Key: "uniq_key"},
							Fields:        "DOC",
						}

						var tid arangodb.TransactionID

						// Start transaction
						require.NoError(t, db.WithTransaction(ctx, arangodb.TransactionCollections{
							Write: []string{
								col.Name(),
							},
						}, &arangodb.BeginTransactionOptions{
							WaitForSync: true,
						}, nil, nil, func(ctx context.Context, transaction arangodb.Transaction) error {
							tid = transaction.ID()

							// Get collection in transaction
							tCol, err := transaction.GetCollection(ctx, col.Name(), nil)
							require.NoError(t, err)

							_, err = tCol.CreateDocument(ctx, d)
							require.NoError(t, err)

							// Check if non transaction handler can read document
							DocumentNotExists(t, col, d)

							// Check if transaction handler can read document
							DocumentExists(t, tCol, d)
							// Do commit
							return nil
						}))

						DocumentExists(t, col, d)

						ensureTransactionStatus(t, db, tid, arangodb.TransactionCommitted)
					})
				})
			})

			t.Run("Transaction - With Error", func(t *testing.T) {
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						d := document{
							basicDocument: basicDocument{Key: "uniq_key"},
							Fields:        "DOC",
						}

						var tid arangodb.TransactionID

						// Start transaction
						require.EqualError(t, db.WithTransaction(ctx, arangodb.TransactionCollections{
							Write: []string{
								col.Name(),
							},
						}, &arangodb.BeginTransactionOptions{
							WaitForSync: true,
						}, nil, nil, func(ctx context.Context, transaction arangodb.Transaction) error {
							tid = transaction.ID()

							// Get collection in transaction
							tCol, err := transaction.GetCollection(ctx, col.Name(), nil)
							require.NoError(t, err)

							_, err = tCol.CreateDocument(ctx, d)
							require.NoError(t, err)

							// Check if non transaction handler can read document
							DocumentNotExists(t, col, d)

							// Check if transaction handler can read document
							DocumentExists(t, tCol, d)

							// Do abort
							return errors.Errorf("CustomAbortError")
						}), "CustomAbortError")

						DocumentNotExists(t, col, d)

						ensureTransactionStatus(t, db, tid, arangodb.TransactionAborted)
					})
				})
			})

			t.Run("Transaction - With Panic", func(t *testing.T) {
				t.Skipf("")
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						d := document{
							basicDocument: basicDocument{Key: "uniq_key"},
							Fields:        "DOC",
						}

						var tid arangodb.TransactionID

						// Start transaction
						ExpectPanic(t, func() {
							require.NoError(t, db.WithTransaction(ctx, arangodb.TransactionCollections{
								Write: []string{
									col.Name(),
								},
							}, &arangodb.BeginTransactionOptions{
								WaitForSync: true,
							}, nil, nil, func(ctx context.Context, transaction arangodb.Transaction) error {
								tid = transaction.ID()

								// Get collection in transaction
								tCol, err := transaction.GetCollection(ctx, col.Name(), nil)
								require.NoError(t, err)

								_, err = tCol.CreateDocument(ctx, d)
								require.NoError(t, err)

								// Check if non transaction handler can read document
								DocumentNotExists(t, col, d)

								// Check if transaction handler can read document
								DocumentExists(t, tCol, d)

								// Do abort
								panic("CustomPanicError")
							}))
						}, "CustomPanicError")

						DocumentNotExists(t, col, d)

						ensureTransactionStatus(t, db, tid, arangodb.TransactionAborted)
					})
				})
			})
		})
	})
}

func Test_DatabaseTransactions_DocumentLock(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					d := document{
						basicDocument: basicDocument{Key: GenerateUUID("test-doc-basic")},
						Fields:        "no1",
					}

					ud := document{
						basicDocument: d.basicDocument,
						Fields:        "newNo",
					}
					_, err := col.CreateDocument(ctx, d)
					require.NoError(t, err)

					t1, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{Write: []string{col.Name()}}, &arangodb.BeginTransactionOptions{
						LockTimeoutDuration: 5 * time.Second,
					})
					require.NoError(t, err)
					defer abortTransaction(t, t1)

					col1, err := t1.GetCollection(ctx, col.Name(), nil)
					require.NoError(t, err)

					t2, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{Write: []string{col.Name()}}, &arangodb.BeginTransactionOptions{
						LockTimeoutDuration: 1 * time.Second,
					})
					require.NoError(t, err)
					defer abortTransaction(t, t1)

					col2, err := t2.GetCollection(ctx, col.Name(), nil)
					require.NoError(t, err)

					_, err = col1.UpdateDocument(ctx, d.Key, ud)
					require.NoError(t, err)

					sctx, c := context.WithTimeout(ctx, 2*time.Second)
					defer c()
					_, err = col2.UpdateDocument(sctx, d.Key, ud)
					require.Error(t, err)
					require.True(t, shared.IsOperationTimeout(err))
				})
			})
		})
	})
}

func Test_DatabaseTransactions_List(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			WithDatabase(t, client, nil, func(db arangodb.Database) {
				t.Run("List all transactions", func(t *testing.T) {
					t1, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{}, nil)
					require.NoError(t, err)
					t2, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{}, nil)
					require.NoError(t, err)
					t3, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{}, nil)
					require.NoError(t, err)

					transactions, err := db.ListTransactions(ctx)
					require.NoError(t, err)

					q := map[arangodb.TransactionID]arangodb.Transaction{}
					for _, transaction := range transactions {
						q[transaction.ID()] = transaction
					}

					_, ok := q[t1.ID()]
					require.True(t, ok)

					_, ok = q[t2.ID()]
					require.True(t, ok)

					_, ok = q[t3.ID()]
					require.True(t, ok)
				})
			})
		})
	})
}

func ensureTransactionStatus(t testing.TB, db arangodb.Database, tid arangodb.TransactionID, status arangodb.TransactionStatus) {
	withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
		transaction, err := db.Transaction(ctx, tid)
		require.NoError(t, err)

		s, err := transaction.Status(ctx)
		require.NoError(t, err)

		require.Equal(t, status, s.Status)
	})
}

func abortTransaction(t testing.TB, transaction arangodb.Transaction) {
	withContextT(t, 10*time.Second, func(ctx context.Context, t testing.TB) {
		require.NoError(t, transaction.Abort(ctx, nil))
	})
}

func databaseReplication2Required(t *testing.T, c arangodb.Client, ctx context.Context) {
	skipBelowVersion(c, context.Background(), "3.12.0", t)
	requireClusterMode(t)

	dbName := "replication2" + GenerateUUID("test-db")
	opts := arangodb.CreateDatabaseOptions{Options: arangodb.CreateDatabaseDefaultOptions{
		ReplicationVersion: arangodb.DatabaseReplicationVersionTwo,
	}}

	db, err := c.CreateDatabase(ctx, dbName, &opts)
	if err == nil {
		require.NoErrorf(t, db.Remove(ctx), "failed to remove testing replication2 database")
		return
	}

	if strings.Contains(err.Error(), "Replication version 2 is disabled in this binary") {
		t.Skipf("ArangoDB is not launched with the option --database.default-replication-version=2")
	}

	// Some other error that has not been expected.
	require.NoError(t, err)
}
