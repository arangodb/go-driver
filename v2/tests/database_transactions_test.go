//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_DatabaseTransactions_DataIsolation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			t.Run("Transaction", func(t *testing.T) {
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
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
							tCol, err := transaction.Collection(ctx, col.Name())
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
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
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
							tCol, err := transaction.Collection(ctx, col.Name())
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
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
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
								tCol, err := transaction.Collection(ctx, col.Name())
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
			withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
				WithCollection(t, db, nil, func(col arangodb.Collection) {
					d := document{
						basicDocument: basicDocument{Key: uuid.New().String()},
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

					col1, err := t1.Collection(ctx, col.Name())
					require.NoError(t, err)

					t2, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{Write: []string{col.Name()}}, &arangodb.BeginTransactionOptions{
						LockTimeoutDuration: 1 * time.Second,
					})
					require.NoError(t, err)
					defer abortTransaction(t, t1)

					col2, err := t2.Collection(ctx, col.Name())
					require.NoError(t, err)

					_, err = col1.UpdateDocument(ctx, d.Key, ud)
					require.NoError(t, err)

					_, err = col2.UpdateDocument(ctx, d.Key, ud)
					require.Error(t, err)
					require.True(t, shared.IsOperationTimeout(err))
				})
			})
		})
	})
}

func Test_DatabaseTransactions_List(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, 30*time.Second, func(ctx context.Context, _ testing.TB) {
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
