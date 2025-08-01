//
// DISCLAIMER
//
// Copyright 2017-2024 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/util"
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

func TestGetCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_get_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	name := "test_wrong_collection"

	_, err := db.Collection(nil, name)
	require.Error(t, err)

	_, err = db.Collection(driver.WithSkipExistCheck(nil, false), name)
	require.Error(t, err)

	_, err = db.Collection(driver.WithSkipExistCheck(nil, true), name)
	require.NoError(t, err)
}

// TestCreateCollection creates a collection and then checks that it exists.
func TestCreateCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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

// TestCollection_CacheEnabled with cacheEnabled and check if exists
func TestCollection_CacheEnabled(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test_cache_enabled", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	t.Run("Default value", func(t *testing.T) {
		name := "test_create_collection_cache_default"
		_, err := db.CreateCollection(nil, name, nil)
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.False(t, prop.CacheEnabled)
	})

	t.Run("False", func(t *testing.T) {
		name := "test_create_collection_cache_false"
		_, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{
			CacheEnabled: util.NewType(false),
		})
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.False(t, prop.CacheEnabled)
	})

	t.Run("True", func(t *testing.T) {
		name := "test_create_collection_cache_true"
		_, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{
			CacheEnabled: util.NewType(true),
		})
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.True(t, prop.CacheEnabled)
	})

	t.Run("With update", func(t *testing.T) {
		name := "test_create_collection_cache_update"
		_, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{
			CacheEnabled: util.NewType(false),
		})
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.False(t, prop.CacheEnabled)

		err = col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
			CacheEnabled: util.NewType(true),
		})
		require.NoError(t, err)

		prop, err = col.Properties(nil)
		require.NoError(t, err)

		require.True(t, prop.CacheEnabled)
	})
}

// TestCollection_ComputedValues
func TestCollection_ComputedValues(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.10", t)
	db := ensureDatabase(nil, c, "collection_test_computed_values", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	t.Run("Create with ComputedValues", func(t *testing.T) {
		name := "test_users_computed_values"

		// Add an attribute with the creation timestamp to new documents
		computedValue := driver.ComputedValue{
			Name:       "createdAt",
			Expression: "RETURN DATE_NOW()",
			Overwrite:  true,
			ComputeOn:  []driver.ComputeOn{driver.ComputeOnInsert},
		}

		_, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{
			ComputedValues: []driver.ComputedValue{computedValue},
		})
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		// Check if the computed value is in the list of computed values
		require.Len(t, prop.ComputedValues, 1)
		require.Equal(t, computedValue.Name, prop.ComputedValues[0].Name)
		require.Len(t, prop.ComputedValues[0].ComputeOn, 1)
		require.Equal(t, computedValue.ComputeOn[0], prop.ComputedValues[0].ComputeOn[0])
		require.Equal(t, computedValue.Expression, prop.ComputedValues[0].Expression)

		// Create a document
		doc := UserDoc{Name: fmt.Sprintf("Jakub")}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			t.Fatalf("Failed to create document: %s", describe(err))
		}

		// Read document
		var readDoc map[string]interface{}
		if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}

		require.Equal(t, doc.Name, readDoc["name"])

		// Verify that the computed value is set
		createdAtValue, createdAtIsPresent := readDoc["createdAt"]
		require.True(t, createdAtIsPresent)

		t.Logf("createdAtValue raw value: %v", createdAtValue)
		createdAtValueInt64, err := parseInt64FromInterface(createdAtValue)
		require.NoError(t, err)
		t.Logf("createdAtValue parsed value: %v", createdAtValueInt64)

		tm := time.Unix(createdAtValueInt64, 0)
		require.True(t, tm.After(time.Now().Add(-time.Second)))
	})

	t.Run("Update to ComputedValues", func(t *testing.T) {
		name := "test_update_computed_values"

		// Add an attribute with the creation timestamp to new documents
		computedValue := driver.ComputedValue{
			Name:       "createdAt",
			Expression: "RETURN DATE_NOW()",
			Overwrite:  true,
			ComputeOn:  []driver.ComputeOn{driver.ComputeOnInsert},
		}

		_, err := db.CreateCollection(nil, name, nil)
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.Len(t, prop.ComputedValues, 0)

		err = col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
			ComputedValues: []driver.ComputedValue{computedValue},
		})
		require.NoError(t, err)

		// Check if the computed value is in the list of computed values
		col, err = db.Collection(nil, name)
		require.NoError(t, err)

		prop, err = col.Properties(nil)
		require.NoError(t, err)

		require.Len(t, prop.ComputedValues, 1)
	})

	t.Run("Use default ComputeOn values in ComputedValues", func(t *testing.T) {
		name := "test_default_computeon_computed_values"

		// Add an attribute with the creation timestamp to new documents
		computedValue := driver.ComputedValue{
			Name:       "createdAt",
			Expression: "RETURN DATE_NOW()",
			Overwrite:  true,
		}

		_, err := db.CreateCollection(nil, name, nil)
		require.NoError(t, err)

		// Collection must exist now
		col, err := db.Collection(nil, name)
		require.NoError(t, err)

		prop, err := col.Properties(nil)
		require.NoError(t, err)

		require.Len(t, prop.ComputedValues, 0)

		err = col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
			ComputedValues: []driver.ComputedValue{computedValue},
		})
		require.NoError(t, err)

		// Check if the computed value is in the list of computed values
		col, err = db.Collection(nil, name)
		require.NoError(t, err)

		prop, err = col.Properties(nil)
		require.NoError(t, err)

		require.Len(t, prop.ComputedValues, 1)
		// we should get the default value for ComputeOn - ["insert", "update", "replace"]
		require.Len(t, prop.ComputedValues[0].ComputeOn, 3)
	})
}

// TestCreateSatelliteCollection create a satellite collection
func TestCreateSatelliteCollection(t *testing.T) {
	skipNoEnterprise(t)
	c := createClient(t, nil)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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

// TestCreateSmartJoinCollection create a collection with smart join attribute
func TestCreateSmartJoinCollection(t *testing.T) {
	skipNoEnterprise(t)
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	name := "test_create_collection_smart_join"
	nameParent := "test_create_collection_smart_join_parent"

	colParent := ensureCollection(nil, db, nameParent, &driver.CreateCollectionOptions{
		ShardKeys:      []string{"_key"},
		NumberOfShards: 2,
	}, t)
	defer clean(t, nil, colParent)

	options := driver.CreateCollectionOptions{
		DistributeShardsLike: nameParent,
		ShardKeys:            []string{"_key:"},
		SmartJoinAttribute:   "smart",
		NumberOfShards:       2,
	}
	col, err := db.CreateCollection(nil, name, &options)
	require.NoError(t, err)
	defer clean(t, nil, col)

	// Collection must exist now
	found, err := db.CollectionExists(nil, name)
	require.NoError(t, err)
	require.True(t, found, "CollectionExists('%s') return false, expected true", name)

	// Check if the collection has a smart join attribute
	colRead, err := db.Collection(nil, name)
	require.NoError(t, err)

	prop, err := colRead.Properties(nil)
	require.NoError(t, err)
	require.Equal(t, "smart", prop.SmartJoinAttribute)
}

// TestCreateCollectionWithShardingStrategy create a collection with non default sharding strategy
func TestCreateCollectionWithShardingStrategy(t *testing.T) {
	skipNoEnterprise(t)
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_create_collection_sharding_strategy"
	options := driver.CreateCollectionOptions{
		ShardingStrategy: driver.ShardingStrategyCommunityCompat,
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
	// Check if the collection has a smart join attribute
	if col, err := db.Collection(nil, name); err != nil {
		t.Errorf("Collection('%s') failed: %s", name, describe(err))
	} else {
		if prop, err := col.Properties(nil); err != nil {
			t.Errorf("Properties() failed: %s", describe(err))
		} else {
			if prop.ShardingStrategy != driver.ShardingStrategyCommunityCompat {
				t.Errorf("Collection does not have the correct sharding strategy value, expected `%s`, found `%s`", driver.ShardingStrategyCommunityCompat, prop.ShardingStrategy)
			}
		}
	}
}

func parseInt64FromInterface(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8, int16, int32, int64:
		return v.(int64), nil
	case uint, uint8, uint16, uint32, uint64:
		return int64(v.(uint64)), nil
	case float32, float64:
		return int64(v.(float64)), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("value is of type %T, not convertible to int64", v)
	}
}

// TestRemoveCollection creates a collection and then removes it.
func TestRemoveCollection(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	// we are not able to unload RocksDB
	skipIfEngineTypeRocksDB(t, db)
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
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_collection_properties"
	col, err := db.CreateCollection(nil, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}

	if p, err := col.Properties(nil); err != nil {
		t.Errorf("Failed to fetch collection properties: %s", describe(err))
	} else {
		if p.ID == "" {
			t.Errorf("Got empty collection ID")
		}
		if version.Version.CompareTo("3.5") >= 0 {
			if p.GloballyUniqueId == "" {
				t.Errorf("Got empty collection globallyUniqueId")
			}
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
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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

func TestCollectionSetPropertiesSatellite(t *testing.T) {
	skipNoEnterprise(t)
	c := createClient(t, nil)

	// Test replication factor
	if _, err := c.Cluster(nil); err == nil {

		db := ensureDatabase(nil, c, "collection_test_satellite", nil, t)
		defer func() {
			err := db.Remove(nil)
			if err != nil {
				t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
			}
		}()
		name := "test_collection_set_properties_sat"
		col, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{ReplicationFactor: driver.ReplicationFactorSatellite})
		if err != nil {
			t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
		}

		// Set ReplicationFactor to satellite (noop)
		replFact := driver.ReplicationFactorSatellite
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
	} else if driver.IsPreconditionFailed(err) {
		t.Logf("ReplicationFactor tests skipped because we're not running in a cluster")
	} else {
		t.Errorf("Cluster failed: %s", describe(err))
	}
}

// TestCollectionRevision creates a collection, checks revision after adding documents.
func TestCollectionRevision(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	name := "test_collection_revision"
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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

// TestCollectionChecksum creates a collection, checks checksum after adding documents.
func TestCollectionChecksum(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_checksum", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_collection_checksum"
	col, err := db.CreateCollection(nil, name, nil)
	require.NoError(t, err)

	// create some documents
	for i := 0; i < 5; i++ {
		before, err := col.Checksum(nil, false, false)
		require.NoError(t, err)

		doc := Book{Title: fmt.Sprintf("Book %d", i)}
		_, err = col.CreateDocument(nil, doc)
		require.NoError(t, err)

		after, err := col.Checksum(nil, false, false)
		require.NoError(t, err)
		require.NotEqual(t, before.Checksum, after.Checksum)

		afterWithRevision, err := col.Checksum(nil, true, false)
		require.NoError(t, err)
		require.NotEqual(t, before.Checksum, afterWithRevision.Checksum)
		require.NotEqual(t, after.Checksum, afterWithRevision.Checksum)

		afterWithData, err := col.Checksum(nil, false, true)
		require.NoError(t, err)
		require.NotEqual(t, before.Checksum, afterWithData.Checksum)
		require.NotEqual(t, after.Checksum, afterWithData.Checksum)
		require.NotEqual(t, afterWithRevision.Checksum, afterWithData.Checksum)
	}
}

// TestCollectionStatistics creates a collection, checks statistics after adding documents.
func TestCollectionStatistics(t *testing.T) {
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
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

// TestCollectionMinReplFactDeprecatedCreate creates a collection with minReplicationFactor != 1
func TestCollectionMinReplFactDeprecatedCreate(t *testing.T) {
	c := createClient(t, nil)
	version := skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_min_repl_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_min_repl_create"
	minRepl := 2
	options := driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl,
		MinReplicationFactor: minRepl,
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
	// Check if the collection has a minReplicationFactor
	if col, err := db.Collection(nil, name); err != nil {
		t.Errorf("Collection('%s') failed: %s", name, describe(err))
	} else {
		if prop, err := col.Properties(nil); err != nil {
			t.Errorf("Properties() failed: %s", describe(err))
		} else {
			if prop.MinReplicationFactor != minRepl {
				t.Errorf("Collection does not have the correct min replication factor value, "+
					"expected `%d`, found `%d`", minRepl, prop.MinReplicationFactor)
			}
			if version.Version.CompareTo("3.6") >= 0 {
				if prop.WriteConcern != minRepl {
					t.Errorf("Collection does not have the correct WriteConcern value, "+
						"expected `%d`, found `%d`", minRepl, prop.WriteConcern)
				}
			}
		}
	}
}

// TestCollectionMinReplFactDeprecatedInvalid creates a collection with minReplicationFactor > replicationFactor
func TestCollectionMinReplFactDeprecatedInvalid(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_min_repl_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_min_repl_create_invalid"
	minRepl := 2
	options := driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl,
		MinReplicationFactor: minRepl + 1,
	}
	if _, err := db.CreateCollection(nil, name, &options); err == nil {
		t.Fatalf("CreateCollection('%s') did not fail", name)
	}
	// Collection must not exist now
	if found, err := db.CollectionExists(nil, name); err != nil {
		t.Errorf("CollectionExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("Collection %s should not exist", name)
	}
}

// TestCollectionMinReplFactDeprecatedClusterInv tests if minReplicationFactor is forwarded to ClusterInfo
func TestCollectionMinReplFactDeprecatedClusterInv(t *testing.T) {
	c := createClient(t, nil)
	version := skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_min_repl_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_min_repl_cluster_invent"
	minRepl := 2
	ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl,
		MinReplicationFactor: minRepl,
	}, t)

	cc, err := c.Cluster(nil)
	if err != nil {
		t.Fatalf("Failed to get Cluster: %s", describe(err))
	}

	inv, err := cc.DatabaseInventory(nil, db)
	if err != nil {
		t.Fatalf("Failed to get Database Inventory: %s", describe(err))
	}

	col, found := inv.CollectionByName(name)
	if !found {
		t.Fatalf("Failed to get find collection: %s", describe(err))
	}

	if col.Parameters.MinReplicationFactor != minRepl {
		t.Errorf("Collection does not have the correct min replication factor value, expected `%d`, found `%d`",
			minRepl, col.Parameters.MinReplicationFactor)
	}
	if version.Version.CompareTo("3.6") >= 0 {
		if col.Parameters.WriteConcern != minRepl {
			t.Errorf("Collection does not have the correct WriteConcern value, expected `%d`, found `%d`",
				minRepl, col.Parameters.WriteConcern)
		}
	}
}

// TestCollectionMinReplFactDeprecatedSetProp updates the minimal replication factor using SetProperties
func TestCollectionMinReplFactDeprecatedSetProp(t *testing.T) {
	c := createClient(t, nil)
	version := skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_min_repl_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_min_repl_set_prop"
	minRepl := 2
	minReplChanged := 1
	col := ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl,
		MinReplicationFactor: minRepl,
	}, t)

	if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
		MinReplicationFactor: minReplChanged,
	}); err != nil {
		t.Fatalf("Failed to update properties: %s", describe(err))
	}

	if prop, err := col.Properties(nil); err != nil {
		t.Fatalf("Failed to get properties: %s", describe(err))
	} else {
		if prop.MinReplicationFactor != minReplChanged {
			t.Fatalf("MinReplicationFactor not updated, expected %d, found %d", minReplChanged,
				prop.MinReplicationFactor)
		}
		if version.Version.CompareTo("3.6") >= 0 {
			if prop.WriteConcern != minReplChanged {
				t.Fatalf("WriteConcern not updated, expected %d, found %d", minReplChanged, prop.WriteConcern)
			}
		}
	}
}

// TestCollectionMinReplFactDeprecatedSetPropInvalid updates the minimal replication factor
// to an invalid value using SetProperties.
func TestCollectionMinReplFactDeprecatedSetPropInvalid(t *testing.T) {
	c := createClient(t, nil)
	version := skipBelowVersion(c, "3.5", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_min_repl_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_min_repl_set_prop_inv"
	minRepl := 2
	col := ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl,
		MinReplicationFactor: minRepl,
	}, t)

	if err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
		MinReplicationFactor: minRepl + 1,
	}); err == nil {
		t.Errorf("SetProperties did not fail")
	}

	if prop, err := col.Properties(nil); err != nil {
		t.Fatalf("Failed to get properties: %s", describe(err))
	} else {
		if prop.MinReplicationFactor != minRepl {
			t.Fatalf("MinReplicationFactor not updated, expected %d, found %d", minRepl,
				prop.MinReplicationFactor)
		}
		if version.Version.CompareTo("3.6") >= 0 {
			if prop.WriteConcern != minRepl {
				t.Fatalf("WriteConcern not updated, expected %d, found %d", minRepl, prop.WriteConcern)
			}
		}
	}
}

// TestCollectionWriteConcernCreate creates a collection with WriteConcern != 1.
func TestCollectionWriteConcernCreate(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_write_concern_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_write_concern_create"
	minRepl := 2
	options := driver.CreateCollectionOptions{
		ReplicationFactor:    minRepl + 1,
		WriteConcern:         minRepl,
		MinReplicationFactor: minRepl,
	}

	_, err := db.CreateCollection(nil, name, &options)
	require.Nilf(t, err, "Failed to create collection '%s': %s", name, describe(err))

	// Collection must exist now
	found, err := db.CollectionExists(nil, name)
	require.Nilf(t, err, "CollectionExists('%s') failed: %s", name, describe(err))
	require.Equalf(t, true, found, "CollectionExists('%s') return false, expected true", name)

	// Check if the collection has a WriteConcern
	col, err := db.Collection(nil, name)
	require.Nilf(t, err, "Collection('%s') failed: %s", name, describe(err))

	prop, err := col.Properties(nil)
	require.Nilf(t, err, "Properties() failed: %s", describe(err))

	assert.Equalf(t, minRepl, prop.WriteConcern,
		"Collection does not have the correct WriteConcern value, expected `%d`, found `%d`", minRepl,
		prop.WriteConcern)
	assert.Equalf(t, minRepl, prop.MinReplicationFactor,
		"Collection does not have the correct MinReplicationFactor value, expected `%d`, found `%d`", minRepl,
		prop.MinReplicationFactor)
}

// TestCollectionWriteConcernInvalid creates a collection with WriteConcern > replicationFactor
func TestCollectionWriteConcernInvalid(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)

	db := ensureDatabase(nil, c, "collection_write_concern_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_write_concern_invalid"
	minRepl := 2
	options := driver.CreateCollectionOptions{
		ReplicationFactor: minRepl,
		WriteConcern:      minRepl + 1,
	}

	_, err := db.CreateCollection(nil, name, &options)
	require.NotNilf(t, err, "CreateCollection('%s') did not fail", name)

	// Collection must not exist now
	found, err := db.CollectionExists(nil, name)
	require.Nilf(t, err, "CollectionExists('%s') failed: %s", name, describe(err))
	assert.Equalf(t, false, found, "Collection %s should not exist", name)
}

// TestCollectionWriteConcernClusterInv tests if WriteConcern is forwarded to ClusterInfo
func TestCollectionWriteConcernClusterInv(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_write_concern_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_write_concern_cluster_invent"
	minRepl := 2
	ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor: minRepl,
		WriteConcern:      minRepl,
	}, t)

	cc, err := c.Cluster(nil)
	require.Nilf(t, err, "Failed to get Cluster: %s", describe(err))

	inv, err := cc.DatabaseInventory(nil, db)
	require.Nilf(t, err, "Failed to get Database Inventory: %s", describe(err))

	col, found := inv.CollectionByName(name)
	require.Equalf(t, true, found, "Failed to get find collection: %s", describe(err))

	assert.Equalf(t, minRepl, col.Parameters.WriteConcern,
		"Collection does not have the correct WriteConcern value, expected `%d`, found `%d`",
		minRepl, col.Parameters.WriteConcern)
}

// TestCollectionWriteConcernSetProp updates the WriteConcern using SetProperties
func TestCollectionWriteConcernSetProp(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_write_concern_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_write_concern_set_prop"
	minRepl := 2
	writeConcernChanged := 1
	col := ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor: minRepl,
		WriteConcern:      minRepl,
	}, t)

	err := col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
		WriteConcern: writeConcernChanged,
	})
	require.Nilf(t, err, "Failed to update properties: %s", describe(err))

	prop, err := col.Properties(nil)
	require.Nilf(t, err, "Failed to get properties: %s", describe(err))

	assert.Equal(t, writeConcernChanged, prop.WriteConcern)
}

// TestCollectionWriteConcernSetPropInvalid updates the writeConcern to an invalid value using SetProperties.
func TestCollectionWriteConcernSetPropInvalid(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)
	db := ensureDatabase(nil, c, "collection_write_concern_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_write_concern_set_prop_inv"
	minRepl := 2
	defaultWriteConcern := 1
	col := ensureCollection(nil, db, name, &driver.CreateCollectionOptions{
		ReplicationFactor: minRepl,
	}, t)

	prop, err := col.Properties(nil)
	require.Nil(t, err, "failed to get properties")
	require.Equal(t, defaultWriteConcern, prop.WriteConcern, "default value is not set")

	err = col.SetProperties(nil, driver.SetCollectionPropertiesOptions{
		WriteConcern: minRepl + 1,
	})
	require.NotNil(t, err, "SetProperties should fail")

	prop, err = col.Properties(nil)
	require.Nilf(t, err, "Failed to get properties: %s", describe(err))
	assert.Equalf(t, defaultWriteConcern, prop.WriteConcern, "MinReplicationFactor not updated, expected %d, found %d",
		minRepl, prop.WriteConcern)
}

// Test_CollectionShards creates a collection and gets the shards' information.
func Test_CollectionShards(t *testing.T) {
	if getTestMode() != testModeCluster {
		t.Skipf("Not a cluster mode")
	}

	databaseName := getCallerFunctionName()
	c := createClient(t, nil)
	db := ensureDatabase(nil, c, databaseName, nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	name := "test_collection_set_properties"
	col, err := db.CreateCollection(nil, name, &driver.CreateCollectionOptions{
		ReplicationFactor: 2,
		NumberOfShards:    2,
	})
	require.NoError(t, err)

	shards, err := col.Shards(context.Background(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, shards.ID)
	assert.Equal(t, name, shards.Name)
	assert.NotEmpty(t, shards.Status)
	assert.Equal(t, driver.CollectionTypeDocument, shards.Type)
	assert.Equal(t, false, shards.IsSystem)
	assert.NotEmpty(t, shards.GloballyUniqueId)
	assert.Equal(t, false, shards.CacheEnabled)
	assert.Equal(t, false, shards.IsSmart)
	assert.Equal(t, driver.KeyGeneratorTraditional, shards.KeyOptions.Type)
	assert.Equal(t, true, shards.KeyOptions.AllowUserKeys)
	assert.Equal(t, 2, shards.NumberOfShards)
	assert.Equal(t, driver.ShardingStrategyHash, shards.ShardingStrategy)
	assert.Equal(t, []string{"_key"}, shards.ShardKeys)
	require.Len(t, shards.Shards, 2, "expected 2 shards")
	var leaders []driver.ServerID
	for _, dbServers := range shards.Shards {
		require.Lenf(t, dbServers, 2, "expected 2 DB servers for the shard")
		leaders = append(leaders, dbServers[0])
	}
	assert.NotEqualf(t, leaders[0], leaders[1], "the leader shard can not be on the same server")
	assert.Equal(t, 2, shards.ReplicationFactor)
	assert.Equal(t, false, shards.WaitForSync)
	assert.Equal(t, 1, shards.WriteConcern)

	t.Run("Satellite collection", func(t *testing.T) {
		skipNoEnterprise(t)
		col, err := db.CreateCollection(nil, "satellite", &driver.CreateCollectionOptions{
			ReplicationFactor: driver.ReplicationFactorSatellite,
		})
		require.NoError(t, err)

		shards, err := col.Shards(context.Background(), true)
		require.NoError(t, err)
		assert.Equal(t, driver.ReplicationFactorSatellite, shards.ReplicationFactor)
	})
}
