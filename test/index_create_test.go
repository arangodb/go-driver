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
	"fmt"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestCreateFullTextIndex creates a collection with a full text index.
func TestCreateFullTextIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.CreateFullTextIndexOptions{
		nil,
		&driver.CreateFullTextIndexOptions{MinLength: 2},
		&driver.CreateFullTextIndexOptions{MinLength: 20},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("fulltext_index_test_%d", i), nil, t)

		idx, err := col.CreateFullTextIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Remove index
		if err := idx.Remove(nil); err != nil {
			t.Fatalf("Failed to remove index '%s': %s", idx.Name(), describe(err))
		}

		// Index must not exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}

// TestCreateGeoIndex creates a collection with a geo index.
func TestCreateGeoIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.CreateGeoIndexOptions{
		nil,
		&driver.CreateGeoIndexOptions{GeoJSON: true},
		&driver.CreateGeoIndexOptions{GeoJSON: false},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("geo_index_test_%d", i), nil, t)

		idx, err := col.CreateGeoIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Remove index
		if err := idx.Remove(nil); err != nil {
			t.Fatalf("Failed to remove index '%s': %s", idx.Name(), describe(err))
		}

		// Index must not exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}

// TestCreateHashIndex creates a collection with a hash index.
func TestCreateHashIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.CreateHashIndexOptions{
		nil,
		&driver.CreateHashIndexOptions{Unique: true, Sparse: false},
		&driver.CreateHashIndexOptions{Unique: true, Sparse: true},
		&driver.CreateHashIndexOptions{Unique: false, Sparse: false},
		&driver.CreateHashIndexOptions{Unique: false, Sparse: true},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("hash_index_test_%d", i), nil, t)

		idx, err := col.CreateHashIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Remove index
		if err := idx.Remove(nil); err != nil {
			t.Fatalf("Failed to remove index '%s': %s", idx.Name(), describe(err))
		}

		// Index must not exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}

// TestCreatePersistentIndex creates a collection with a persistent index.
func TestCreatePersistentIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.CreatePersistentIndexOptions{
		nil,
		&driver.CreatePersistentIndexOptions{Unique: true, Sparse: false},
		&driver.CreatePersistentIndexOptions{Unique: true, Sparse: true},
		&driver.CreatePersistentIndexOptions{Unique: false, Sparse: false},
		&driver.CreatePersistentIndexOptions{Unique: false, Sparse: true},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("persistent_index_test_%d", i), nil, t)

		idx, err := col.CreatePersistentIndex(nil, []string{"age", "name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Remove index
		if err := idx.Remove(nil); err != nil {
			t.Fatalf("Failed to remove index '%s': %s", idx.Name(), describe(err))
		}

		// Index must not exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}

// TestCreateSkipListIndex creates a collection with a skiplist index.
func TestCreateSkipListIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.CreateSkipListIndexOptions{
		nil,
		&driver.CreateSkipListIndexOptions{Unique: true, Sparse: false},
		&driver.CreateSkipListIndexOptions{Unique: true, Sparse: true},
		&driver.CreateSkipListIndexOptions{Unique: false, Sparse: false},
		&driver.CreateSkipListIndexOptions{Unique: false, Sparse: true},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("skiplist_index_test_%d", i), nil, t)

		idx, err := col.CreateSkipListIndex(nil, []string{"name", "title"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Remove index
		if err := idx.Remove(nil); err != nil {
			t.Fatalf("Failed to remove index '%s': %s", idx.Name(), describe(err))
		}

		// Index must not exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}
