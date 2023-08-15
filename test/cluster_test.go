//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
)

// TestClusterHealth tests the Cluster.Health method.
func TestClusterHealth(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else if err != nil {
		t.Fatalf("Health failed: %s", describe(err))
	} else {
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		if h.ID == "" {
			t.Error("Expected cluster ID to be non-empty")
		}
		agents := 0
		dbservers := 0
		coordinators := 0
		for _, sh := range h.Health {

			if v, err := c.Version(nil); err == nil {
				if v.Version.CompareTo(sh.Version) != 0 {
					t.Logf("Server version differs from `_api/version`, got `%s` and `%s`", v.Version, sh.Version)
				}
			} else {
				t.Errorf("Version failed: %s", describe(err))
			}

			switch sh.Role {
			case driver.ServerRoleAgent:
				agents++
			case driver.ServerRoleDBServer:
				dbservers++
			case driver.ServerRoleCoordinator:
				coordinators++
			}
		}
		if agents == 0 {
			t.Error("Expected at least 1 agent")
		}
		if dbservers == 0 {
			t.Error("Expected at least 1 dbserver")
		}
		if coordinators == 0 {
			t.Error("Expected at least 1 coordinator")
		}
	}
}

// TestClusterDatabaseInventory tests the Cluster.DatabaseInventory method.
func TestClusterDatabaseInventory(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		for _, col := range inv.Collections {
			if len(col.Parameters.Shards) == 0 {
				t.Errorf("Expected 1 or more shards in collection %s, got 0", col.Parameters.Name)
			}
			for shardID, dbServers := range col.Parameters.Shards {
				for _, serverID := range dbServers {
					if _, found := h.Health[serverID]; !found {
						t.Errorf("Unexpected dbserver ID for shard '%s': %s", shardID, serverID)
					}
				}
			}
		}
	}
}

// TestClusterDatabaseInventorySatellite tests the Cluster.DatabaseInventory method with satellite collections
func TestClusterDatabaseInventorySatellite(t *testing.T) {
	skipNoEnterprise(t)
	name := "satellite_collection_dbinv"
	ctx := context.Background()
	c := createClient(t, nil)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		col := ensureCollection(ctx, db, name, &driver.CreateCollectionOptions{
			ReplicationFactor: driver.ReplicationFactorSatellite,
		}, t)
		defer clean(t, ctx, col)
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		foundSatellite := false
		for _, col := range inv.Collections {
			if len(col.Parameters.Shards) == 0 {
				t.Errorf("Expected 1 or more shards in collection %s, got 0", col.Parameters.Name)
			}
			if col.Parameters.IsSatellite() {
				foundSatellite = true
			}
			for shardID, dbServers := range col.Parameters.Shards {
				for _, serverID := range dbServers {
					if _, found := h.Health[serverID]; !found {
						t.Errorf("Unexpected dbserver ID for shard '%s': %s", shardID, serverID)
					}
				}
			}
		}

		if !foundSatellite {
			t.Errorf("No satellite collection.")
		}
	}
}

// TestClusterDatabaseInventorySmartJoin tests the Cluster.DatabaseInventory method with smart joins
func TestClusterDatabaseInventorySmartJoin(t *testing.T) {
	skipNoEnterprise(t)
	name := "smart_join_collection_dbinv"
	nameParent := "smart_join_collection_dbinv_parent"
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4.5", t)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		colParent := ensureCollection(ctx, db, nameParent, &driver.CreateCollectionOptions{
			ShardKeys:      []string{"_key"},
			NumberOfShards: 2,
		}, t)
		defer clean(t, ctx, colParent)

		col := ensureCollection(ctx, db, name, &driver.CreateCollectionOptions{
			DistributeShardsLike: nameParent,
			ShardKeys:            []string{"_key:"},
			SmartJoinAttribute:   "smart",
			NumberOfShards:       2,
		}, t)
		defer clean(t, ctx, col)
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		foundSmartJoin := false
		for _, col := range inv.Collections {
			if col.Parameters.Name == name && col.Parameters.SmartJoinAttribute == "smart" {
				foundSmartJoin = true
			}
		}

		if !foundSmartJoin {
			t.Errorf("No smart join attribute.")
		}
	}
}

// TestClusterDatabaseInventoryShardingStrategy tests the Cluster.DatabaseInventory method with sharding strategy
func TestClusterDatabaseInventoryShardingStrategy(t *testing.T) {
	skipNoEnterprise(t)
	name := "shard_strat_collection_dbinv"
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		col := ensureCollection(ctx, db, name, &driver.CreateCollectionOptions{
			ShardingStrategy: driver.ShardingStrategyCommunityCompat,
		}, t)
		defer clean(t, ctx, col)
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		for _, col := range inv.Collections {
			if col.Parameters.Name == name {
				if col.Parameters.ShardingStrategy != driver.ShardingStrategyCommunityCompat {
					t.Errorf("Invalid sharding strategy, expected `%s`, found `%s`.", driver.ShardingStrategyCommunityCompat, col.Parameters.ShardingStrategy)
				}
				break
			}
		}
	}
}

// TestClusterMoveShard tests the Cluster.MoveShard method.
func TestClusterMoveShard(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		col, err := db.CreateCollection(ctx, "test_move_shard", &driver.CreateCollectionOptions{
			NumberOfShards: 12,
		})
		if err != nil {
			t.Fatalf("CreateCollection failed: %s", describe(err))
		}
		defer clean(t, ctx, col)
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		var targetServerID driver.ServerID
		for id, s := range h.Health {
			if s.Role == driver.ServerRoleDBServer {
				targetServerID = id
				break
			}
		}
		if len(targetServerID) == 0 {
			t.Fatalf("Failed to find any dbserver")
		}
		movedShards := 0
		for _, colInv := range inv.Collections {
			if colInv.Parameters.Name == col.Name() {
				for shardID, dbServers := range colInv.Parameters.Shards {
					if dbServers[0] != targetServerID {
						movedShards++
						var rawResponse []byte
						var jobID string
						jobCtx := driver.WithJobIDResponse(driver.WithRawResponse(ctx, &rawResponse), &jobID)
						if err := cl.MoveShard(jobCtx, col, shardID, dbServers[0], targetServerID); err != nil {
							t.Errorf("MoveShard for shard %s in collection %s failed: %s (raw response '%s' %x)", shardID, col.Name(), describe(err), string(rawResponse), rawResponse)
						}
						defer waitForJob(t, jobID, c)()
					}
				}
			}
		}
		if movedShards == 0 {
			t.Fatal("Expected to have moved at least 1 shard, all seem to be on target server already")
		}
		// Wait until all shards are on the targetServerID
		start := time.Now()
		maxTestTime := time.Minute
		lastShardsNotOnTargetServerID := movedShards
		for {
			shardsNotOnTargetServerID := 0
			inv, err := cl.DatabaseInventory(ctx, db)
			if err != nil {
				t.Errorf("DatabaseInventory failed: %s", describe(err))
			} else {
				for _, colInv := range inv.Collections {
					if colInv.Parameters.Name == col.Name() {
						for shardID, dbServers := range colInv.Parameters.Shards {
							if dbServers[0] != targetServerID {
								shardsNotOnTargetServerID++
								t.Logf("Shard %s in on %s, wanted %s", shardID, dbServers[0], targetServerID)
							}
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
	}
}

func TestClusterResignLeadership(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5.1", t)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		col, err := db.CreateCollection(ctx, "test_resign_leadership", &driver.CreateCollectionOptions{
			NumberOfShards:    12,
			ReplicationFactor: 2,
		})
		if err != nil {
			t.Fatalf("CreateCollection failed: %s", describe(err))
		}
		defer clean(t, ctx, col)
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		var targetServerID driver.ServerID
	collectionLoop:
		for _, colInv := range inv.Collections {
			if colInv.Parameters.Name == col.Name() {
				for _, dbServers := range colInv.Parameters.Shards {
					targetServerID = dbServers[0]

					var jobID string
					jobCtx := driver.WithJobIDResponse(context.Background(), &jobID)

					if err := cl.ResignServer(jobCtx, string(targetServerID)); err != nil {
						t.Errorf("ResignLeadership for %s failed: %s", targetServerID, describe(err))
					}
					defer waitForJob(t, jobID, c)()

					break collectionLoop
				}
			}
		}

		// Wait until targetServerID is no longer leader
		start := time.Now()
		maxTestTime := time.Minute
		lastLeaderForShardsNum := 0
		for {
			leaderForShardsNum := 0
			inv, err := cl.DatabaseInventory(ctx, db)
			if err != nil {
				t.Errorf("DatabaseInventory failed: %s", describe(err))
			} else {
				for _, colInv := range inv.Collections {
					if colInv.Parameters.Name == col.Name() {
						for shardID, dbServers := range colInv.Parameters.Shards {
							if dbServers[0] == targetServerID {
								leaderForShardsNum++
								t.Logf("%s is still leader for %s", targetServerID, shardID)
							}
						}
					}
				}
			}
			if leaderForShardsNum == 0 {
				// We're done
				break
			}
			if leaderForShardsNum != lastLeaderForShardsNum {
				// Something changed, we give a bit more time
				maxTestTime = maxTestTime + time.Second*15
				lastLeaderForShardsNum = leaderForShardsNum
			}
			if time.Since(start) > maxTestTime {
				t.Errorf("%s did not resign within %s", targetServerID, maxTestTime)
				break
			}
			t.Log("Waiting a bit")
			time.Sleep(time.Second * 5)
		}
	}
}

// TestClusterMoveShardWithViews tests the Cluster.MoveShard method with collection
// that are being used in views.
func TestClusterMoveShardWithViews(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}
		col, err := db.CreateCollection(ctx, "test_move_shard_with_view", &driver.CreateCollectionOptions{
			NumberOfShards: 12,
		})
		if err != nil {
			t.Fatalf("CreateCollection failed: %s", describe(err))
		}
		defer clean(t, ctx, col)
		opts := &driver.ArangoSearchViewProperties{
			Links: driver.ArangoSearchLinks{
				"test_move_shard_with_view": driver.ArangoSearchElementProperties{},
			},
		}
		viewName := "test_move_shard_view"
		view, err := db.CreateArangoSearchView(ctx, viewName, opts)
		if err != nil {
			t.Fatalf("Failed to create view '%s': %s", viewName, describe(err))
		}
		defer clean(t, ctx, view)
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		inv, err := cl.DatabaseInventory(ctx, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		var targetServerID driver.ServerID
		for id, s := range h.Health {
			if s.Role == driver.ServerRoleDBServer {
				targetServerID = id
				break
			}
		}
		if len(targetServerID) == 0 {
			t.Fatalf("Failed to find any dbserver")
		}
		movedShards := 0
		for _, colInv := range inv.Collections {
			if colInv.Parameters.Name == col.Name() {
				for shardID, dbServers := range colInv.Parameters.Shards {
					if dbServers[0] != targetServerID {
						movedShards++
						var rawResponse []byte
						var jobID string
						jobCtx := driver.WithJobIDResponse(driver.WithRawResponse(ctx, &rawResponse), &jobID)
						if err := cl.MoveShard(jobCtx, col, shardID, dbServers[0], targetServerID); err != nil {
							t.Errorf("MoveShard for shard %s in collection %s failed: %s (raw response '%s' %x)", shardID, col.Name(), describe(err), string(rawResponse), rawResponse)
						}
						defer waitForJob(t, jobID, c)()
					}
				}
			}
		}
		if movedShards == 0 {
			t.Fatal("Expected to have moved at least 1 shard, all seem to be on target server already")
		}
		// Wait until all shards are on the targetServerID
		start := time.Now()
		maxTestTime := time.Minute
		lastShardsNotOnTargetServerID := movedShards
		for {
			shardsNotOnTargetServerID := 0
			inv, err := cl.DatabaseInventory(ctx, db)
			if err != nil {
				t.Errorf("DatabaseInventory failed: %s", describe(err))
			} else {
				for _, colInv := range inv.Collections {
					if colInv.Parameters.Name == col.Name() {
						for shardID, dbServers := range colInv.Parameters.Shards {
							if dbServers[0] != targetServerID {
								shardsNotOnTargetServerID++
								t.Logf("Shard %s in on %s, wanted %s", shardID, dbServers[0], targetServerID)
							}
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
	}
}
