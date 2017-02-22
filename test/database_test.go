package test

import "testing"

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
