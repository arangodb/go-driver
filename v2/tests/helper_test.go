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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

var (
	generateLock sync.Mutex
	generateID   uint64
)

func GenerateUUID(prefix string) string {
	generateLock.Lock()
	defer generateLock.Unlock()

	generateID++

	if prefix == "" {
		prefix = "test"
	}

	return fmt.Sprintf("%s-%s-%04d", prefix, uuid.New().String(), generateID)
}

func WithDatabase(t testing.TB, client arangodb.Client, opts *arangodb.CreateDatabaseOptions, f func(db arangodb.Database)) {
	name := GenerateUUID("test-DB")

	t.Logf("Creating DB %s, time: %s", name, time.Now())

	withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		db, err := client.CreateDatabase(ctx, name, opts)
		require.NoError(t, err, fmt.Sprintf("Failed to create DB %s: %s", name, err))

		defer func() {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute*2)
				defer cancel()
				err := db.Remove(timeoutCtx)
				if err != nil {
					t.Logf("Removing DB %s failed, time: %s with %s", db.Name(), time.Now(), err)
				}
			})
		}()

		f(db)
	})
}

func WithCollectionV2(t testing.TB, db arangodb.Database, props *arangodb.CreateCollectionPropertiesV2, f func(col arangodb.Collection)) {
	name := GenerateUUID("test-COL")

	t.Logf("Creating COL %s, time: %s", name, time.Now())

	withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		// In cluster mode, collection creation may need retries due to resource contention
		// when running tests in parallel (TESTV2PARALLEL > 1)
		var col arangodb.Collection
		var err error

		if getTestMode() == string(testModeCluster) {
			// Use retry logic for cluster mode to handle resource contention
			// when running tests in parallel (TESTV2PARALLEL > 1)
			// Collection creation can fail temporarily due to cluster coordination delays
			retryInterval := 250 * time.Millisecond
			retryTimeout := 30 * time.Second

			err = NewTimeout(func() error {
				var createErr error
				col, createErr = db.CreateCollectionV2(ctx, name, props)
				if createErr == nil {
					return Interrupt{}
				}

				// Check if it's a retryable error (service unavailable, timeout, conflict, etc.)
				if ok, arangoErr := shared.IsArangoError(createErr); ok {
					// Retry on service unavailable (503), timeout (408) errors
					// Also retry on internal server errors (500)
					// These are common in cluster mode under load
					if arangoErr.Code == 503 || arangoErr.Code == 408 || arangoErr.Code == 500 {
						return nil // Retry
					}
				}

				// For other errors (like duplicate name), return them immediately
				return createErr
			}).Timeout(retryTimeout, retryInterval)

			if err != nil {
				// If timeout occurred, try one more time to get the actual error for better diagnostics
				if col == nil {
					col, err = db.CreateCollectionV2(ctx, name, props)
				}
			}
		} else {
			// Single server mode - direct creation (no retry needed)
			col, err = db.CreateCollectionV2(ctx, name, props)
		}

		require.NoError(t, err, fmt.Sprintf("Failed to create COL %s after retries", name))

		NewTimeout(func() error {
			_, err := db.GetCollection(ctx, name, nil)
			if err == nil {
				return Interrupt{}
			}

			if shared.IsNotFound(err) {
				return nil
			}

			return err
		}).TimeoutT(t, 15*time.Second, 125*time.Millisecond)

		defer func() {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				if col == nil {
					t.Log("Skipping collection removal: collection is nil")
					return
				}

				timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				defer cancel()

				if err := col.Remove(timeoutCtx); err != nil {
					t.Logf(
						"Removing Collection %s failed, time: %s, err: %v",
						col.Name(),
						time.Now(),
						err,
					)
				}
			})
		}()

		f(col)
	})
}

func WithUserDocs(t *testing.T, col arangodb.Collection, f func(users []UserDoc)) {
	users := []UserDoc{
		{Name: "John", Age: 13},
		{Name: "Jake", Age: 25},
		{Name: "Clair", Age: 12},
		{Name: "Johnny", Age: 42},
		{Name: "Blair", Age: 67},
	}

	withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
		_, err := col.CreateDocuments(ctx, users)
		require.NoError(t, err)

		f(users)
	})
}

func WithGraph(t *testing.T, db arangodb.Database, graphDef *arangodb.GraphDefinition, opts *arangodb.CreateGraphOptions, f func(g arangodb.Graph)) {
	name := db.Name() + "_graph"
	t.Logf("Creating Graph %s", name)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		g, err := db.CreateGraph(ctx, name, graphDef, opts)
		require.NoError(t, err, fmt.Sprintf("Failed to create Graph %s", name))

		defer func() {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				defer cancel()

				// Remove graph with all collections to ensure proper cleanup
				removeOpts := &arangodb.RemoveGraphOptions{
					DropCollections: true,
				}
				if err := g.Remove(timeoutCtx, removeOpts); err != nil {
					t.Logf(
						"Removing Graph %s failed, time: %s, err: %v",
						g.Name(),
						time.Now(),
						err,
					)
				}
			})
		}()

		f(g)
	})
}

func WaitForHealthyCluster(t *testing.T, client arangodb.Client, timeout time.Duration, checkAvailability bool) {
	NewTimeout(func() error {
		return withContext(time.Second*3, func(ctx context.Context) error {
			health, err := client.Health(ctx)
			if err != nil {
				return nil
			}

			for id, server := range health.Health {
				if server.Status != arangodb.ServerStatusGood {
					t.Logf("Server %s is not healthy", server.ShortName)
					return nil
				}

				if checkAvailability {
					err = client.CheckAvailability(ctx, server.Endpoint)
					if err != nil {
						t.Logf("Server %s (Endpoint: %s) is not available, err: %v", id, server.Endpoint, err)
						return nil
					}
				}
			}

			return Interrupt{}
		})
	}).TimeoutT(t, timeout, 500*time.Millisecond)

}

func getBool(b *bool, d bool) bool {
	if b == nil {
		return d
	}

	return *b
}

func newVersion(val string) *arangodb.Version {
	return utils.NewType(arangodb.Version(val))
}

// isNoAuth returns true if TEST_AUTH="none"
// Tests that require superuser rights should only run in no-auth mode
func isNoAuth() bool {
	return os.Getenv("TEST_AUTH") == "none"
}
