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
	"github.com/stretchr/testify/require"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// ensureDatabase is a helper to check if a database exists and create it if needed.
// It will fail the test when an error occurs.
func ensureDatabase(ctx context.Context, c driver.Client, name string, options *driver.CreateDatabaseOptions, t testEnv) driver.Database {
	db, err := c.Database(ctx, name)
	if driver.IsNotFound(err) {
		db, err = c.CreateDatabase(ctx, name, options)
		if err != nil {
			if driver.IsConflict(err) {
				t.Fatalf("Failed to create database (conflict) '%s': %s %#v", name, describe(err), err)
			} else {
				t.Fatalf("Failed to create database '%s': %s %#v", name, describe(err), err)
			}
		}
	} else if err != nil {
		t.Fatalf("Failed to open database '%s': %s", name, describe(err))
	}
	return db
}

func skipIfEngineTypeRocksDB(t *testing.T, db driver.Database) {
	skipIfEngineType(t, db, driver.EngineTypeRocksDB)
}

func skipIfEngineTypeMMFiles(t *testing.T, db driver.Database) {
	skipIfEngineType(t, db, driver.EngineTypeMMFiles)
}

func skipIfEngineType(t *testing.T, db driver.Database, engineType driver.EngineType) {
	info, err := db.EngineInfo(nil)
	require.NoError(t, err)

	if info.Type == engineType {
		t.Skipf("test not supported on engine type %s", engineType)
	}
}

// TestCreateDatabase creates a database and then checks that it exists.
func TestCreateDatabase(t *testing.T) {
	c := createClientFromEnv(t, true)
	name := "create_test1"
	if _, err := c.CreateDatabase(nil, name, nil); err != nil {
		t.Fatalf("Failed to create database '%s': %s", name, describe(err))
	}
	// Database must exist now
	if found, err := c.DatabaseExists(nil, name); err != nil {
		t.Errorf("DatabaseExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("DatabaseExists('%s') return false, expected true", name)
	}
}

// TestRemoveDatabase creates a database and then removes it.
func TestRemoveDatabase(t *testing.T) {
	c := createClientFromEnv(t, true)
	name := "remove_test1"
	d, err := c.CreateDatabase(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create database '%s': %s", name, describe(err))
	}
	// Database must exist now
	if found, err := c.DatabaseExists(nil, name); err != nil {
		t.Errorf("DatabaseExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("DatabaseExists('%s') return false, expected true", name)
	}

	// Remove database
	if err := d.Remove(context.Background()); err != nil {
		t.Fatalf("Failed to remove database: %s", describe(err))
	}

	// Database must not exist now
	if found, err := c.DatabaseExists(nil, name); err != nil {
		t.Errorf("DatabaseExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("DatabaseExists('%s') return true, expected false", name)
	}
}

// TestDatabaseInfo tests Database.Info.
func TestDatabaseInfo(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)

	// Test system DB
	db := ensureDatabase(ctx, c, "_system", nil, t)
	info, err := db.Info(ctx)
	if err != nil {
		t.Fatalf("Failed to get _system database info: %s", describe(err))
	}
	if info.Name != "_system" {
		t.Errorf("Invalid Name. Got '%s', expected '_system'", info.Name)
	}
	if !info.IsSystem {
		t.Error("Invalid IsSystem. Got false, expected true")
	}
	if info.ID == "" {
		t.Error("Empty ID")
	}

	name := "info_test"
	d, err := c.CreateDatabase(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to create database '%s': %s", name, describe(err))
	}
	info, err = d.Info(ctx)
	if err != nil {
		t.Fatalf("Failed to get %s database info: %s", name, describe(err))
	}
	if info.Name != name {
		t.Errorf("Invalid Name. Got '%s', expected '%s'", info.Name, name)
	}
	if info.IsSystem {
		t.Error("Invalid IsSystem. Got true, expected false")
	}
	if info.ID == "" {
		t.Error("Empty ID")
	}

	// Cleanup: Remove database
	if err := d.Remove(context.Background()); err != nil {
		t.Fatalf("Failed to remove database: %s", describe(err))
	}
}
