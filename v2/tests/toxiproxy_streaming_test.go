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
	"strings"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
)

type streamingPayloadDoc struct {
	Key     string `json:"_key"`
	Payload string `json:"payload"`
	Index   int    `json:"index"`
}

// streamingTestDoc builds a document with a fixed-size payload for streaming tests.
func streamingTestDoc(keyPrefix string, index int, payloadBytes int) streamingPayloadDoc {
	return streamingPayloadDoc{
		Key:     fmt.Sprintf("%s_%04d", keyPrefix, index),
		Payload: strings.Repeat("x", payloadBytes),
		Index:   index,
	}
}

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

				var doc streamingPayloadDoc
				for i := 0; i < toxiproxyStreamingDocsBeforeDisconnect; i++ {
					_, err := cursor.ReadDocument(readCtx, &doc)
					require.NoError(tb, err, "read document %d before disconnect", i)
				}

				require.NoError(tb, proxy.Disable())

				require.NotPanics(tb, func() {
					_, err := cursor.ReadDocument(readCtx, &doc)
					require.Error(tb, err)
					tb.Logf("error message: %v", err)
					require.True(tb, isStreamingInterruptError(err),
						"expected cursor error after disconnect during iteration, got: %v", err)
				})

				require.NoError(tb, proxy.Enable())
				waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())
			})
		})
	})
}

// TestToxiproxy_DisconnectDuringQueryExecution validates that db.Query returns a clean error
// (no panic) when the network drops before the cursor handle is returned.
func TestToxiproxy_DisconnectDuringQueryExecution(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testDisconnectDuringQueryExecution)
}

// testDisconnectDuringQueryExecution starts a slow AQL query and disables the proxy while the
// initial POST /_api/cursor is still in flight.
func testDisconnectDuringQueryExecution(t *testing.T, connFactory toxiproxyConnectionFactory) {
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
			queryCtx, cancelQuery := context.WithTimeout(ctx, toxiproxyStreamingInterruptTimeout)
			defer cancelQuery()

			queryDone := make(chan error, 1)
			go func() {
				_, err := db.Query(queryCtx, streamingSlowStartingQuery(), nil)
				queryDone <- err
			}()

			time.Sleep(toxiproxyQueryStartupDisconnectDelay)
			require.NoError(tb, proxy.Disable())

			require.NotPanics(tb, func() {
				select {
				case err := <-queryDone:
					require.Error(tb, err)
					tb.Logf("error message: %s", err.Error())
					require.True(tb, isStreamingInterruptError(err),
						"expected query startup interrupted with clean error, got: %v", err)
				case <-time.After(toxiproxyStreamingInterruptTimeout):
					require.Fail(tb, "expected db.Query to fail after disconnect during query startup")
				}
			})

			require.NoError(tb, proxy.Enable())
			waitForSuccessfulVersion(tb, client, toxiproxyRecoveryTimeout())
		})
	})
}

// seedStreamingCollection inserts documents for cursor streaming tests.
func seedStreamingCollection(t testing.TB, col arangodb.Collection) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyStreamingOperationTimeout())
	defer cancel()

	docs := make([]any, toxiproxyStreamingDocCount)
	for i := 0; i < toxiproxyStreamingDocCount; i++ {
		docs[i] = streamingTestDoc("stream", i, toxiproxyStreamingPayloadBytes)
	}

	_, err := col.CreateDocuments(ctx, docs)
	require.NoError(t, err)
}

// streamingCollectionQuery returns an AQL query that streams all documents from a collection.
func streamingCollectionQuery(collectionName string) string {
	return fmt.Sprintf("FOR d IN `%s` RETURN d", collectionName)
}

// streamingSlowStartingQuery returns an AQL query that blocks on the server before returning
// a cursor, so the driver stays inside db.Query() long enough to inject a network fault.
func streamingSlowStartingQuery() string {
	return fmt.Sprintf("FOR i IN 1..1 RETURN { i: i, burn: SLEEP(%g) }", toxiproxyQueryStartupSleepSeconds)
}
