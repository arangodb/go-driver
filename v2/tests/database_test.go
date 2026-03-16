//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func TestGetDatabase(t *testing.T) {
	Wrap(t, func(t *testing.T, c arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			// The database should not be found
			_, err := c.GetDatabase(ctx, "wrong-name", nil)
			require.NotNil(t, err)

			// IsExist validation should be skipped
			_, err = c.GetDatabase(ctx, "wrong-name", &arangodb.GetDatabaseOptions{SkipExistCheck: true})
			require.Nil(t, err)
		})
	})
}

// --database.extended-names-databases=true are enabled by default in 3.12
func TestDatabaseNameUnicode(t *testing.T) {
	Wrap(t, func(t *testing.T, c arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			skipBelowVersion(c, ctx, "3.12", t)

			random := GenerateUUID("test-db-unicode")
			dbName := "\u006E\u0303\u00f1" + random
			_, err := c.CreateDatabase(ctx, dbName, nil)
			require.EqualError(t, err, "database name is not properly UTF-8 NFC-normalized")

			normalized := norm.NFC.String(dbName)
			_, err = c.CreateDatabase(ctx, normalized, nil)
			require.NoError(t, err)

			// The database should not be found by the not normalized name.
			_, err = c.GetDatabase(ctx, dbName, nil)
			require.NotNil(t, err)

			// The database should be found by the normalized name.
			exist, err := c.DatabaseExists(ctx, normalized)
			require.NoError(t, err)
			require.True(t, exist)

			var found bool
			databases, err := c.Databases(ctx)
			require.NoError(t, err)
			for _, database := range databases {
				if database.Name() == normalized {
					found = true
					break
				}
			}
			require.Truef(t, found, "the database %s should have been found", normalized)

			// The database should return handler to the database by the normalized name.
			db, err := c.GetDatabase(ctx, normalized, nil)
			require.NoError(t, err)
			require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
		})
	})
}

func TestDatabaseSharding_Defaults(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, c arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			skipNoEnterprise(c, ctx, t)

			type scenario struct {
				name     string
				sharding arangodb.DatabaseSharding
				allowed  []arangodb.DatabaseSharding
			}

			scenarios := []scenario{
				{
					name:     "none",
					sharding: arangodb.DatabaseShardingNone,
					allowed:  []arangodb.DatabaseSharding{arangodb.DatabaseShardingNone},
				},
				{
					name:     "single",
					sharding: arangodb.DatabaseShardingSingle,
					allowed:  []arangodb.DatabaseSharding{arangodb.DatabaseShardingNone, arangodb.DatabaseShardingSingle},
				},
				{
					name:     "flexible",
					sharding: arangodb.DatabaseShardingFlexible,
					allowed:  []arangodb.DatabaseSharding{arangodb.DatabaseShardingNone, arangodb.DatabaseShardingFlexible},
				},
			}

			for _, sc := range scenarios {
				t.Run(sc.name, func(t *testing.T) {
					dbName := GenerateUUID(fmt.Sprintf("test-db-sharding-%s", sc.name))
					opts := &arangodb.CreateDatabaseOptions{
						Options: arangodb.CreateDatabaseDefaultOptions{
							Sharding: sc.sharding,
						},
					}

					db, err := c.CreateDatabase(ctx, dbName, opts)
					if sc.name == "flexible" && err != nil {
						if ok, arangoErr := shared.IsArangoError(err); ok {
							t.Skipf("sharding=flexible not supported by this server/version: code=%d errorNum=%d message=%s", arangoErr.Code, arangoErr.ErrorNum, arangoErr.ErrorMessage)
						}
					}
					require.NoError(t, err)

					t.Cleanup(func() {
						err := db.Remove(ctx)
						if err != nil {
							t.Logf("failed to remove db %s: %v", db.Name(), err)
						}
					})

					info, err := db.Info(ctx)
					require.NoError(t, err)
					t.Logf("database sharding raw value: %q", info.Sharding)
					require.Containsf(t, sc.allowed, info.Sharding, "server returned unexpected sharding value for input %q", sc.sharding)
				})
			}
		})
	})
}
