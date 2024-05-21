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
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
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

	t.Logf("Creating DB %s", name)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		db, err := client.CreateDatabase(ctx, name, opts)
		require.NoError(t, err, fmt.Sprintf("Failed to create DB %s", name))

		defer func() {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				t.Logf("Removing DB %s", db.Name())
				require.NoError(t, db.Remove(ctx))
			})
		}()

		f(db)
	})
}

func WithCollection(t testing.TB, db arangodb.Database, props *arangodb.CreateCollectionProperties, f func(col arangodb.Collection)) {
	name := GenerateUUID("test-COL")

	t.Logf("Creating COL %s", name)

	withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
		col, err := db.CreateCollection(ctx, name, props)
		require.NoError(t, err, fmt.Sprintf("Failed to create COL %s", name))

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

		f(g)
	})
}

func WaitForHealthyCluster(t *testing.T, client arangodb.Client, timeout time.Duration) {
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

				// check server availability
				t.Logf("Checking server availability %s", id)

				urlEndpoint := connection.NewUrl("_admin", "server", "availability")
				req, err := client.Connection().NewRequestWithEndpoint(utils.FixupEndpointURLScheme(server.Endpoint), http.MethodGet, urlEndpoint)
				require.NoError(t, err)

				_, err = client.Connection().Do(ctx, req, nil, http.StatusOK)
				if err != nil {
					t.Logf("Server %s is not available", id)
					return nil
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

func newBool(b bool) *bool {
	return newT(b)
}

func newT[T any](val T) *T {
	return &val
}

func newVersion(val string) *arangodb.Version {
	return newT(arangodb.Version(val))
}

func newInt(i int) *int {
	return &i
}

func newInt64(i int64) *int64 {
	return &i
}
