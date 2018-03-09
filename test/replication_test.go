//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

	driver "github.com/arangodb/go-driver"
)

// TestReplicationDatabaseInventory tests the Replication.DatabaseInventory method.
func TestReplicationDatabaseInventory(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if _, err := c.Cluster(ctx); err == nil {
		// Cluster, not supported for this test
		t.Skip("Skipping in cluster")
	} else if !driver.IsPreconditionFailed(err) {
		t.Errorf("Failed to query cluster: %s", describe(err))
	} else {
		// Single server (what we need)
		rep := c.Replication()
		db, err := c.Database(ctx, "_system")
		if err != nil {
			t.Fatalf("Failed to open _system database: %s", describe(err))
		}

		version, err := c.Version(nil)
		if err != nil {
			t.Fatalf("Version failed: %s", describe(err))
		}

		ctx_inv := ctx 
		if version.Version.CompareTo("3.2") >= 0 {
			// RocksDB requires batchID
			batch, err := rep.CreateBatch(ctx, 1337, db)
			if err != nil {
				t.Fatalf("CreateBatch failed: %s", describe(err))
			}
			ctx_inv = driver.WithBatchID(ctx, batch.ID)
			defer rep.DeleteBatch(ctx, db, batch.ID)
		}

		inv, err := rep.DatabaseInventory(ctx_inv, db)
		if err != nil {
			t.Fatalf("DatabaseInventory failed: %s", describe(err))
		}
		if len(inv.Collections) == 0 {
			t.Error("Expected multiple collections, got 0")
		}
		foundSystemCol := false
		for _, col := range inv.Collections {
			if col.Parameters.Name == "" {
				t.Error("Expected non-empty name")
			}
			if col.Parameters.IsSystem {
				foundSystemCol = true
			}
		}
		if !foundSystemCol {
			t.Error("Expected multiple system collections, found none")
		}
	}
}
