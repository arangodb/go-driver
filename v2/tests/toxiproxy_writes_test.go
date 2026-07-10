//go:build toxiproxy

//
// DISCLAIMER
//
// Copyright 2026 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/stretchr/testify/require"
)

type writeInterruptDoc struct {
	Key   string `json:"_key"`
	Value string `json:"value"`
}

// TestToxiproxy_DisconnectDuringInsert validates that CreateDocument returns a clean
// network error (no panic/hang) when the connection drops while the write is in flight.
//
// Important: after a mid-flight disconnect the client cannot know whether the server
// committed the document before the response was lost. The test asserts only a clean
// transport error and recovery — not document presence or absence.
func TestToxiproxy_DisconnectDuringInsert(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testDisconnectDuringInsert)
}

// testDisconnectDuringInsert delays the response path, starts CreateDocument, then
// disables the proxy so the client sees a network error while the outcome may be unknown.
//
// Downstream (server→client) latency is intentional: the request can reach ArangoDB and
// possibly commit before the response returns, so Disable() mid-flight models a lost
// reply with an unknown write outcome. A short sleep before Disable is a timing window
// (same pattern as DisconnectDuringQueryExecution); it is acceptable for this resiliency
// test but could flake under extreme scheduling delay.
func testDisconnectDuringInsert(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, toxiproxyRecoveryTimeout())

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				doc := writeInterruptDoc{
					Key:   fmt.Sprintf("insert_cut_%d", time.Now().UnixNano()),
					Value: "disconnect-during-insert",
				}

				addDownstreamLatency(tb, proxy, "latency_down", toxiproxyWriteResponseLatencyMs)

				writeCtx, cancelWrite := context.WithTimeout(ctx, toxiproxyWriteInterruptTimeout)
				defer cancelWrite()

				writeDone := make(chan error, 1)
				go func() {
					_, err := col.CreateDocument(writeCtx, doc)
					writeDone <- err
				}()

				// Give the request time to leave the client / reach the server before the cut.
				time.Sleep(toxiproxyWriteDisconnectDelay)
				require.NoError(tb, proxy.Disable())

				require.NotPanics(tb, func() {
					select {
					case err := <-writeDone:
						require.Error(tb, err)
						tb.Logf("error message: %s", err.Error())
						require.True(tb, isWriteInterruptError(err),
							"expected clean network error during insert, got: %v", err)
						tb.Logf("write outcome cannot be determined after transport interruption (server may or may not have committed)")
					case <-time.After(toxiproxyWriteInterruptTimeout):
						require.Fail(tb, "expected CreateDocument to fail after disconnect during insert")
					}
				})

				require.NoError(tb, proxy.Enable())
				// Remove the latency toxic before continuing with recovery checks.
				require.NoError(tb, proxy.RemoveToxic("latency_down"))
				waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())

				// Informational only — do not assert committed vs not committed.
				readCtx, cancelRead := context.WithTimeout(ctx, 15*time.Second)
				defer cancelRead()
				var got writeInterruptDoc
				_, readErr := col.ReadDocument(readCtx, doc.Key, &got)
				if readErr == nil {
					tb.Logf("after recovery: document %s is present (the document exists after recovery)", doc.Key)
				} else if shared.IsNotFound(readErr) {
					tb.Logf("after recovery: document %s is absent (the document is not visible after recovery)", doc.Key)
				} else {
					tb.Logf("after recovery: could not determine document state for %s: %v", doc.Key, readErr)
				}
			})
		})
	})
}

// TestToxiproxy_DisconnectDuringTransactionCommit validates that Commit returns a clean
// network error (no panic/deadlock) when the connection drops while commit is in flight.
//
// Important: the commit outcome may be unknown — the transaction could have been committed
// on the server before the response was lost.
func TestToxiproxy_DisconnectDuringTransactionCommit(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testDisconnectDuringTransactionCommit)
}

// testDisconnectDuringTransactionCommit begins a transaction, writes a document, then
// cuts the network during Commit while the response path is delayed.
//
// Downstream latency is intentional (see DisconnectDuringInsert): the commit request can
// succeed on the server before the client receives the response. The short sleep before
// Disable is a timing window shared with other mid-flight disconnect tests.
func testDisconnectDuringTransactionCommit(t *testing.T, connFactory toxiproxyConnectionFactory) {
	tp := newToxiproxyEnv(t)
	proxy := tp.proxy(t)
	t.Cleanup(func() {
		ensureProxyEnabled(t, proxy)
		clearProxyToxics(t, proxy)
	})

	client := newToxiproxyClient(t, connFactory)
	waitForSuccessfulVersion(t, client, toxiproxyRecoveryTimeout())

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				doc := writeInterruptDoc{
					Key:   fmt.Sprintf("txn_commit_cut_%d", time.Now().UnixNano()),
					Value: "disconnect-during-commit",
				}

				txn, err := db.BeginTransaction(ctx, arangodb.TransactionCollections{
					Write: []string{col.Name()},
				}, &arangodb.BeginTransactionOptions{
					WaitForSync: true,
				})
				require.NoError(tb, err)
				tb.Logf("began transaction ID=%s", txn.ID())

				tCol, err := txn.GetCollection(ctx, col.Name(), nil)
				require.NoError(tb, err)
				_, err = tCol.CreateDocument(ctx, doc)
				require.NoError(tb, err)

				addDownstreamLatency(tb, proxy, "latency_down", toxiproxyWriteResponseLatencyMs)

				commitCtx, cancelCommit := context.WithTimeout(ctx, toxiproxyWriteInterruptTimeout)
				defer cancelCommit()

				commitDone := make(chan error, 1)
				go func() {
					commitDone <- txn.Commit(commitCtx, nil)
				}()

				// Give the commit request time to leave the client / reach the server before the cut.
				time.Sleep(toxiproxyWriteDisconnectDelay)
				require.NoError(tb, proxy.Disable())

				require.NotPanics(tb, func() {
					select {
					case err := <-commitDone:
						require.Error(tb, err)
						tb.Logf("error message: %s", err.Error())
						require.True(tb, isWriteInterruptError(err),
							"expected clean network error during transaction commit, got: %v", err)
						tb.Logf("commit outcome is unknown after mid-flight disconnect (txn may or may not be committed)")
					case <-time.After(toxiproxyWriteInterruptTimeout):
						require.Fail(tb, "expected Commit to fail after disconnect during transaction commit")
					}
				})

				require.NoError(tb, proxy.Enable())
				// Remove the latency toxic before continuing with recovery checks.
				require.NoError(tb, proxy.RemoveToxic("latency_down"))
				waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())

				// Best effort: if Commit never reached the server, the transaction may
				// still be open. Ignore Abort errors because the transaction may already
				// have been committed or closed.
				abortCtx, cancelAbort := context.WithTimeout(ctx, 15*time.Second)
				defer cancelAbort()
				if abortErr := txn.Abort(abortCtx, nil); abortErr != nil {
					tb.Logf("post-recovery Abort (best-effort): %v", abortErr)
				}

				// Informational only — do not assert committed vs aborted.
				readCtx, cancelRead := context.WithTimeout(ctx, 15*time.Second)
				defer cancelRead()
				var got writeInterruptDoc
				_, readErr := col.ReadDocument(readCtx, doc.Key, &got)
				if readErr == nil {
					tb.Logf("after recovery: document %s is visible (commit likely succeeded before cut)", doc.Key)
				} else if shared.IsNotFound(readErr) {
					tb.Logf("after recovery: document %s is not visible (commit likely did not land)", doc.Key)
				} else {
					tb.Logf("after recovery: could not determine document state for %s: %v", doc.Key, readErr)
				}
			})
		})
	})
}
