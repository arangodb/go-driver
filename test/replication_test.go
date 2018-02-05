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
)

// TestReplicationDatabaseInventory tests the Replication.DatabaseInventory method.
func TestReplicationDatabaseInventory(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	rep := c.Replication()
	db, err := c.Database(ctx, "_system")
	if err != nil {
		t.Fatalf("Failed to open _system database: %s", describe(err))
	}
	inv, err := rep.DatabaseInventory(ctx, db)
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
