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

// ensureCollection is a helper to check if a collection exists and create if if needed.
// It will fail the test when an error occurs.
func ensureCollection(ctx context.Context, db driver.Database, name string, options *driver.CreateCollectionOptions, t testEnv) driver.Collection {
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

// assertCollection is a helper to check if a collection exists and fail if it does not.
func assertCollection(ctx context.Context, db driver.Database, name string, t *testing.T) driver.Collection {
	c, err := db.Collection(ctx, name)
	if driver.IsNotFound(err) {
		t.Fatalf("Collection '%s': does not exist", name)
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

// TestCreateSatelliteCollection create a satellite collection
func TestCreateSatelliteCollection(t *testing.T) {
	skipNoEnterprise(t)
	c := createClientFromEnv(t, true)
	_, err := c.Cluster(nil)
	if driver.IsPreconditionFailed(err) {
		t.Skipf("Not a cluster")
	} else if err != nil {
		t.Fatalf("Failed to get cluster: %s", describe(err))
	}
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_create_collection_satellite"
	options := driver.CreateCollectionOptions{
		ReplicationFactor: driver.ReplicationFactorSatellite,
	}
	if _, err := db.CreateCollection(nil, name, &options); err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// Collection must exist now
	if found, err := db.CollectionExists(nil, name); err != nil {
		t.Errorf("CollectionExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("CollectionExists('%s') return false, expected true", name)
	}
	// Check if the collection is a satellite collection
	if col, err := db.Collection(nil, name); err != nil {
		t.Errorf("Collection('%s') failed: %s", name, describe(err))
	} else {
		if prop, err := col.Properties(nil); err != nil {
			t.Errorf("Properties() failed: %s", describe(err))
		} else {
			if !prop.IsSatellite() {
				t.Errorf("Collection %s is not satellite", name)
			}
		}
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

// TestLoadUnloadCollection creates a collection and unloads, loads & unloads it.
func TestLoadUnloadCollection(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_load_collection"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// Collection must be loaded
	if status, err := col.Status(nil); err != nil {
		t.Errorf("Status failed: %s", describe(err))
	} else if status != driver.CollectionStatusLoaded {
		t.Errorf("Expected status loaded, got %v", status)
	}

	// Unload the collection now
	if err := col.Unload(nil); err != nil {
		t.Errorf("Unload failed: %s", describe(err))
	}

	// Collection must be unloaded
	deadline := time.Now().Add(time.Second * 15)
	for {
		if status, err := col.Status(nil); err != nil {
			t.Fatalf("Status failed: %s", describe(err))
		} else if status != driver.CollectionStatusUnloaded {
			if time.Now().After(deadline) {
				t.Errorf("Expected status unloaded, got %v", status)
				break
			} else {
				time.Sleep(time.Millisecond * 10)
			}
		} else {
			break
		}
	}

	// Load the collection now
	if err := col.Load(nil); err != nil {
		t.Errorf("Load failed: %s", describe(err))
	}

	// Collection must be loaded
	deadline = time.Now().Add(time.Second * 15)
	for {
		if status, err := col.Status(nil); err != nil {
			t.Fatalf("Status failed: %s", describe(err))
		} else if status != driver.CollectionStatusLoaded {
			if time.Now().After(deadline) {
				t.Errorf("Expected status loaded, got %v", status)
				break
			} else {
				time.Sleep(time.Millisecond * 10)
			}
		} else {
			break
		}
	}
}

// TestCollectionName creates a collection and checks its name
func TestCollectionName(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_name"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	if col.Name() != name {
		t.Errorf("Collection.Name() is wrong, got '%s', expected '%s'", col.Name(), name)
	}
}

// TestCollectionTruncate creates a collection, adds some documents and truncates it.
func TestCollectionTruncate(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_truncate"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}

	// create some documents
	for i := 0; i < 10; i++ {
		doc := Book{Title: fmt.Sprintf("Book %d", i)}
		if _, err := col.CreateDocument(nil, doc); err != nil {
			t.Fatalf("Failed to create document: %s", describe(err))
		}
	}

	// count before truncation
	if c, err := col.Count(nil); err != nil {
		t.Errorf("Failed to count documents: %s", describe(err))
	} else if c != 10 {
		t.Errorf("Expected 10 documents, got %d", c)
	}

	// Truncate collection
	if err := col.Truncate(nil); err != nil {
		t.Errorf("Failed to truncate collection: %s", describe(err))
	}

	// count after truncation
	if c, err := col.Count(nil); err != nil {
		t.Errorf("Failed to count documents: %s", describe(err))
	} else if c != 0 {
		t.Errorf("Expected 0 documents, got %d", c)
	}
}

// TestCollectionProperties creates a collection and checks its properties
func TestCollectionProperties(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_properties"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	if p, err := col.Properties(nil); err != nil {
		t.Errorf("Failed to fetch collection properties: %s", describe(err))
	} else {
		if p.ID == "" {
			t.Errorf("Got empty collection ID")
		}
		if p.Name != name {
			t.Errorf("Expected name '%s', got '%s'", name, p.Name)
		}
		if p.Type != driver.CollectionTypeDocument {
			t.Errorf("Expected type %d, got %d", driver.CollectionTypeDocument, p.Type)
		}
	}
}

// TestCollectionSetProperties creates a collection and modifies its properties
func TestCollectionSetProperties(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_set_properties"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}

	// Set WaitForSync to false
	waitForSync := false
	if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{WaitForSync: &waitForSync}); err != nil {
		t.Fatalf("Failed to set properties: %s", describe(err))
	}
	if p, err := col.Properties(nil); err != nil {
		t.Errorf("Failed to fetch collection properties: %s", describe(err))
	} else {
		if p.WaitForSync != waitForSync {
			t.Errorf("Expected WaitForSync %v, got %v", waitForSync, p.WaitForSync)
		}
	}

	// Set WaitForSync to true
	waitForSync = true
	if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{WaitForSync: &waitForSync}); err != nil {
		t.Fatalf("Failed to set properties: %s", describe(err))
	}
	if p, err := col.Properties(nil); err != nil {
		t.Errorf("Failed to fetch collection properties: %s", describe(err))
	} else {
		if p.WaitForSync != waitForSync {
			t.Errorf("Expected WaitForSync %v, got %v", waitForSync, p.WaitForSync)
		}
	}

	// Query engine info (on rocksdb, JournalSize is always 0)
	info, err := db.EngineInfo(nil)
	if err != nil {
		t.Fatalf("Failed to get engine info: %s", describe(err))
	}

	if info.Type == driver.EngineTypeMMFiles {
		// Set JournalSize
		journalSize := int64(1048576 * 17)
		if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{JournalSize: journalSize}); err != nil {
			t.Fatalf("Failed to set properties: %s", describe(err))
		}
		if p, err := col.Properties(nil); err != nil {
			t.Errorf("Failed to fetch collection properties: %s", describe(err))
		} else {
			if p.JournalSize != journalSize {
				t.Errorf("Expected JournalSize %v, got %v", journalSize, p.JournalSize)
			}
		}

		// Set JournalSize again
		journalSize = int64(1048576 * 21)
		if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{JournalSize: journalSize}); err != nil {
			t.Fatalf("Failed to set properties: %s", describe(err))
		}
		if p, err := col.Properties(nil); err != nil {
			t.Errorf("Failed to fetch collection properties: %s", describe(err))
		} else {
			if p.JournalSize != journalSize {
				t.Errorf("Expected JournalSize %v, got %v", journalSize, p.JournalSize)
			}
		}
	} else {
		t.Skipf("JournalSize tests are being skipped on engine type '%s'", info.Type)
	}

	// Test replication factor
	if _, err := c.Cluster(nil); err == nil {
		// Set ReplicationFactor to 2
		replFact := 2
		ctx := driver.WithEnforceReplicationFactor(context.Background(), false)
		if err := col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{ReplicationFactor: replFact}); err != nil {
			t.Fatalf("Failed to set properties: %s", describe(err))
		}
		if p, err := col.Properties(nil); err != nil {
			t.Errorf("Failed to fetch collection properties: %s", describe(err))
		} else {
			if p.ReplicationFactor != replFact {
				t.Errorf("Expected ReplicationFactor %d, got %d", replFact, p.ReplicationFactor)
			}
		}

		// Set ReplicationFactor back 1
		replFact = 1
		if err := col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{ReplicationFactor: replFact}); err != nil {
			t.Fatalf("Failed to set properties: %s", describe(err))
		}
		if p, err := col.Properties(nil); err != nil {
			t.Errorf("Failed to fetch collection properties: %s", describe(err))
		} else {
			if p.ReplicationFactor != replFact {
				t.Errorf("Expected ReplicationFactor %d, got %d", replFact, p.ReplicationFactor)
			}
		}
	} else if driver.IsPreconditionFailed(err) {
		t.Logf("ReplicationFactor tests skipped because we're not running in a cluster")
	} else {
		t.Errorf("Cluster failed: %s", describe(err))
	}
}

// TestCollectionRevision creates a collection, checks revision after adding documents.
func TestCollectionRevision(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_revision"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}

	// create some documents
	for i := 0; i < 10; i++ {
		before, err := col.Revision(nil)
		if err != nil {
			t.Fatalf("Failed to fetch before revision: %s", describe(err))
		}
		doc := Book{Title: fmt.Sprintf("Book %d", i)}
		if _, err := col.CreateDocument(nil, doc); err != nil {
			t.Fatalf("Failed to create document: %s", describe(err))
		}
		after, err := col.Revision(nil)
		if err != nil {
			t.Fatalf("Failed to fetch after revision: %s", describe(err))
		}
		if before == after {
			t.Errorf("Expected revision before, after to be different. Got '%s', '%s'", before, after)
		}
	}
}

// TestCollectionStatistics creates a collection, checks statistics after adding documents.
func TestCollectionStatistics(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_statistics"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}

	// create some documents
	for i := 0; i < 10; i++ {
		before, err := col.Statistics(nil)
		if err != nil {
			t.Fatalf("Failed to fetch before statistics: %s", describe(err))
		}
		doc := Book{Title: fmt.Sprintf("Book %d", i)}
		if _, err := col.CreateDocument(nil, doc); err != nil {
			t.Fatalf("Failed to create document: %s", describe(err))
		}
		after, err := col.Statistics(nil)
		if err != nil {
			t.Fatalf("Failed to fetch after statistics: %s", describe(err))
		}
		if before.Count+1 != after.Count {
			t.Errorf("Expected Count before, after to be 1 different. Got %d, %d", before.Count, after.Count)
		}
		if before.Figures.DataFiles.FileSize > after.Figures.DataFiles.FileSize {
			t.Errorf("Expected DataFiles.FileSize before <= after. Got %d, %d", before.Figures.DataFiles.FileSize, after.Figures.DataFiles.FileSize)
		}
	}
}
