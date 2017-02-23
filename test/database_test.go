package test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// ensureDatabase is a helper to check if a database exists and create it if needed.
// It will fail the test when an error occurs.
func ensureDatabase(ctx context.Context, c driver.Client, name string, options *driver.CreateDatabaseOptions, t *testing.T) driver.Database {
	db, err := c.Database(ctx, name)
	if driver.IsNotFound(err) {
		db, err = c.CreateDatabase(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create database '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open database '%s': %s", name, describe(err))
	}
	return db
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
