//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
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
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
)

// TestClientDatabase tests basic database functionality.
func TestClientDatabase(t *testing.T) {
	Wrap(t, func(t *testing.T, c arangodb.Client) {
		withContext(30*time.Second, func(ctx context.Context) error {
			random := uuid.New().String()
			name := "database_" + random

			// Create the database.
			db, err := c.CreateDatabase(nil, name, nil)
			require.NoErrorf(t, err, "failed to create the database: %s", name)

			list, err := c.Databases(ctx)
			require.NoError(t, err, "failed to fetch databases")
			require.GreaterOrEqualf(t, len(list), 2, "Two databases should exist: _system and %s", name)

			list, err = c.AccessibleDatabases(ctx)
			require.NoError(t, err, "failed to fetch accessible databases")
			require.GreaterOrEqualf(t, len(list), 2, "Two databases should exist: _system and %s", name)

			for _, n := range []string{"_system", name} {
				d, err := c.Database(ctx, n)
				require.NoErrorf(t, err, "failed to fetch database %s", n)
				require.Equalf(t, n, d.Name(), "database name is not the same")

				found, err := c.DatabaseExists(ctx, n)
				require.NoErrorf(t, err, "failed to check existence of the database %s", n)
				require.Truef(t, found, "the database %s must exist", n)
			}

			db.Remove(ctx)

			return nil
		})
	})
}
