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
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

func Test_CreateNewBatch(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
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
							resp, err := client.Dump(ctx, db.Name(), arangodb.ReplicationDumpParams{
								BatchID:    batch.ID,
								Collection: col.Name(),
							})
							require.NoError(t, err)
							require.GreaterOrEqual(t, len(resp), 0)
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
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}
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
	})
}

func Test_UpdateApplierConfig(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}
				t.Run("Update applier config with setting global:true works for only _system database", func(t *testing.T) {
					_, err := client.UpdateApplierConfig(ctx, db.Name(), utils.NewType(true), arangodb.ApplierOptions{
						ChunkSize: utils.NewType(1234),
						AutoStart: utils.NewType(true),
						Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
						Database:  utils.NewType(db.Name()),
						Username:  utils.NewType("root"),
					})
					require.Error(t, err)
				})
				t.Run("Update applier config with setting global:false", func(t *testing.T) {
					resp, err := client.UpdateApplierConfig(ctx, db.Name(), utils.NewType(false), arangodb.ApplierOptions{
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
	})
}

func Test_ApplierStart(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)
				time.Sleep(1 * time.Second)
				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}
				batch, err := client.CreateNewBatch(ctx, db.Name(), nil, utils.NewType(true), arangodb.CreateNewBatchOptions{
					Ttl: 600,
				})
				require.NoError(t, err)
				require.NotNil(t, batch)
				t.Run("Update applier config with setting global:false", func(t *testing.T) {
					resp, err := client.UpdateApplierConfig(ctx, db.Name(), utils.NewType(false), arangodb.ApplierOptions{
						ChunkSize: utils.NewType(2596),
						AutoStart: utils.NewType(false),
						Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
						Database:  utils.NewType(db.Name()),
					})
					require.NoError(t, err)
					require.NotNil(t, resp)
				})
				t.Logf("Batch ID: %s", batch.ID)
				t.Run("Applier Start with query params", func(t *testing.T) {
					resp, err := client.ApplierStart(ctx, db.Name(), utils.NewType(false), utils.NewType(batch.ID))
					require.NoError(t, err)
					require.NotNil(t, resp)
					// Log useful debug info
					t.Logf("Applier start:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*resp.State.Running,
						*resp.State.Phase,
						*resp.State.Progress.Message,
						*resp.State.Progress.FailedConnects,
					)
				})
				t.Run("Applier_State_with_query_params", func(t *testing.T) {
					ctx := context.Background()

					state, err := client.GetApplierState(ctx, db.Name(), utils.NewType(false))
					require.NoError(t, err, "failed to get applier state")
					require.NotNil(t, state.State)

					// Log useful debug info
					t.Logf("Applier state:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*state.State.Running,
						*state.State.Phase,
						*state.State.Progress.Message,
						*state.State.Progress.FailedConnects,
					)
				})
				t.Run("Applier Stop with query params", func(t *testing.T) {
					resp, err := client.ApplierStop(ctx, db.Name(), utils.NewType(false))
					require.NoError(t, err)
					require.NotNil(t, resp)
					// Log useful debug info
					t.Logf("Applier stop:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*resp.State.Running,
						*resp.State.Phase,
						*resp.State.Progress.Message,
						*resp.State.Progress.FailedConnects,
					)
				})
				t.Run("Update applier config with out query params", func(t *testing.T) {
					resp, err := client.UpdateApplierConfig(ctx, db.Name(), nil, arangodb.ApplierOptions{
						ChunkSize: utils.NewType(2596),
						AutoStart: utils.NewType(false),
						Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
						Database:  utils.NewType(db.Name()),
					})
					require.NoError(t, err)
					require.NotNil(t, resp)
				})
				t.Logf("Batch ID: %s", batch.ID)
				t.Run("Applier Start with out query params", func(t *testing.T) {
					resp, err := client.ApplierStart(ctx, db.Name(), nil, nil)
					require.NoError(t, err)
					require.NotNil(t, resp)
					// Log useful debug info
					t.Logf("Applier start:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*resp.State.Running,
						*resp.State.Phase,
						*resp.State.Progress.Message,
						*resp.State.Progress.FailedConnects,
					)
				})
				t.Run("Applier State with out query params", func(t *testing.T) {
					ctx := context.Background()

					state, err := client.GetApplierState(ctx, db.Name(), nil)
					require.NoError(t, err, "failed to get applier state")
					require.NotNil(t, state.State)

					// Log useful debug info
					t.Logf("Applier state:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*state.State.Running,
						*state.State.Phase,
						*state.State.Progress.Message,
						*state.State.Progress.FailedConnects,
					)
				})
				t.Run("Applier Stop with out query params", func(t *testing.T) {
					resp, err := client.ApplierStop(ctx, db.Name(), nil)
					require.NoError(t, err)
					require.NotNil(t, resp)
					// Log useful debug info
					t.Logf("Applier stop:\n  running=%v\n  phase=%s\n  message=%s\n  failedConnects=%d",
						*resp.State.Running,
						*resp.State.Phase,
						*resp.State.Progress.Message,
						*resp.State.Progress.FailedConnects,
					)
				})
			})
		})
	})
}

func Test_GetReplicationServerId(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				t.Run("Get replication server ID", func(t *testing.T) {
					resp, err := client.GetReplicationServerId(ctx, db.Name())
					require.NoError(t, err)
					require.NotNil(t, resp)
					t.Logf("Replication Server ID: %s", resp)
				})
			})
		})
	})
}

func Test_MakeFollower(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}
				t.Run("Make Follower", func(t *testing.T) {
					resp, err := client.MakeFollower(ctx, db.Name(), arangodb.ApplierOptions{
						ChunkSize: utils.NewType(1234),
						Endpoint:  utils.NewType("tcp://127.0.0.1:8529"),
						Database:  utils.NewType(db.Name()),
						Username:  utils.NewType("root"),
					})
					require.NoError(t, err)
					require.NotNil(t, resp)
				})
			})
		})
	})
}

func Test_GetWALReplicationEndpoints(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)
				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}

				t.Run("Get WAL range", func(t *testing.T) {
					resp, err := client.GetWALRange(ctx, db.Name())
					require.NoError(t, err)
					require.NotNil(t, resp)
				})

				t.Run("Get WAL last tick", func(t *testing.T) {
					resp, err := client.GetWALLastTick(ctx, db.Name())
					require.NoError(t, err)
					require.NotNil(t, resp)
				})

				WithCollectionV2(t, db, nil, func(coll arangodb.Collection) {
					// WAL range before inserts
					rangeResp, err := client.GetWALRange(ctx, db.Name())
					require.NoError(t, err)
					fromTick, err := strconv.ParseInt(rangeResp.TickMax, 10, 64)
					require.NoError(t, err)
					t.Logf("Starting fromTick: %d\n", fromTick)
					t.Run("Update applier config with out query params", func(t *testing.T) {
						resp, err := client.UpdateApplierConfig(ctx, db.Name(), nil, arangodb.ApplierOptions{
							Endpoint: utils.NewType("tcp://127.0.0.1:8529"),
							Database: utils.NewType(db.Name()),
							Verbose:  utils.NewType(true),
						})
						require.NoError(t, err)
						require.NotNil(t, resp)
					})
					t.Run("Applier Start with out query params", func(t *testing.T) {
						resp, err := client.ApplierStart(ctx, db.Name(), nil, nil)
						require.NoError(t, err)
						require.NotNil(t, resp)
					})
					// Insert docs
					t.Run("Inserting 5 documents", func(t *testing.T) {
						for i := 0; i < 5; i++ {
							resp, err := coll.CreateDocument(ctx, map[string]string{"foo": fmt.Sprintf("bar-%d", i)})
							require.NoError(t, err)
							require.NotNil(t, resp)
						}
					})
					// Force sync and check WAL range again
					time.Sleep(500 * time.Millisecond) // Increase sleep time
					t.Run("Get WAL Tail with query params", func(t *testing.T) {
						tailResp, err := client.GetWALTail(ctx, db.Name(),
							&arangodb.WALTailOptions{
								Global:      utils.NewType(false),
								From:        utils.NewType(fromTick),
								ChunkSize:   utils.NewType(1024 * 1024),
								LastScanned: utils.NewType(0),
							})
						require.NoError(t, err)
						require.GreaterOrEqual(t, len(tailResp), 0)
					})

					t.Run("Applier Stop with out query params", func(t *testing.T) {
						resp, err := client.ApplierStop(ctx, db.Name(), nil)
						require.NoError(t, err)
						require.NotNil(t, resp)
					})
				})
			})
		})
	})
}

func Test_RebuildShardRevisionTree(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				// Version checking
				if os.Getenv("TEST_CONNECTION") == "vst" {
					skipBelowVersion(client, ctx, "3.8", t)
				}

				// Role check
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole is %s\n", serverRole)

				if serverRole != arangodb.ServerRoleDBServer {
					t.Skipf("Not supported on role: %s", serverRole)
				}

				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					// Insert documents
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

					var shardId arangodb.ShardID
					shards, err := col.Shards(ctx, true)
					require.NoError(t, err)
					require.NotNil(t, shards)

					for existingShardId := range shards.Shards {
						shardId = existingShardId
						break
					}
					// Call Rebuild Shard Revision Tree
					err = client.RebuildShardRevisionTree(ctx, db.Name(), shardId)
					require.NoError(t, err)
				})
			})
		})
	})
}
func Test_ListDocumentRevisionsInRange(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				// Version checking
				if os.Getenv("TEST_CONNECTION") == "vst" {
					skipBelowVersion(client, ctx, "3.8", t)
				}
				// Role check
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole: %s", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}

				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					var revs []string
					// Insert documents
					for i := 1; i <= 10; i++ {
						doc := map[string]interface{}{
							"_key": fmt.Sprintf("doc%d", i),
							"name": fmt.Sprintf("User %d", i),
						}
						resp, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
						require.NotNil(t, resp)
						revs = append(revs, resp.Rev)
					}
					require.NotEmpty(t, revs)
					time.Sleep(500 * time.Millisecond)

					// Create a replication batch
					state := utils.NewType(true)
					batch, err := client.CreateNewBatch(ctx, db.Name(), nil, state, arangodb.CreateNewBatchOptions{Ttl: 300})
					require.NoError(t, err)
					require.NotNil(t, batch)
					require.NotEmpty(t, batch.ID)

					// Prepare pairs for ListDocumentRevisionsInRange
					var opts [][2]string
					for i := 0; i < len(revs)-1; i++ {
						opts = append(opts, [2]string{revs[i], revs[i+1]})
					}

					// Call ListDocumentRevisionsInRange
					revIds, err := client.ListDocumentRevisionsInRange(ctx, db.Name(), arangodb.RevisionQueryParams{
						BatchId:    batch.ID,
						Collection: col.Name(),
					}, opts)
					require.NoError(t, err)
					require.NotNil(t, revIds)
				})
			})
		})
	})
}

func Test_FetchRevisionDocuments(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				// Version checking
				if os.Getenv("TEST_CONNECTION") == "vst" {
					skipBelowVersion(client, ctx, "3.8", t)
				}

				// Role check
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole: %s", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator {
					t.Skipf("Not supported on Coordinators (role: %s)", serverRole)
				}

				WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
					var revs []string
					// Insert documents
					docs := []map[string]interface{}{}
					docs = append(docs, map[string]interface{}{"_key": "doc1", "name": "Alice"})
					docs = append(docs, map[string]interface{}{"_key": "doc2", "subjects": []string{"English", "Maths"}})
					docs = append(docs, map[string]interface{}{"_key": "doc3", "age": 30, "active": true})
					docs = append(docs, map[string]interface{}{"_key": "doc4", "profile": map[string]interface{}{"city": "Berlin", "country": "Germany"}})

					for _, doc := range docs {
						resp, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)
						require.NotNil(t, resp)
						revs = append(revs, resp.Rev)
					}
					require.NotEmpty(t, revs)
					time.Sleep(500 * time.Millisecond)

					// Create a replication batch
					state := utils.NewType(true)
					batch, err := client.CreateNewBatch(ctx, db.Name(), nil, state, arangodb.CreateNewBatchOptions{Ttl: 300})
					require.NoError(t, err)
					require.NotNil(t, batch)
					require.NotEmpty(t, batch.ID)

					// Call FetchRevisionDocuments
					revDocs, err := client.FetchRevisionDocuments(ctx, db.Name(), arangodb.RevisionQueryParams{
						BatchId:    batch.ID,
						Collection: col.Name(),
					}, revs)
					require.NoError(t, err)
					require.NotNil(t, revDocs)
				})
			})
		})
	})
}

func Test_StartReplicationSync(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				// Version checking
				if os.Getenv("TEST_CONNECTION") == "vst" {
					skipBelowVersion(client, ctx, "3.8", t)
				}

				// Role check
				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				t.Logf("ServerRole: %s", serverRole)

				if serverRole == arangodb.ServerRoleCoordinator || serverRole == arangodb.ServerRoleSingle {
					t.Skipf("Replication sync not supported on role: %s", serverRole)
				}

				opts := arangodb.ReplicationSyncOptions{
					Endpoint: "http+tcp://127.0.0.1:8529",
					Username: "root",
				}

				result, err := client.StartReplicationSync(ctx, db.Name(), opts)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if len(result.Collections) == 0 {
					t.Errorf("expected collections in result")
				}
				if result.LastLogTick == "" {
					t.Errorf("expected lastLogTick in result")
				}
			})
		})
	})
}
