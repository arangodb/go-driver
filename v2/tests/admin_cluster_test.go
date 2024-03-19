//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_ClusterHealth(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

			health, err := client.Health(ctx)
			require.NoError(t, err, "Health failed")
			require.NotNil(t, health, "Health did not return a health")
			require.NotEmpty(t, health.ID, "Health did not return a cluster id")

			agents := 0
			dbServers := 0
			coordinators := 0
			for _, sh := range health.Health {
				v, err := client.Version(ctx)
				require.NoError(t, err, "Version failed")
				if v.Version.CompareTo(sh.Version) != 0 {
					t.Logf("Server version differs from `_api/version`, got `%s` instead `%s`", v.Version, sh.Version)
				}

				switch sh.Role {
				case arangodb.ServerRoleAgent:
					agents++
				case arangodb.ServerRoleDBServer:
					dbServers++
				case arangodb.ServerRoleCoordinator:
					coordinators++
				}
			}

			require.GreaterOrEqual(t, agents, 1, "Health did not return at least one agent")
			require.GreaterOrEqual(t, dbServers, 1, "Health did not return at least one dbServer")
			require.GreaterOrEqual(t, coordinators, 1, "Health did not return at least one coordinator")
		})

	})
}

func Test_ClusterDatabaseInventory(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

			health, err := client.Health(ctx)
			require.NoError(t, err, "Health failed")
			require.NotNil(t, health, "Health did not return a health")
			require.NotEmpty(t, health.ID, "Health did not return a cluster id")

			inv, err := client.DatabaseInventory(ctx, "_system")
			require.NoError(t, err, "DatabaseInventory failed")
			require.NotNil(t, inv, "DatabaseInventory did not return a inventory")
			require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

			for _, col := range inv.Collections {
				require.Greater(t, len(col.Parameters.Shards), 0, "Expected 1 or more shards in collection %s, got 0", col.Parameters.Name)

				for shardID, dbServers := range col.Parameters.Shards {
					for _, serverID := range dbServers {
						require.Contains(t, health.Health, serverID, "Unexpected dbServer ID for shard '%s': %s", shardID, serverID)
					}
				}
			}
		})
	})
}
