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

// TestDefaultIndexes creates a collection without any custom index.
func TestDefaultIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	col := ensureCollection(nil, db, "def_indexes_test", nil, t)

	// Get list of indexes
	if idxs, err := col.Indexes(context.Background()); err != nil {
		t.Fatalf("Failed to get indexes: %s", describe(err))
	} else {
		if len(idxs) != 1 {
			// 1 is always added by the system
			t.Errorf("Expected 1 index, got %d", len(idxs))
		}
	}
}

// TestDefaultEdgeIndexes creates a edge collection without any custom index.
func TestDefaultEdgeIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	col := ensureCollection(nil, db, "def_indexes_edge_test", &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge}, t)

	// Get list of indexes
	if idxs, err := col.Indexes(context.Background()); err != nil {
		t.Fatalf("Failed to get indexes: %s", describe(err))
	} else {
		if len(idxs) != 2 {
			// 2 is always added by the system
			t.Errorf("Expected 2 index, got %d", len(idxs))
		}

		// ensure edge type returned
		var existed bool
		for _, idx := range idxs {
			if idx.Type() == driver.EdgeIndex {
				existed = true
				break
			}
		}

		if !existed {
			t.Errorf("Expected `%s` index presents, got no", driver.EdgeIndex)
		}
	}
}

// TestCreateFullTextIndex creates a collection with a full text index.
func TestIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	col := ensureCollection(nil, db, "indexes_test", nil, t)

	// Create some indexes
	if idx, _, err := col.EnsureFullTextIndex(nil, []string{"name"}, nil); err == nil {
		if idxType := idx.Type(); idxType != driver.FullTextIndex {
			t.Errorf("Expected FullTextIndex, found `%s`", idxType)
		}
	} else {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}

	if idx, _, err := col.EnsureHashIndex(nil, []string{"age", "gender"}, nil); err == nil {
		if idxType := idx.Type(); idxType != driver.HashIndex {
			t.Errorf("Expected HashIndex, found `%s`", idxType)
		}
	} else {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}

	// Get list of indexes
	if idxs, err := col.Indexes(context.Background()); err != nil {
		t.Fatalf("Failed to get indexes: %s", describe(err))
	} else {
		if len(idxs) != 3 {
			// We made 2 indexes, 1 is always added by the system
			t.Errorf("Expected 3 indexes, got %d", len(idxs))
		}

		// Try opening the indexes 1 by 1
		for _, x := range idxs {
			if idx, err := col.Index(nil, x.Name()); err != nil {
				t.Errorf("Failed to open index '%s': %s", x.Name(), describe(err))
			} else if idx.Name() != x.Name() {
				t.Errorf("Got different index name. Expected '%s', got '%s'", x.Name(), idx.Name())
			}
		}
	}

	// Check index count
	if stats, err := col.Statistics(nil); err != nil {
		t.Fatalf("Statistics failed: %s", describe(err))
	} else if stats.Figures.Indexes.Count != 3 {
		// 3 because 1 system index + 2 created above
		t.Errorf("Expected 3 indexes, got %d", stats.Figures.Indexes.Count)
	}
}

// TestMultipleIndexes creates a collection with a full text index.
func TestMultipleIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	col := ensureCollection(nil, db, "multiple_indexes_test", nil, t)

	// Create some indexes of same type & fields, but different options
	if _, _, err := col.EnsureFullTextIndex(nil, []string{"name"}, &driver.EnsureFullTextIndexOptions{MinLength: 2}); err != nil {
		t.Fatalf("Failed to create new index (1): %s", describe(err))
	}
	if _, _, err := col.EnsureFullTextIndex(nil, []string{"name"}, &driver.EnsureFullTextIndexOptions{MinLength: 7}); err != nil {
		t.Fatalf("Failed to create new index (2): %s", describe(err))
	}

	// Get list of indexes
	if idxs, err := col.Indexes(context.Background()); err != nil {
		t.Fatalf("Failed to get indexes: %s", describe(err))
	} else {
		if len(idxs) != 3 {
			// We made 2 indexes, 1 is always added by the system
			t.Errorf("Expected 3 indexes, got %d", len(idxs))
		}

		// Try opening the indexes 1 by 1
		for _, x := range idxs {
			if idx, err := col.Index(nil, x.Name()); err != nil {
				t.Errorf("Failed to open index '%s': %s", x.Name(), describe(err))
			} else if idx.Name() != x.Name() {
				t.Errorf("Got different index name. Expected '%s', got '%s'", x.Name(), idx.Name())
			}
		}
	}

	// Check index count
	if stats, err := col.Statistics(nil); err != nil {
		t.Fatalf("Statistics failed: %s", describe(err))
	} else if stats.Figures.Indexes.Count != 3 {
		// 3 because 1 system index + 2 created above
		t.Errorf("Expected 3 indexes, got %d", stats.Figures.Indexes.Count)
	}
}

// TestIndexesDeduplicateHash tests no-deduplicate on hash index.
func TestIndexesDeduplicateHash(t *testing.T) {
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	if !isv32p {
		t.Skip("Test requires 3.2")
	} else {
		db := ensureDatabase(nil, c, "index_test", nil, t)

		{
			// Create some indexes with de-duplication off
			col := ensureCollection(nil, db, "indexes_hash_deduplicate_false_test", nil, t)
			if _, _, err := col.EnsureHashIndex(nil, []string{"tags[*]"}, &driver.EnsureHashIndexOptions{
				Unique:        true,
				Sparse:        false,
				NoDeduplicate: true,
			}); err != nil {
				t.Fatalf("Failed to create new index: %s", describe(err))
			}

			doc := struct {
				Tags []string `json:"tags"`
			}{
				Tags: []string{"a", "a", "b"},
			}
			if _, err := col.CreateDocument(nil, doc); !driver.IsConflict(err) {
				t.Errorf("Expected Conflict error, got %s", describe(err))
			}
		}

		{
			// Create some indexes with de-duplication on
			col := ensureCollection(nil, db, "indexes_hash_deduplicate_true_test", nil, t)
			if _, _, err := col.EnsureHashIndex(nil, []string{"tags"}, &driver.EnsureHashIndexOptions{
				Unique:        true,
				Sparse:        false,
				NoDeduplicate: false,
			}); err != nil {
				t.Fatalf("Failed to create new index: %s", describe(err))
			}

			doc := struct {
				Tags []string `json:"tags"`
			}{
				Tags: []string{"a", "a", "b"},
			}
			if _, err := col.CreateDocument(nil, doc); err != nil {
				t.Errorf("Expected success, got %s", describe(err))
			}
		}
	}
}

// TestIndexesDeduplicateSkipList tests no-deduplicate on skiplist index.
func TestIndexesDeduplicateSkipList(t *testing.T) {
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	if !isv32p {
		t.Skip("Test requires 3.2")
	} else {
		db := ensureDatabase(nil, c, "index_test", nil, t)

		{
			// Create some indexes with de-duplication off
			col := ensureCollection(nil, db, "indexes_skiplist_deduplicate_false_test", nil, t)
			if _, _, err := col.EnsureSkipListIndex(nil, []string{"tags[*]"}, &driver.EnsureSkipListIndexOptions{
				Unique:        true,
				Sparse:        false,
				NoDeduplicate: true,
			}); err != nil {
				t.Fatalf("Failed to create new index: %s", describe(err))
			}

			doc := struct {
				Tags []string `json:"tags"`
			}{
				Tags: []string{"a", "a", "b"},
			}
			if _, err := col.CreateDocument(nil, doc); !driver.IsConflict(err) {
				t.Errorf("Expected Conflict error, got %s", describe(err))
			}
		}

		{
			// Create some indexes with de-duplication on
			col := ensureCollection(nil, db, "indexes_skiplist_deduplicate_true_test", nil, t)
			if _, _, err := col.EnsureSkipListIndex(nil, []string{"tags"}, &driver.EnsureSkipListIndexOptions{
				Unique:        true,
				Sparse:        false,
				NoDeduplicate: false,
			}); err != nil {
				t.Fatalf("Failed to create new index: %s", describe(err))
			}

			doc := struct {
				Tags []string `json:"tags"`
			}{
				Tags: []string{"a", "a", "b"},
			}
			if _, err := col.CreateDocument(nil, doc); err != nil {
				t.Errorf("Expected success, got %s", describe(err))
			}
		}
	}
}
