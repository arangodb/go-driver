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
// Author Tomasz Mielech
//

package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

		ctx2 := ctx
		if version.Version.CompareTo("3.2") >= 0 {
			var serverID int64 = 1337 // Random test value
			// RocksDB requires batchID
			batch, err := rep.CreateBatch(ctx, db, serverID, time.Second*60)
			if err != nil {
				t.Fatalf("CreateBatch failed: %s", describe(err))
			}
			ctx2 = driver.WithBatchID(ctx, batch.BatchID())
			defer batch.Delete(ctx)
		}

		inv, err := rep.DatabaseInventory(ctx2, db)
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

func TestReplicationBatch(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)

	if _, err := c.Cluster(ctx); err == nil {
		// Cluster, not supported for this test
		t.Skip("Skipping in cluster")
	} else if !driver.IsPreconditionFailed(err) {
		t.Errorf("Failed to query cluster: %s", describe(err))
	}

	rep := c.Replication()
	db, err := c.Database(ctx, "_system")
	require.NoError(t, err, "failed to open _system database")

	var serverID int64 = 1338 // Random test value
	batch, err := rep.CreateBatch(ctx, db, serverID, time.Second*60)
	require.NoError(t, err, "can not create a batch")

	ctxCancel, cancel := context.WithCancel(ctx)
	errExtend := make(chan error)
	go func(channel chan<- error) {
		for {
			select {
			case <-ctxCancel.Done():
				// The batch should be closed.
				channel <- batch.Extend(ctx, time.Second*60)
				return
			default:
				// Extend the batch immediately.
				if e := batch.Extend(ctx, time.Second*60); e == driver.ErrBatchClosed {
					if ctxError := ctx.Err(); ctxError != nil && ctxError != context.Canceled {
						channel <- errors.New("the batch extension is closed before it is expected")
						return
					}
				}
			}
		}
	}(errExtend)

	// Extend the created batch for 1 second.
	time.Sleep(time.Second * 1)

	// Delete the batch and interrupt the Go routine for batch extension.
	err = batch.Delete(ctx)
	cancel()
	assert.NoError(t, err, "can not delete a batch")
	require.EqualError(t, <-errExtend, driver.ErrBatchClosed.Error(), "batch should be already closed")
}
