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
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/stretchr/testify/require"
)

type bandwidthPayloadDoc struct {
	Key     string `json:"_key"`
	Payload string `json:"payload"`
	Index   int    `json:"index"`
}

// TestToxiproxy_SlowUpload validates that upstream bandwidth limiting slows bulk inserts
// without corrupting stored documents.
func TestToxiproxy_SlowUpload(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testSlowUpload)
}

// testSlowUpload measures insert duration with and without a 20 KB/s upload cap.
func testSlowUpload(t *testing.T, connFactory toxiproxyConnectionFactory) {
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
				baseline, _ := measureBandwidthBulkInsert(tb, col, "base", toxiproxyBandwidthUploadDocCount)

				addUpstreamBandwidth(t, proxy, "bandwidth_up", toxiproxyBandwidthLimitKBs)
				t.Cleanup(func() {
					_ = proxy.RemoveToxic("bandwidth_up")
				})

				throttled, throttledKeys := measureBandwidthBulkInsert(tb, col, "slow", toxiproxyBandwidthUploadDocCount)
				requireBandwidthSlowdown(tb, baseline, throttled)
				verifyBandwidthDocuments(tb, col, throttledKeys, "slow", toxiproxyBandwidthUploadPayloadBytes)
			})
		})
	})
}

// TestToxiproxy_SlowDownload validates that downstream bandwidth limiting slows large reads.
func TestToxiproxy_SlowDownload(t *testing.T) {
	runToxiproxyWithHTTPProtocols(t, testSlowDownload)
}

// testSlowDownload measures full-collection cursor reads with and without a download cap.
func testSlowDownload(t *testing.T, connFactory toxiproxyConnectionFactory) {
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
				keys := seedBandwidthDownloadCollection(tb, col)

				baseline := measureBandwidthCollectionRead(tb, db, col.Name())

				addDownstreamBandwidth(t, proxy, "bandwidth_down", toxiproxyBandwidthLimitKBs)
				t.Cleanup(func() {
					_ = proxy.RemoveToxic("bandwidth_down")
				})

				throttled := measureBandwidthCollectionRead(tb, db, col.Name())
				requireBandwidthSlowdown(tb, baseline, throttled)
				verifyBandwidthDocuments(tb, col, keys, "read", toxiproxyBandwidthDownloadPayloadBytes)
			})
		})
	})
}

// bandwidthTestDoc builds a document with a fixed-size payload for bandwidth tests.
func bandwidthTestDoc(keyPrefix string, index int, payloadBytes int) bandwidthPayloadDoc {
	return bandwidthPayloadDoc{
		Key:     fmt.Sprintf("%s_%04d", keyPrefix, index),
		Payload: strings.Repeat("x", payloadBytes),
		Index:   index,
	}
}

// measureBandwidthBulkInsert inserts documents and returns duration plus document keys.
func measureBandwidthBulkInsert(t testing.TB, col arangodb.Collection, keyPrefix string, docCount int) (time.Duration, []string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	docs := make([]any, docCount)
	keys := make([]string, docCount)
	for i := 0; i < docCount; i++ {
		doc := bandwidthTestDoc(keyPrefix, i, toxiproxyBandwidthUploadPayloadBytes)
		docs[i] = doc
		keys[i] = doc.Key
	}

	start := time.Now()
	_, err := col.CreateDocuments(ctx, docs)
	require.NoError(t, err, "bulk insert failed for prefix %s", keyPrefix)

	return time.Since(start), keys
}

// seedBandwidthDownloadCollection inserts a large collection used for download throttling tests.
func seedBandwidthDownloadCollection(t testing.TB, col arangodb.Collection) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	keys := make([]string, toxiproxyBandwidthDownloadDocCount)
	docs := make([]any, toxiproxyBandwidthDownloadDocCount)
	for i := 0; i < toxiproxyBandwidthDownloadDocCount; i++ {
		doc := bandwidthTestDoc("read", i, toxiproxyBandwidthDownloadPayloadBytes)
		docs[i] = doc
		keys[i] = doc.Key
	}

	_, err := col.CreateDocuments(ctx, docs)
	require.NoError(t, err)
	return keys
}

// measureBandwidthCollectionRead times reading all documents from a collection via AQL.
func measureBandwidthCollectionRead(t testing.TB, db arangodb.Database, collectionName string) time.Duration {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	query := fmt.Sprintf("FOR d IN `%s` RETURN d", collectionName)
	start := time.Now()

	cursor, err := db.Query(ctx, query, nil)
	require.NoError(t, err)
	defer cursor.Close()

	var doc bandwidthPayloadDoc
	for {
		_, err := cursor.ReadDocument(ctx, &doc)
		if shared.IsNoMoreDocuments(err) {
			break
		}
		require.NoError(t, err)
	}

	return time.Since(start)
}

// requireBandwidthSlowdown asserts throttled operations take materially longer than baseline.
func requireBandwidthSlowdown(t testing.TB, baseline, throttled time.Duration) {
	t.Helper()

	minThrottled := time.Duration(float64(baseline) * toxiproxyBandwidthMinSlowdownFactor)
	if minThrottled < 2*time.Second {
		minThrottled = 2 * time.Second
	}
	require.Greater(t, throttled, minThrottled,
		"expected throttled duration > %v (baseline %v, throttled %v)", minThrottled, baseline, throttled)
}

// verifyBandwidthDocuments reads documents back and checks payload integrity.
func verifyBandwidthDocuments(t testing.TB, col arangodb.Collection, keys []string, keyPrefix string, payloadBytes int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), toxiproxyBandwidthOperationTimeout())
	defer cancel()

	wantPayload := strings.Repeat("x", payloadBytes)
	for _, key := range keys {
		var got bandwidthPayloadDoc
		_, err := col.ReadDocument(ctx, key, &got)
		require.NoError(t, err, "read back document %s", key)

		require.True(t, strings.HasPrefix(got.Key, keyPrefix+"_"), "unexpected key %s", got.Key)
		require.Len(t, got.Payload, payloadBytes, "payload corrupted for key %s", key)
		require.Equal(t, wantPayload, got.Payload, "payload content corrupted for key %s", key)
	}
}
