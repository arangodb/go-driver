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

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

func Test_CreateNewBatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole is %s\n", serverRole)

			var dbServer *string
			state := utils.NewType(true)

			if serverRole == arangodb.ServerRoleCoordinator {
				clusterHealth, err := client.Health(ctx) // Ensure the client is healthy
				require.NoError(t, err)
				for id, db := range clusterHealth.Health {
					if db.Role == arangodb.ServerRoleDBServer {
						s := string(id)
						dbServer = &s
						break
					}
				}
			}

			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)

			batch, err := client.CreateNewBatch(ctx, db.Name(), dbServer, state, arangodb.CreateNewBatchOptions{
				Ttl: 300,
			})
			require.NoError(t, err)
			require.NotNil(t, batch)
			require.NotEmpty(t, batch.ID)
			require.NotEmpty(t, batch.LastTick)
			require.NotNil(t, batch.State)

			t.Run("GetInventory", func(t *testing.T) {
				resp, err := client.GetInventory(ctx, db.Name(), arangodb.InventoryQueryParams{
					BatchID:  batch.ID,
					DBserver: dbServer,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
			})

			t.Run("DeleteBatch", func(t *testing.T) {
				err := client.DeleteBatch(ctx, db.Name(), dbServer, batch.ID)
				require.NoError(t, err)
			})
		})
	})
}
