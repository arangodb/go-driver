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
	"time"

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

			t.Run("ExtendBatch", func(t *testing.T) {
				err := client.ExtendBatch(ctx, db.Name(), dbServer, batch.ID, arangodb.CreateNewBatchOptions{
					Ttl: 600,
				})
				require.NoError(t, err)
			})

			t.Run("GetReplicationDump", func(t *testing.T) {
				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					docs := []map[string]interface{}{
						{"_key": "doc1", "name": "Alice"},
						{"_key": "doc2", "name": "Bob"},
						{"_key": "doc3", "name": "Charlie"},
					}
					for _, doc := range docs {
						resp, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
						require.NotNil(t, resp)
					}

					// Give Arango some time to flush
					time.Sleep(200 * time.Millisecond)
					// Attempt to dump the collection
					if serverRole == arangodb.ServerRoleSingle {
						_, err := client.Dump(ctx, db.Name(), arangodb.ReplicationDumpParams{
							BatchID:    batch.ID,
							Collection: col.Name(),
						})
						require.NoError(t, err)
					} else {
						t.Skipf("Dump only allowed for single server deployments. This is a %s server", serverRole)
					}
				})
			})

			t.Run("DeleteBatch", func(t *testing.T) {
				err := client.DeleteBatch(ctx, db.Name(), dbServer, batch.ID)
				require.NoError(t, err)
			})
		})
	})
}

func Test_LoggerState(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				var dbServer *string
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
				resp, err := client.LoggerState(ctx, db.Name(), dbServer)
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.State)
				require.NotEmpty(t, resp.Server)
				require.GreaterOrEqual(t, len(resp.Clients), 0)
			})
		})
	})
}

func Test_LoggerFirstTick(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}

				resp, err := client.LoggerFirstTick(ctx, db.Name())
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.FirstTick)
			})
		})
	})
}

func Test_LoggerTickRange(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}

				resp, err := client.LoggerTickRange(ctx, db.Name())
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})
	})
}

func Test_GetApplierConfig(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole is %s\n", serverRole)

			if serverRole == arangodb.ServerRoleCoordinator {
				t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
			}
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			t.Run("Running applier config with setting global:true", func(t *testing.T) {

				resp, err := client.GetApplierConfig(ctx, db.Name(), utils.NewType(false))
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
			t.Run("Running applier config with setting global:nil", func(t *testing.T) {
				resp, err := client.GetApplierConfig(ctx, db.Name(), nil)
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})
	})
}

func Test_UpdateApplierConfig(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole is %s\n", serverRole)

			if serverRole == arangodb.ServerRoleCoordinator {
				t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
			}
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			t.Run("Update applier config with setting global:true", func(t *testing.T) {
				resp, err := client.UpdateApplierConfig(ctx, db.Name(), utils.NewType(true), arangodb.UpdateApplierConfigOptions{
					ChunkSize: utils.NewType(1234),
					AutoStart: utils.NewType(true),
					Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
					Database:  utils.NewType(db.Name()),
					Username:  utils.NewType("root"),
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
			t.Run("Update applier config with setting global:false", func(t *testing.T) {
				resp, err := client.UpdateApplierConfig(ctx, db.Name(), utils.NewType(false), arangodb.UpdateApplierConfigOptions{
					ChunkSize: utils.NewType(2596),
					AutoStart: utils.NewType(false),
					Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
					Database:  utils.NewType(db.Name()),
				})
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})
	})
}
