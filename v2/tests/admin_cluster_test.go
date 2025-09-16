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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
)

func Test_ClusterHealth(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			requireClusterMode(t)
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
		requireClusterMode(t)
		t.Run("DatabaseInventory simple checks", func(t *testing.T) {
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
					require.Greater(t, len(col.Parameters.Shards), 0,
						"Expected 1 or more shards in collection %s, got 0", col.Parameters.Name)

					for shardID, dbServers := range col.Parameters.Shards {
						for _, serverID := range dbServers {
							require.Contains(t, health.Health, serverID,
								"Unexpected dbServer ID for shard '%s': %s", shardID, serverID)
						}
					}
				}
			})
		})

		t.Run("tests the DatabaseInventory with SatelliteCollections", func(t *testing.T) {
			skipNoEnterprise(client, context.Background(), t)

			WithDatabase(t, client, nil, func(db arangodb.Database) {
				optionsSatellite := arangodb.CreateCollectionPropertiesV2{
					ReplicationFactor: utils.NewType(arangodb.ReplicationFactorSatellite),
				}
				WithCollectionV2(t, db, &optionsSatellite, func(col arangodb.Collection) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
						health, err := client.Health(ctx)
						require.NoError(t, err, "Health failed")

						inv, err := client.DatabaseInventory(ctx, db.Name())
						require.NoError(t, err, "DatabaseInventory failed")
						require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

						foundSatellite := false
						for _, cl := range inv.Collections {
							require.Greater(t, len(cl.Parameters.Shards), 0,
								"Expected 1 or more shards in collection %s, got 0", cl.Parameters.Name)

							if cl.Parameters.IsSatellite() {
								foundSatellite = true
							}
							for shardID, dbServers := range cl.Parameters.Shards {
								for _, serverID := range dbServers {
									require.Contains(t, health.Health, serverID,
										"Unexpected dbServer ID for shard '%s': %s", shardID, serverID)
								}
							}
						}
						require.True(t, foundSatellite, "DatabaseInventory did not return any SatelliteCollections")
					})
				})
			})
		})

		t.Run("tests the DatabaseInventory with with SmartJoins", func(t *testing.T) {
			skipNoEnterprise(client, context.Background(), t)

			WithDatabase(t, client, nil, func(db arangodb.Database) {
				optionsSatellite := arangodb.CreateCollectionPropertiesV2{
					ShardKeys:      &[]string{"_key"},
					NumberOfShards: utils.NewType(2),
				}
				WithCollectionV2(t, db, &optionsSatellite, func(colParent arangodb.Collection) {
					optionsSmartJoins := arangodb.CreateCollectionPropertiesV2{
						DistributeShardsLike: utils.NewType(colParent.Name()),
						ShardKeys:            &[]string{"_key:"},
						SmartJoinAttribute:   utils.NewType("smart"),
						NumberOfShards:       utils.NewType(2),
					}
					WithCollectionV2(t, db, &optionsSmartJoins, func(col arangodb.Collection) {
						withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
							inv, err := client.DatabaseInventory(ctx, db.Name())
							require.NoError(t, err, "DatabaseInventory failed")
							require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

							foundSmartJoin := false
							for _, cl := range inv.Collections {
								if cl.Parameters.Name == col.Name() && cl.Parameters.SmartJoinAttribute == "smart" {
									foundSmartJoin = true
								}
							}
							require.True(t, foundSmartJoin, "DatabaseInventory did not return any SmartJoin collections")
						})
					})
				})
			})
		})

		t.Run("tests the DatabaseInventory with sharding strategy", func(t *testing.T) {
			skipNoEnterprise(client, context.Background(), t)

			WithDatabase(t, client, nil, func(db arangodb.Database) {
				optionsSatellite := arangodb.CreateCollectionPropertiesV2{
					ShardingStrategy: utils.NewType(arangodb.ShardingStrategyCommunityCompat),
				}
				WithCollectionV2(t, db, &optionsSatellite, func(col arangodb.Collection) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
						inv, err := client.DatabaseInventory(ctx, db.Name())
						require.NoError(t, err, "DatabaseInventory failed")
						require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

						for _, cl := range inv.Collections {
							if cl.Parameters.Name == col.Name() {
								require.Equal(t, arangodb.ShardingStrategyCommunityCompat, cl.Parameters.ShardingStrategy)
							}
						}
					})
				})
			})
		})
	})
}

func Test_ClusterMoveShards(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			optionsShards := arangodb.CreateCollectionPropertiesV2{
				NumberOfShards: utils.NewType(12),
			}
			WithCollectionV2(t, db, &optionsShards, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					health, err := client.Health(ctx)
					require.NoError(t, err, "Health failed")

					inv, err := client.DatabaseInventory(ctx, db.Name())
					require.NoError(t, err, "DatabaseInventory failed")
					require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

					var targetServerID arangodb.ServerID
					for id, s := range health.Health {
						if s.Role == arangodb.ServerRoleDBServer {
							targetServerID = id
							break
						}
					}
					require.NotEmpty(t, targetServerID, "No dbServer found")

					movedShards := 0
					for _, colInv := range inv.Collections {
						if colInv.Parameters.Name == col.Name() {
							for shardID, dbServers := range colInv.Parameters.Shards {
								if dbServers[0] != targetServerID {
									movedShards++
									jobID, err := client.MoveShard(ctx, col, shardID, dbServers[0], targetServerID)
									require.NoError(t, err, "MoveShard for shard %s in collection %s failed", shardID, col.Name())
									require.NotEmpty(t, jobID, "MoveShard for shard %s in collection %s did not return a jobID", shardID, col.Name())
								}
							}
						}
					}
					require.Greater(t, movedShards, 0, "No shards moved")

					t.Run("Check if shards are moved", func(t *testing.T) {
						start := time.Now()
						maxTestTime := 2 * time.Minute
						lastShardsNotOnTargetServerID := movedShards

						for {
							shardsNotOnTargetServerID := 0

							inventory, err := client.DatabaseInventory(ctx, db.Name())
							require.NoError(t, err, "DatabaseInventory failed")

							for _, colInv := range inventory.Collections {
								if colInv.Parameters.Name == col.Name() {
									for shardID, dbServers := range colInv.Parameters.Shards {
										if dbServers[0] != targetServerID {
											shardsNotOnTargetServerID++
											t.Logf("Shard %s in on %s, wanted %s", shardID, dbServers[0], targetServerID)
										}
									}
								}
							}

							if shardsNotOnTargetServerID == 0 {
								// We're done
								break
							}

							if shardsNotOnTargetServerID != lastShardsNotOnTargetServerID {
								// Something changed, we give a bit more time
								maxTestTime = maxTestTime + time.Second*15
								lastShardsNotOnTargetServerID = shardsNotOnTargetServerID
							}

							if time.Since(start) > maxTestTime {
								t.Errorf("%d shards did not move within %s", shardsNotOnTargetServerID, maxTestTime)
								break
							}

							t.Log("Waiting a bit")
							time.Sleep(time.Second * 5)
						}
					})
				})
			})
		})
	})
}

func Test_ClusterResignLeadership(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			optionsShards := arangodb.CreateCollectionPropertiesV2{
				NumberOfShards:    utils.NewType(12),
				ReplicationFactor: utils.NewType(arangodb.ReplicationFactor(2)),
			}
			WithCollectionV2(t, db, &optionsShards, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					// First, ensure we have multiple DB servers available
					health, err := client.Health(ctx)
					require.NoError(t, err, "Health failed")

					var dbServerCount int
					for _, server := range health.Health {
						if server.Role == arangodb.ServerRoleDBServer {
							dbServerCount++
						}
					}
					require.GreaterOrEqual(t, dbServerCount, 2, "ResignLeadership test requires at least 2 DB servers, found %d", dbServerCount)

					inv, err := client.DatabaseInventory(ctx, db.Name())
					require.NoError(t, err, "DatabaseInventory failed")
					require.Greater(t, len(inv.Collections), 0, "DatabaseInventory did not return any collections")

					// Find a server that is a leader for some shards
					var targetServerID arangodb.ServerID
					leaderShardCount := 0
					for _, colInv := range inv.Collections {
						if colInv.Parameters.Name == col.Name() {
							for _, dbServers := range colInv.Parameters.Shards {
								if targetServerID == "" {
									targetServerID = dbServers[0] // Pick the first leader we find
								}
								if dbServers[0] == targetServerID {
									leaderShardCount++
								}
							}
						}
					}
					require.NotEmpty(t, targetServerID, "No dbServer found")
					require.Greater(t, leaderShardCount, 0, "Target server %s is not leader for any shards", targetServerID)

					t.Logf("Target server %s is leader for %d shards", targetServerID, leaderShardCount)

					// Now call ResignServer once for the selected server
					jobID, err := client.ResignServer(ctx, targetServerID)
					require.NoError(t, err, "ResignServer for server %s failed", targetServerID)
					require.NotEmpty(t, jobID, "ResignServer for server %s did not return a jobID", targetServerID)

					t.Run("Check if targetServerID is no longer leader", func(t *testing.T) {
						// Give the resign operation some time to start before checking
						t.Logf("Expecting %s to resign leadership from %d shards", targetServerID, leaderShardCount)
						t.Log("Giving resign operation time to start...")
						time.Sleep(time.Second * 10)

						start := time.Now()
						maxTestTime := 2 * time.Minute             // Increased from 1 minute to match MoveShard timeout
						lastLeaderForShardsNum := leaderShardCount // Start with the expected initial count

						for {
							leaderForShardsNum := 0
							inventory, err := client.DatabaseInventory(ctx, db.Name())
							require.NoError(t, err, "DatabaseInventory failed")

							for _, colInv := range inventory.Collections {
								if colInv.Parameters.Name == col.Name() {
									for shardID, dbServers := range colInv.Parameters.Shards {
										if dbServers[0] == targetServerID {
											leaderForShardsNum++
											t.Logf("%s is still leader for %s", targetServerID, shardID)
										}
									}
								}
							}

							if leaderForShardsNum == 0 {
								// We're done
								t.Logf("Successfully resigned leadership from %s (was leader for %d shards)", targetServerID, leaderShardCount)
								break
							}

							if leaderForShardsNum != lastLeaderForShardsNum && lastLeaderForShardsNum != -1 {
								// Something changed, we give a bit more time
								maxTestTime = maxTestTime + time.Second*15
								t.Logf("Leadership count changed from %d to %d, extending timeout to %v", lastLeaderForShardsNum, leaderForShardsNum, maxTestTime)
								lastLeaderForShardsNum = leaderForShardsNum
							} else if lastLeaderForShardsNum == -1 {
								// First check after initial delay
								lastLeaderForShardsNum = leaderForShardsNum
							}

							if time.Since(start) > maxTestTime {
								t.Errorf("%s did not resign from %d shards within %s (still leader for %d shards)", targetServerID, leaderShardCount, maxTestTime, leaderForShardsNum)
								break
							}

							t.Logf("Waiting for leadership resignation... (%d/%d shards still led by target)", leaderForShardsNum, leaderShardCount)
							time.Sleep(time.Second * 5)
						}
					})
				})
			})
		})
	})
}

func Test_ClusterStatistics(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			requireClusterMode(t)
			skipBelowVersion(client, ctx, "3.7", t)
			// Detect DB-Server ID
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole is %s\n", serverRole)

			var dbServerId string
			if serverRole == arangodb.ServerRoleCoordinator {
				clusterHealth, err := client.Health(ctx)
				require.NoError(t, err)

				// Pick first DBServer ID
				for id, db := range clusterHealth.Health {
					if db.Role == arangodb.ServerRoleDBServer {
						dbServerId = string(id)
						break
					}
				}
			} else {
				t.Skip("ClusterStatistics test requires coordinator access to get DB-Server IDs")
			}
			statistics, err := client.ClusterStatistics(ctx, dbServerId)
			require.NoError(t, err)
			require.NotNil(t, statistics)
		})
	})
}

func Test_ClusterEndpoints(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			requireClusterMode(t)
			endpoints, err := client.ClusterEndpoints(ctx)
			require.NoError(t, err)
			require.NotNil(t, endpoints)
			require.GreaterOrEqual(t, len(endpoints.Endpoints), 0)
		})
	})
}

func Test_DBServerMaintenance(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			requireClusterMode(t)
			skipBelowVersion(client, ctx, "3.10", t)

			// Detect DB-Server ID
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole is %s\n", serverRole)

			var dbServerId string
			if serverRole == arangodb.ServerRoleCoordinator {
				clusterHealth, err := client.Health(ctx)
				require.NoError(t, err)

				// Pick first DBServer ID
				for id, db := range clusterHealth.Health {
					if db.Role == arangodb.ServerRoleDBServer {
						dbServerId = string(id)
						break
					}
				}

				// Toggle cluster maintenance (cluster-wide, no need to check agents)
				err = client.SetClusterMaintenance(ctx, "on")
				require.NoError(t, err, "failed to enable cluster maintenance")

				// Give cluster time to apply state
				time.Sleep(1 * time.Second)

				err = client.SetClusterMaintenance(ctx, "off")
				require.NoError(t, err, "failed to disable cluster maintenance")
			} else {
				t.Skip("DBServerMaintenance test requires coordinator access to get DB-Server IDs")
			}

			// Call GetDBServerMaintenance
			clusterMaintenanceInfo, err := client.GetDBServerMaintenance(ctx, dbServerId)
			require.NoError(t, err)
			require.NotNil(t, clusterMaintenanceInfo)

			// Validate fields if in maintenance
			if clusterMaintenanceInfo.Mode != "" {
				require.Equal(t, "maintenance", clusterMaintenanceInfo.Mode)
				require.NotEmpty(t, clusterMaintenanceInfo.Until)
			}

			respJson, err := utils.ToJSONString(clusterMaintenanceInfo)
			require.NoError(t, err)
			t.Logf("Before Cluster Maintenance Response: %s\n", respJson)

			// Update DBServer Maintenance
			err = client.SetDBServerMaintenance(ctx, dbServerId, &arangodb.ClusterMaintenanceOpts{
				Mode:    "maintenance",
				Timeout: utils.NewType(30),
			})
			require.NoError(t, err)

			// Call GetDBServerMaintenance
			clusterMaintenanceInfo, err = client.GetDBServerMaintenance(ctx, dbServerId)
			require.NoError(t, err)
			require.NotNil(t, clusterMaintenanceInfo)

			// Validate fields if in maintenance
			if clusterMaintenanceInfo.Mode != "" {
				require.Equal(t, "maintenance", clusterMaintenanceInfo.Mode)
				require.NotEmpty(t, clusterMaintenanceInfo.Until)
			}

			respJson, err = utils.ToJSONString(clusterMaintenanceInfo)
			require.NoError(t, err)
			t.Logf("After ClusterMaintenanceResponse: %s\n", respJson)
		})
	})
}

func Test_GetClusterRebalance(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipBelowVersion(client, ctx, "3.10", t)
			// Ensure the test only runs in cluster mode
			requireClusterMode(t)

			// Call the API
			rebalanceShardInfo, err := client.GetClusterRebalance(ctx)
			require.NoError(t, err)
			require.NotNil(t, rebalanceShardInfo)

			// Validate leader stats
			require.NotNil(t, rebalanceShardInfo.Leader)
			// Validate shard stats
			require.NotNil(t, rebalanceShardInfo.Shards)

			// Validate pending and todo move shard counts
			require.NotNil(t, rebalanceShardInfo.PendingMoveShards)
			require.NotNil(t, rebalanceShardInfo.TodoMoveShards)
			require.GreaterOrEqual(t, *rebalanceShardInfo.PendingMoveShards, int64(0))
			require.GreaterOrEqual(t, *rebalanceShardInfo.TodoMoveShards, int64(0))
		})
	})
}

func Test_ComputeClusterRebalance(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipBelowVersion(client, ctx, "3.10", t)
			// Ensure the test only runs in cluster mode
			requireClusterMode(t)
			WithDatabase(t, client, nil, func(db arangodb.Database) {
				rf := arangodb.ReplicationFactor(2)

				coll, err := db.CreateCollectionV2(ctx, "rebalance_test_coll_1", &arangodb.CreateCollectionPropertiesV2{
					NumberOfShards:    utils.NewType(12), // big number
					ReplicationFactor: &rf,               // ensures leaders + followers
				})
				require.NoError(t, err)
				require.NotNil(t, coll)

				// Call the API
				requestBody := &arangodb.RebalanceRequestBody{
					Version:              utils.NewType(1),
					MaximumNumberOfMoves: utils.NewType(10),
				}
				rebalanceShardResp, err := client.ComputeClusterRebalance(ctx, requestBody)
				require.NoError(t, err)
				require.NotNil(t, rebalanceShardResp)

				require.NotNil(t, rebalanceShardResp.ImbalanceBefore)
				require.NotNil(t, rebalanceShardResp.ImbalanceBefore.Leader)
				require.NotNil(t, rebalanceShardResp.ImbalanceBefore.Shards)

				require.NotNil(t, rebalanceShardResp.ImbalanceAfter)
				require.NotNil(t, rebalanceShardResp.ImbalanceAfter.Leader)
				require.NotNil(t, rebalanceShardResp.ImbalanceAfter.Shards)

				require.NotNil(t, rebalanceShardResp.Moves)
				require.GreaterOrEqual(t, len(rebalanceShardResp.Moves), 0)

				if len(rebalanceShardResp.Moves) > 0 {
					err := client.ExecuteClusterRebalance(ctx, &arangodb.ExecuteRebalanceRequestBody{
						Moves:   rebalanceShardResp.Moves,
						Version: utils.NewType(1),
					})
					require.NoError(t, err)
				}

				// Call the GetClusterRebalance API to validate it works after ComputeClusterRebalance
				rebalanceShardInfo, err := client.GetClusterRebalance(ctx)
				require.NoError(t, err)
				require.NotNil(t, rebalanceShardInfo)

				// Validate leader stats
				require.NotNil(t, rebalanceShardInfo.Leader)
				// Validate shard stats
				require.NotNil(t, rebalanceShardInfo.Shards)

				// Validate pending and todo move shard counts
				require.NotNil(t, rebalanceShardInfo.PendingMoveShards)
				require.NotNil(t, rebalanceShardInfo.TodoMoveShards)
				require.GreaterOrEqual(t, *rebalanceShardInfo.PendingMoveShards, int64(0))
				require.GreaterOrEqual(t, *rebalanceShardInfo.TodoMoveShards, int64(0))
			})
		})
	})
}

func Test_ComputeAndExecuteClusterRebalance(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipBelowVersion(client, ctx, "3.10", t)
			// Ensure the test only runs in cluster mode
			requireClusterMode(t)
			WithDatabase(t, client, nil, func(db arangodb.Database) {
				rf := arangodb.ReplicationFactor(2)

				coll, err := db.CreateCollectionV2(ctx, "rebalance_test_coll_2", &arangodb.CreateCollectionPropertiesV2{
					NumberOfShards:    utils.NewType(12), // big number
					ReplicationFactor: &rf,               // ensures leaders + followers
				})
				require.NoError(t, err)
				require.NotNil(t, coll)

				// Call the API
				requestBody := &arangodb.RebalanceRequestBody{
					Version:              utils.NewType(1),
					MaximumNumberOfMoves: utils.NewType(10),
				}
				rebalanceShardResp, err := client.ComputeAndExecuteClusterRebalance(ctx, requestBody)
				require.NoError(t, err)
				require.NotNil(t, rebalanceShardResp)

				require.NotNil(t, rebalanceShardResp.ImbalanceBefore)
				require.NotNil(t, rebalanceShardResp.ImbalanceBefore.Leader)
				require.NotNil(t, rebalanceShardResp.ImbalanceBefore.Shards)

				require.NotNil(t, rebalanceShardResp.ImbalanceAfter)
				require.NotNil(t, rebalanceShardResp.ImbalanceAfter.Leader)
				require.NotNil(t, rebalanceShardResp.ImbalanceAfter.Shards)

				require.NotNil(t, rebalanceShardResp.Moves)
				require.GreaterOrEqual(t, len(rebalanceShardResp.Moves), 0)

				// Call the GetClusterRebalance API to validate it works after ComputeClusterRebalance
				rebalanceShardInfo, err := client.GetClusterRebalance(ctx)
				require.NoError(t, err)
				require.NotNil(t, rebalanceShardInfo)

				// Validate leader stats
				require.NotNil(t, rebalanceShardInfo.Leader)
				// Validate shard stats
				require.NotNil(t, rebalanceShardInfo.Shards)

				// Validate pending and todo move shard counts
				require.NotNil(t, rebalanceShardInfo.PendingMoveShards)
				require.NotNil(t, rebalanceShardInfo.TodoMoveShards)
				require.GreaterOrEqual(t, *rebalanceShardInfo.PendingMoveShards, int64(0))
				require.GreaterOrEqual(t, *rebalanceShardInfo.TodoMoveShards, int64(0))
			})
		})
	})
}
