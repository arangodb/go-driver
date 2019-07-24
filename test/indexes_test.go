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
	"fmt"
	"testing"
	"time"

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

// TestIndexesTTL tests TTL index.
func TestIndexesTTL(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)

	db := ensureDatabase(nil, c, "index_test", nil, t)

	// Create some indexes with de-duplication off
	col := ensureCollection(nil, db, "indexes_ttl_test", nil, t)
	if _, _, err := col.EnsureTTLIndex(nil, "createdAt", 10, nil); err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}

	doc := struct {
		CreatedAt int64 `json:"createdAt,omitempty"`
	}{
		CreatedAt: time.Now().Add(10 * time.Second).Unix(),
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}

	wasThere := false

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for {
		if found, err := col.DocumentExists(ctx, meta.Key); err != nil {
			t.Fatalf("Failed to test if document exists: %s", describe(err))
		} else {
			if found {
				if !wasThere {
					t.Log("Found document")
				}
				wasThere = true
			} else {
				break
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Timeout while waiting for document to be deleted: %s", ctx.Err())
		case <-time.After(time.Second):
			break
		}
	}

	if !wasThere {
		t.Fatalf("Document never existed")
	}
}

var namedIndexTestCases = []struct {
	Name           string
	CreateCallback func(col driver.Collection, name string) (driver.Index, error)
}{
	{
		Name: "FullText",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsureFullTextIndex(nil, []string{"text"}, &driver.EnsureFullTextIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
	{
		Name: "Geo",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsureGeoIndex(nil, []string{"geo"}, &driver.EnsureGeoIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
	{
		Name: "Hash",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsureHashIndex(nil, []string{"name"}, &driver.EnsureHashIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
	{
		Name: "Persistent",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsurePersistentIndex(nil, []string{"pername"}, &driver.EnsurePersistentIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
	{
		Name: "skipList",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsureSkipListIndex(nil, []string{"pername"}, &driver.EnsureSkipListIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
	{
		Name: "TTL",
		CreateCallback: func(col driver.Collection, name string) (driver.Index, error) {
			idx, _, err := col.EnsureTTLIndex(nil, "createdAt", 3600, &driver.EnsureTTLIndexOptions{
				Name: name,
			})
			return idx, err
		},
	},
}

func TestNamedIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)

	db := ensureDatabase(nil, c, "named_index_test", nil, t)
	col := ensureCollection(nil, db, "named_index_test_col", nil, t)

	for _, testCase := range namedIndexTestCases {
		t.Run(fmt.Sprintf("TestNamedIndexes%s", testCase.Name), func(t *testing.T) {
			// Check if index name is forwarded through out all APIs
			idx, err := testCase.CreateCallback(col, testCase.Name)
			if err != nil {
				t.Fatalf("Failed to create index: %s", describe(err))
			}

			if idx.UserName() != testCase.Name {
				t.Errorf("Expected user name: %s, found: %s", testCase.Name, idx.UserName())
			}

			// Now get the index list
			idxlist, err := col.Indexes(nil)
			if err != nil {
				t.Fatalf("Failed to get index list: %s", describe(err))
			}

			found := false
			for _, i := range idxlist {
				if i.ID() == idx.ID() {
					found = true
					if i.UserName() != testCase.Name {
						t.Errorf("Expected user name: %s, found: %s", testCase.Name, i.UserName())
					}
					break
				}
			}

			if !found {
				t.Fatal("Index not found in list")
			}

			// Try to access index by id
			idx2, err := col.Index(nil, idx.Name())
			if err != nil {
				t.Fatalf("Failed to get index by name: %s", describe(err))
			}

			if idx2.UserName() != testCase.Name {
				t.Errorf("Expected user name: %s, found: %s", testCase.Name, idx2.UserName())
			}
		})
	}
}

func TestNamedIndexesClusterInventory(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	colname := "named_index_test_col_inv"
	db := ensureDatabase(nil, c, "named_index_test_inv", nil, t)
	col := ensureCollection(nil, db, colname, nil, t)

	cc, err := c.Cluster(nil)
	if err != nil {
		t.Fatalf("Failed to obtain cluster client: %s", describe(err))
	}

	for _, testCase := range namedIndexTestCases {
		t.Run(fmt.Sprintf("TestNamedIndexes%s", testCase.Name), func(t *testing.T) {
			// Check if index name is forwarded through out all APIs
			idx, err := testCase.CreateCallback(col, testCase.Name)
			if err != nil {
				t.Fatalf("Failed to create index: %s", describe(err))
			}

			inv, err := cc.DatabaseInventory(nil, db)
			if err != nil {
				t.Fatalf("Failed to obtain cluster inventory: %s", describe(err))
			}

			invcol, found := inv.CollectionByName(colname)
			if !found {
				t.Fatalf("Collection not in inventory!")
			}

			found = false
			for _, i := range invcol.Indexes {
				if i.ID == idx.Name() {
					found = true
					if i.Name != testCase.Name {
						t.Errorf("Expected user name: %s, found: %s", testCase.Name, i.Name)
					}
				}
			}

			if !found {
				t.Errorf("Index with id %s not found", idx.ID())
			}
		})
	}
}

func TestTTLIndexesClusterInventory(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	ttl := 3600
	colname := "ttl_index_test_col_inv"
	db := ensureDatabase(nil, c, "index_test_inv", nil, t)
	col := ensureCollection(nil, db, colname, nil, t)

	cc, err := c.Cluster(nil)
	if err != nil {
		t.Fatalf("Failed to obtain cluster client: %s", describe(err))
	}

	idx, _, err := col.EnsureTTLIndex(nil, "createdAt", ttl, nil)
	if err != nil {
		t.Fatalf("Failed to create ttl index: %s", describe(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for {

		var raw []byte
		rctx := driver.WithRawResponse(ctx, &raw)

		inv, err := cc.DatabaseInventory(rctx, db)
		if err != nil {
			t.Fatalf("Failed to obtain cluster inventory: %s", describe(err))
		}

		invcol, found := inv.CollectionByName(colname)
		if !found {
			t.Fatalf("Collection not in inventory!")
		}

		found = false
		for _, i := range invcol.Indexes {
			if i.ID == idx.Name() {
				found = true
				if i.ExpireAfter != ttl {
					t.Errorf("Expected ttl value: %d, found: %d", ttl, i.ExpireAfter)
				}
			}
		}

		if found {
			break
		}

		select {
		case <-time.After(1 * time.Second):
			break
		case <-ctx.Done():
			t.Fatalf("Index not created: %s", describe(ctx.Err()))
		}
	}

}
