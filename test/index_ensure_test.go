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

// TestEnsureFullTextIndex creates a collection with a full text index.
func TestEnsureFullTextIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureFullTextIndexOptions{
		nil,
		&driver.EnsureFullTextIndexOptions{MinLength: 2},
		&driver.EnsureFullTextIndexOptions{MinLength: 20},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("fulltext_index_test_%d", i), nil, t)

		idx, created, err := col.EnsureFullTextIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}
		if !created {
			t.Error("Expected created to be true, got false")
		}
		if idxType := idx.Type(); idxType != driver.FullTextIndex {
			t.Errorf("Expected FullTextIndex, found `%s`", idxType)
		}
		if options != nil && idx.MinLength() != options.MinLength {
			t.Errorf("Expected %d, found `%d`", options.MinLength, idx.MinLength())
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Ensure again, created must be false now
		_, created, err = col.EnsureFullTextIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to re-create index: %s", describe(err))
		}
		if created {
			t.Error("Expected created to be false, got true")
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

// TestEnsureGeoIndex creates a collection with a geo index.
func TestEnsureGeoIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureGeoIndexOptions{
		nil,
		&driver.EnsureGeoIndexOptions{GeoJSON: true},
		&driver.EnsureGeoIndexOptions{GeoJSON: false},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("geo_index_test_%d", i), nil, t)

		idx, created, err := col.EnsureGeoIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}
		if !created {
			t.Error("Expected created to be true, got false")
		}
		if idxType := idx.Type(); idxType != driver.GeoIndex {
			t.Errorf("Expected GeoIndex, found `%s`", idxType)
		}
		if options != nil && idx.GeoJSON() != options.GeoJSON {
			t.Errorf("Expected GeoJSON to be %t, found `%t`", options.GeoJSON, idx.GeoJSON())
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Ensure again, created must be false now
		_, created, err = col.EnsureGeoIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to re-create index: %s", describe(err))
		}
		if created {
			t.Error("Expected created to be false, got true")
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

// TestEnsureHashIndex creates a collection with a hash index.
func TestEnsureHashIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureHashIndexOptions{
		nil,
		&driver.EnsureHashIndexOptions{Unique: true, Sparse: false},
		&driver.EnsureHashIndexOptions{Unique: true, Sparse: true},
		&driver.EnsureHashIndexOptions{Unique: false, Sparse: false},
		&driver.EnsureHashIndexOptions{Unique: false, Sparse: true},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("hash_index_test_%d", i), nil, t)

		idx, created, err := col.EnsureHashIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}
		if !created {
			t.Error("Expected created to be true, got false")
		}
		if idxType := idx.Type(); idxType != driver.HashIndex {
			t.Errorf("Expected HashIndex, found `%s`", idxType)
		}
		if options != nil && idx.Unique() != options.Unique {
			t.Errorf("Expected Unique to be %t, found `%t`", options.Unique, idx.Unique())
		}
		if options != nil && idx.Sparse() != options.Sparse {
			t.Errorf("Expected Sparse to be %t, found `%t`", options.Sparse, idx.Sparse())
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Ensure again, created must be false now
		_, created, err = col.EnsureHashIndex(nil, []string{"name"}, options)
		if err != nil {
			t.Fatalf("Failed to re-create index: %s", describe(err))
		}
		if created {
			t.Error("Expected created to be false, got true")
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

// TestEnsurePersistentIndex creates a collection with a persistent index.
func TestEnsurePersistentIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsurePersistentIndexOptions{
		nil,
		&driver.EnsurePersistentIndexOptions{Unique: true, Sparse: false},
		&driver.EnsurePersistentIndexOptions{Unique: true, Sparse: true},
		&driver.EnsurePersistentIndexOptions{Unique: false, Sparse: false},
		&driver.EnsurePersistentIndexOptions{Unique: false, Sparse: true},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("persistent_index_test_%d", i), nil, t)

		idx, created, err := col.EnsurePersistentIndex(nil, []string{"age", "name"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}
		if !created {
			t.Error("Expected created to be true, got false")
		}
		if idxType := idx.Type(); idxType != driver.PersistentIndex {
			t.Errorf("Expected PersistentIndex, found `%s`", idxType)
		}
		if options != nil && idx.Unique() != options.Unique {
			t.Errorf("Expected Unique to be %t, found `%t`", options.Unique, idx.Unique())
		}
		if options != nil && idx.Sparse() != options.Sparse {
			t.Errorf("Expected Sparse to be %t, found `%t`", options.Sparse, idx.Sparse())
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Ensure again, created must be false now
		_, created, err = col.EnsurePersistentIndex(nil, []string{"age", "name"}, options)
		if err != nil {
			t.Fatalf("Failed to re-create index: %s", describe(err))
		}
		if created {
			t.Error("Expected created to be false, got true")
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

// TestEnsureSkipListIndex creates a collection with a skiplist index.
func TestEnsureSkipListIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureSkipListIndexOptions{
		nil,
		&driver.EnsureSkipListIndexOptions{Unique: true, Sparse: false, NoDeduplicate: true},
		&driver.EnsureSkipListIndexOptions{Unique: true, Sparse: true, NoDeduplicate: true},
		&driver.EnsureSkipListIndexOptions{Unique: false, Sparse: false, NoDeduplicate: false},
		&driver.EnsureSkipListIndexOptions{Unique: false, Sparse: true, NoDeduplicate: false},
	}

	for i, options := range testOptions {
		col := ensureCollection(nil, db, fmt.Sprintf("skiplist_index_test_%d", i), nil, t)

		idx, created, err := col.EnsureSkipListIndex(nil, []string{"name", "title"}, options)
		if err != nil {
			t.Fatalf("Failed to create new index: %s", describe(err))
		}
		if !created {
			t.Error("Expected created to be true, got false")
		}
		if idxType := idx.Type(); idxType != driver.SkipListIndex {
			t.Errorf("Expected SkipListIndex, found `%s`", idxType)
		}
		if options != nil && idx.Unique() != options.Unique {
			t.Errorf("Expected Unique to be %t, found `%t`", options.Unique, idx.Unique())
		}
		if options != nil && idx.Sparse() != options.Sparse {
			t.Errorf("Expected Sparse to be %t, found `%t`", options.Sparse, idx.Sparse())
		}
		if options != nil && !idx.Deduplicate() != options.NoDeduplicate {
			t.Errorf("Expected NoDeduplicate to be %t, found `%t`", options.NoDeduplicate, idx.Deduplicate())
		}

		// Index must exists now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if !found {
			t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
		}

		// Ensure again, created must be false now
		_, created, err = col.EnsureSkipListIndex(nil, []string{"name", "title"}, options)
		if err != nil {
			t.Fatalf("Failed to re-create index: %s", describe(err))
		}
		if created {
			t.Error("Expected created to be false, got true")
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

// TestEnsureTTLIndex creates a collection with a ttl index.
func TestEnsureTTLIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	skipBelowVersion(c, "3.5", t)

	col := ensureCollection(nil, db, "ttl_index_test", nil, t)
	idx, created, err := col.EnsureTTLIndex(nil, "createdAt", 3600, nil)
	if err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}
	if !created {
		t.Error("Expected created to be true, got false")
	}
	if idxType := idx.Type(); idxType != driver.TTLIndex {
		t.Errorf("Expected TTLIndex, found `%s`", idxType)
	}
	if idx.ExpireAfter() != 3600 {
		t.Errorf("Expected ExpireAfter to be 3600, found `%d`", idx.ExpireAfter())
	}

	// Index must exists now
	if found, err := col.IndexExists(nil, idx.Name()); err != nil {
		t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
	} else if !found {
		t.Errorf("Index '%s' does not exist, expected it to exist", idx.Name())
	}

	// Ensure again, created must be false now
	_, created, err = col.EnsureTTLIndex(nil, "createdAt", 3600, nil)
	if err != nil {
		t.Fatalf("Failed to re-create index: %s", describe(err))
	}
	if created {
		t.Error("Expected created to be false, got true")
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
