//
// DISCLAIMER
//
// Copyright 2017-2021 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"strings"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"github.com/arangodb/go-driver"
)

// databaseName is helper to create database name in non-colliding way
func databaseName(parts ...string) string {
	return fmt.Sprintf("%s_%s", strings.Join(parts, "_"), uniuri.NewLen(8))
}

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

func TestDatabaseNameUnicode(t *testing.T) {
	c := createClientFromEnv(t, true)
	databaseExtendedNamesRequired(t, c)

	dbName := "\u006E\u0303\u00f1"
	normalized := norm.NFC.String(dbName)
	ctx := context.Background()
	_, err := c.CreateDatabase(ctx, dbName, nil)
	require.EqualError(t, err, "database name is not properly UTF-8 NFC-normalized")

	_, err = c.CreateDatabase(ctx, normalized, nil)
	require.NoError(t, err)

	// The database should not be found by the not normalized name.
	_, err = c.Database(ctx, dbName)
	require.NotNil(t, err)

	// The database should be found by the normalized name.
	exist, err := c.DatabaseExists(ctx, normalized)
	require.NoError(t, err)
	require.True(t, exist)

	var found bool
	databases, err := c.Databases(ctx)
	require.NoError(t, err)
	for _, database := range databases {
		if database.Name() == normalized {
			found = true
			break
		}
	}
	require.Truef(t, found, "the database %s should have been found", normalized)

	// The database should return handler to the database by the normalized name.
	db, err := c.Database(ctx, normalized)
	require.NoError(t, err)
	require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
}

// TestCreateDatabaseReplication2 creates a database with replication version two.
func TestCreateDatabaseReplication2(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.12.0"))

	name := "create_test1"
	opts := driver.CreateDatabaseOptions{Options: driver.CreateDatabaseDefaultOptions{
		ReplicationVersion: driver.DatabaseReplicationVersionTwo,
	}}
	if _, err := c.CreateDatabase(nil, name, &opts); err != nil {
		t.Fatalf("Failed to create database '%s': %s", name, describe(err))
	}
	// Database must exist now
	if found, err := c.DatabaseExists(nil, name); err != nil {
		t.Errorf("DatabaseExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("DatabaseExists('%s') return false, expected true", name)
	}

	// Read database properties
	db, err := c.Database(nil, name)
	if err != nil {
		t.Fatal("Failed to get database ")
	}
	info, err := db.Info(nil)
	if err != nil {
		t.Fatal("Failed to get database name")
	}

	if info.ReplicationVersion != driver.DatabaseReplicationVersionTwo {
		t.Errorf("Wrong replication version, expected %s, found %s", driver.DatabaseReplicationVersionTwo, info.ReplicationVersion)
	}
}

// databaseExtendedNamesRequired skips test if the version is < 3.9.0 or the ArangoDB has not been launched
// with the option --database.extended-names-databases=true.
func databaseExtendedNamesRequired(t *testing.T, c driver.Client) {
	ctx := context.Background()
	version, err := c.Version(ctx)
	require.NoError(t, err)

	if version.Version.CompareTo("3.9.0") < 0 {
		t.Skipf("Version of the ArangoDB should be at least 3.9.0")
	}

	// If the database can be created with the below name then it means that it excepts unicode names.
	dbName := "\u006E\u0303\u00f1"
	normalized := norm.NFC.String(dbName)
	db, err := c.CreateDatabase(ctx, normalized, nil)
	if err == nil {
		require.NoErrorf(t, db.Remove(ctx), "failed to remove testing database")
		return
	}

	if driver.IsArangoErrorWithErrorNum(err, driver.ErrArangoDatabaseNameInvalid, driver.ErrArangoIllegalName) {
		t.Skipf("ArangoDB is not launched with the option --database.extended-names-databases=true")
	}

	// Some other error which has not been expected.
	require.NoError(t, err)
}
