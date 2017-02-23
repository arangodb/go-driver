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

// TestRemoveCollection creates a collection and then removes it.
func TestRemoveCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_remove_collection"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// Collection must exist now
	if found, err := db.CollectionExists(nil, name); err != nil {
		t.Errorf("CollectionExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("CollectionExists('%s') return false, expected true", name)
	}
	// Now remove it
	if err := col.Remove(nil); err != nil {
		t.Fatalf("Failed to remove collection '%s': %s", name, describe(err))
	}
	// Collection must not exist now
	if found, err := db.CollectionExists(nil, name); err != nil {
		t.Errorf("CollectionExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("CollectionExists('%s') return true, expected false", name)
	}
}

// TestCollectionName creates a collection and checks its name
func TestCollectionName(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_remove_collection"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	if col.Name() != name {
		t.Errorf("Collection.Name() is wrong, got '%s', expected '%s'", col.Name(), name)
	}
}
