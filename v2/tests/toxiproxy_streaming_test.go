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

// TestToxiproxy_DisconnectDuringCursorIteration validates that ReadDocument returns a clean
// error (no panic) when the network drops mid-cursor.
func TestToxiproxy_DisconnectDuringCursorIteration(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testDisconnectDuringCursorIteration)
}

// testDisconnectDuringCursorIteration opens a streaming cursor, reads several batches, then
// disables the proxy before the next ReadDocument call.
func testDisconnectDuringCursorIteration(t *testing.T, connFactory toxiproxyConnectionFactory) {
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
				seedStreamingCollection(tb, col)

				readCtx, cancelRead := context.WithTimeout(ctx, toxiproxyStreamingInterruptTimeout)
				defer cancelRead()

				cursor, err := db.Query(readCtx, streamingCollectionQuery(col.Name()), &arangodb.QueryOptions{
					BatchSize: 1,
				})
				require.NoError(tb, err)
				defer cursor.Close()

				var doc bandwidthPayloadDoc
				for i := 0; i < toxiproxyStreamingDocsBeforeDisconnect; i++ {
					_, err := cursor.ReadDocument(readCtx, &doc)
					require.NoError(tb, err, "read document %d before disconnect", i)
				}

				require.NoError(tb, proxy.Disable())

				require.NotPanics(tb, func() {
					_, err := cursor.ReadDocument(readCtx, &doc)
					t.Logf("error message: %s", err.Error())
					require.Error(tb, err)
					require.True(tb, isStreamingInterruptError(err),
						"expected cursor error after disconnect during iteration, got: %v", err)
				})

				require.NoError(tb, proxy.Enable())
				waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())
			})
		})
	})
}

// TestToxiproxy_DisconnectDuringLargeAQLQuery validates that a long-running query is
// interrupted with a clean error when the connection drops mid-stream.
func TestToxiproxy_DisconnectDuringLargeAQLQuery(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testDisconnectDuringLargeAQLQuery)
}

// testDisconnectDuringLargeAQLQuery runs a CPU-heavy AQL query and drops the proxy while
// the cursor is actively streaming results.
func testDisconnectDuringLargeAQLQuery(t *testing.T, connFactory toxiproxyConnectionFactory) {
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
				seedStreamingSlowQueryCollection(tb, col)

				readCtx, cancelRead := context.WithTimeout(ctx, toxiproxyStreamingCursorOpenTimeout)
				defer cancelRead()

				queryFailed := make(chan error, 1)
				cursorOpen := make(chan struct{})

				go func() {
					cursor, err := db.Query(readCtx, streamingSlowAQLQuery(col.Name()), &arangodb.QueryOptions{
						BatchSize: 1,
					})
					if err != nil {
						queryFailed <- err
						return
					}
					defer cursor.Close()
					close(cursorOpen)

					var doc map[string]any
					for {
						_, err := cursor.ReadDocument(readCtx, &doc)
						if shared.IsNoMoreDocuments(err) {
							queryFailed <- fmt.Errorf("slow query finished before disconnect")
							return
						}
						if err != nil {
							queryFailed <- err
							return
						}
					}
				}()

				select {
				case <-cursorOpen:
				case err := <-queryFailed:
					require.Fail(tb, "slow query failed before cursor opened: %v", err)
				case <-time.After(toxiproxyStreamingCursorOpenTimeout):
					require.Fail(tb, "cursor did not open before timeout")
				}

				require.NoError(tb, proxy.Disable())

				require.NotPanics(tb, func() {
					select {
					case err := <-queryFailed:
						require.Error(tb, err)
						tb.Logf("error message: %s", err.Error())
						require.True(tb, isStreamingInterruptError(err),
							"expected query interrupted with clean error, got: %v", err)
					case <-time.After(toxiproxyStreamingInterruptTimeout):
						require.Fail(tb, "expected query to fail after disconnect during large AQL query")
					}
				})

				cancelRead()
				require.NoError(tb, proxy.Enable())
				waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())
			})
		})
	})
}

// seedStreamingCollection inserts documents for cursor streaming tests.
func seedStreamingCollection(t testing.TB, col arangodb.Collection) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	docs := make([]any, toxiproxyStreamingDocCount)
	for i := 0; i < toxiproxyStreamingDocCount; i++ {
		docs[i] = bandwidthTestDoc("stream", i, toxiproxyBandwidthDownloadPayloadBytes)
	}

	_, err := col.CreateDocuments(ctx, docs)
	require.NoError(t, err)
}

// seedStreamingSlowQueryCollection inserts documents for slow AQL streaming tests.
func seedStreamingSlowQueryCollection(t testing.TB, col arangodb.Collection) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	require.NoError(t, arangodb.CreateDocuments(ctx, col, toxiproxyStreamingSlowQueryDocCount, func(index int) any {
		return map[string]any{"value": index}
	}))
}

// streamingCollectionQuery returns an AQL query that streams all documents from a collection.
func streamingCollectionQuery(collectionName string) string {
	return fmt.Sprintf("FOR d IN `%s` RETURN d", collectionName)
}

// streamingSlowAQLQuery returns a native AQL query that keeps the coordinator busy per document.
func streamingSlowAQLQuery(collectionName string) string {
	return fmt.Sprintf(
		"FOR doc IN `%s` SORT doc.value LET burn = (FOR i IN 1..%d LET x = MD5(CONCAT(TO_STRING(doc.value), TO_STRING(i))) FILTER x != null RETURN 1) RETURN doc",
		collectionName,
		toxiproxyStreamingSlowQueryBurnIterations,
	)
}
