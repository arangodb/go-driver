//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
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
	c := createClientFromEnv(t, true)
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
	c := createClientFromEnv(t, true)
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

// TestClusterMoveShard tests the Cluster.MoveShard method.
func TestClusterMoveShard(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
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
						if err := cl.MoveShard(driver.WithRawResponse(ctx, &rawResponse), col, shardID, dbServers[0], targetServerID); err != nil {
							t.Errorf("MoveShard for shard %s in collection %s failed: %s (raw response '%s' %x)", shardID, col.Name(), describe(err), string(rawResponse), rawResponse)
						}
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

// TestClusterMoveShardWithViews tests the Cluster.MoveShard method with collection
// that are being used in views.
func TestClusterMoveShardWithViews(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
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
		opts := &driver.ArangoSearchViewProperties{
			Links: driver.ArangoSearchLinks{
				"test_move_shard_with_view": driver.ArangoSearchElementProperties{},
			},
		}
		viewName := "test_move_shard_view"
		if _, err := db.CreateArangoSearchView(ctx, viewName, opts); err != nil {
			t.Fatalf("Failed to create view '%s': %s", viewName, describe(err))
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
						if err := cl.MoveShard(driver.WithRawResponse(ctx, &rawResponse), col, shardID, dbServers[0], targetServerID); err != nil {
							t.Errorf("MoveShard for shard %s in collection %s failed: %s (raw response '%s' %x)", shardID, col.Name(), describe(err), string(rawResponse), rawResponse)
						}
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
