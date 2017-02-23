package test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// ensureCollection is a helper to check if a collection exists and create if if needed.
// It will fail the test when an error occurs.
func ensureCollection(ctx context.Context, db driver.Database, name string, options *driver.CreateCollectionOptions, t *testing.T) driver.Collection {
	c, err := db.Collection(ctx, name)
	if driver.IsNotFound(err) {
		c, err = db.CreateCollection(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open collection '%s': %s", name, describe(err))
	}
	return c
}

// TestCreateCollection creates a collection and then checks that it exists.
func TestCreateCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_create_collection"
	if _, err := db.CreateCollection(nil, name, nil); err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// Collection must exist now
	if found, err := db.CollectionExists(nil, name); err != nil {
		t.Errorf("CollectionExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("CollectionExists('%s') return false, expected true", name)
	}
}
