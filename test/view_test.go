//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

	driver "github.com/arangodb/go-driver"
)

// ensureArangoSearchView is a helper to check if an arangosearch view exists and create if if needed.
// It will fail the test when an error occurs.
func ensureArangoSearchView(ctx context.Context, db driver.Database, name string, options *driver.ArangoSearchViewProperties, t testEnv) driver.ArangoSearchView {
	v, err := db.View(ctx, name)
	if driver.IsNotFound(err) {
		v, err = db.CreateArangoSearchView(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create arangosearch view '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open view '%s': %s", name, describe(err))
	}
	result, err := v.ArangoSearchView()
	if err != nil {
		t.Fatalf("Failed to open view '%s' as arangosearch view: %s", name, describe(err))
	}
	return result
}

// assertArangoSearchView is a helper to check if an arangosearch view exists and fail if it does not.
func assertArangoSearchView(ctx context.Context, db driver.Database, name string, t *testing.T) driver.ArangoSearchView {
	v, err := db.View(ctx, name)
	if driver.IsNotFound(err) {
		t.Fatalf("View '%s': does not exist", name)
	} else if err != nil {
		t.Fatalf("Failed to open view '%s': %s", name, describe(err))
	}
	result, err := v.ArangoSearchView()
	if err != nil {
		t.Fatalf("Failed to open view '%s' as arangosearch view: %s", name, describe(err))
	}
	return result
}

// TestCreateArangoSearchView creates an arangosearch view and then checks that it exists.
func TestCreateArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	ensureCollection(ctx, db, "someCol", nil, t)
	name := "test_create_asview"
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"someCol": driver.ArangoSearchElementProperties{},
		},
	}
	v, err := db.CreateArangoSearchView(ctx, name, opts)
	if err != nil {
		t.Fatalf("Failed to create view '%s': %s", name, describe(err))
	}
	// View must exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", name)
	}
	// Check v.Name
	if actualName := v.Name(); actualName != name {
		t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, name)
	}
	// Check v properties
	p, err := v.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %s", describe(err))
	}
	if len(p.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(p.Links))
	}
}

// TestCreateArangoSearchViewInvalidLinks attempts to create an arangosearch view with invalid links and then checks that it does not exists.
func TestCreateArangoSearchViewInvalidLinks(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	name := "test_create_asview"
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"some_nonexistent_col": driver.ArangoSearchElementProperties{},
		},
	}
	_, err := db.CreateArangoSearchView(ctx, name, opts)
	if err == nil {
		t.Fatalf("Creating view did not fail")
	}
	// View must not exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("ViewExists('%s') return true, expected false", name)
	}
	// Try to open view, must fail as well
	if v, err := db.View(ctx, name); !driver.IsNotFound(err) {
		t.Errorf("Expected NotFound error from View('%s'), got %s instead (%#v)", name, describe(err), v)
	}
}

// TestCreateEmptyArangoSearchView creates an arangosearch view without any links.
func TestCreateEmptyArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	name := "test_create_empty_asview"
	v, err := db.CreateArangoSearchView(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to create view '%s': %s", name, describe(err))
	}
	// View must exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", name)
	}
	// Check v properties
	p, err := v.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %s", describe(err))
	}
	if len(p.Links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(p.Links))
	}
}

// TestCreateDuplicateArangoSearchView creates an arangosearch view twice and then checks that it exists.
func TestCreateDuplicateArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	name := "test_create_dup_asview"
	if _, err := db.CreateArangoSearchView(ctx, name, nil); err != nil {
		t.Fatalf("Failed to create view '%s': %s", name, describe(err))
	}
	// View must exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", name)
	}
	// Try to create again. Must fail
	if _, err := db.CreateArangoSearchView(ctx, name, nil); !driver.IsConflict(err) {
		t.Fatalf("Expect a Conflict error from CreateArangoSearchView, got %s", describe(err))
	}
}

// TestGetArangoSearchView creates an arangosearch view and then gets it again.
func TestGetArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	col := ensureCollection(ctx, db, "someCol", nil, t)
	name := "test_get_asview"
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"someCol": driver.ArangoSearchElementProperties{},
		},
	}
	if _, err := db.CreateArangoSearchView(ctx, name, opts); err != nil {
		t.Fatalf("Failed to create view '%s': %s", name, describe(err))
	}
	// Get view
	v, err := db.View(ctx, name)
	if err != nil {
		t.Fatalf("View('%s') failed: %s", name, describe(err))
	}
	asv, err := v.ArangoSearchView()
	if err != nil {
		t.Fatalf("ArangoSearchView() failed: %s", describe(err))
	}
	// Check v.Name
	if actualName := v.Name(); actualName != name {
		t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, name)
	}
	// Check asv properties
	p, err := asv.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %s", describe(err))
	}
	if len(p.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(p.Links))
	}
	// Check indexes on collection
	indexes, err := col.Indexes(ctx)
	if err != nil {
		t.Fatalf("Indexes() failed: %s", describe(err))
	}
	if len(indexes) != 1 {
		// 1 is always added by the system
		t.Errorf("Expected 1 index, got %d", len(indexes))
	}
}

// TestGetArangoSearchViews creates several arangosearch views and then gets all of them.
func TestGetArangoSearchViews(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	// Get views before adding some
	before, err := db.Views(ctx)
	if err != nil {
		t.Fatalf("Views failed: %s", describe(err))
	}
	// Create views
	names := make([]string, 5)
	for i := 0; i < len(names); i++ {
		names[i] = fmt.Sprintf("test_get_views_%d", i)
		if _, err := db.CreateArangoSearchView(ctx, names[i], nil); err != nil {
			t.Fatalf("Failed to create view '%s': %s", names[i], describe(err))
		}
	}
	// Get views
	after, err := db.Views(ctx)
	if err != nil {
		t.Fatalf("Views failed: %s", describe(err))
	}
	// Check count
	if len(before)+len(names) != len(after) {
		t.Errorf("Expected %d views, got %d", len(before)+len(names), len(after))
	}
	// Check view names
	for _, n := range names {
		found := false
		for _, v := range after {
			if v.Name() == n {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected view '%s' is not found", n)
		}
	}
}

// TestRemoveArangoSearchView creates an arangosearch view and then removes it.
func TestRemoveArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	name := "test_remove_asview"
	v, err := db.CreateArangoSearchView(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// View must exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", name)
	}
	// Now remove it
	if err := v.Remove(ctx); err != nil {
		t.Fatalf("Failed to remove view '%s': %s", name, describe(err))
	}
	// View must not exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("ViewExists('%s') return true, expected false", name)
	}
}

// TestRenameArangoSearchView creates an arangosearch view and then rename it.
func TestRenameArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(nil, c, "view_test", nil, t)
	name := "test_rename_asview"
	v, err := db.CreateArangoSearchView(ctx, name, nil)
	if err != nil {
		t.Fatalf("Failed to create collection '%s': %s", name, describe(err))
	}
	// View must exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", name)
	}
	// Now rename it
	newName := name + "_new"
	if err := v.Rename(ctx, newName); err != nil {
		t.Fatalf("Failed to remove view '%s': %s", name, describe(err))
	}
	// Name() must return the new name
	if actualName := v.Name(); actualName != newName {
		t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, newName)
	}
	// View with old name must not exist now
	if found, err := db.ViewExists(ctx, name); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", name, describe(err))
	} else if found {
		t.Errorf("ViewExists('%s') return true, expected false", name)
	}
	// View with new name must exist now
	if found, err := db.ViewExists(ctx, newName); err != nil {
		t.Errorf("ViewExists('%s') failed: %s", newName, describe(err))
	} else if !found {
		t.Errorf("ViewExists('%s') return false, expected true", newName)
	}
}

// TestUseArangoSearchView tries to create a view and actually use it in
// an AQL query.
func TestUseArangoSearchView(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(nil, c, "view_test", nil, t)
	col := ensureCollection(ctx, db, "some_collection", nil, t)

	ensureArangoSearchView(ctx, db, "some_view", &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"some_collection": driver.ArangoSearchElementProperties{
				Fields: driver.ArangoSearchFields{
					"name": driver.ArangoSearchElementProperties{},
				},
			},
		},
	}, t)

	docs := []UserDoc{
		UserDoc{
			"John",
			23,
		},
		UserDoc{
			"Alice",
			43,
		},
		UserDoc{
			"Helmut",
			56,
		},
	}

	_, errs, err := col.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// now access it via AQL with waitForSync
	{
		cur, err := db.Query(driver.WithQueryCount(ctx), `FOR doc IN some_view SEARCH doc.name == "John" OPTIONS {waitForSync:true} RETURN doc`, nil)
		if err != nil {
			t.Fatalf("Failed to query data using arangodsearch: %s", describe(err))
		} else if cur.Count() != 1 || !cur.HasMore() {
			t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
		}

		var doc UserDoc
		_, err = cur.ReadDocument(ctx, &doc)
		if err != nil {
			t.Fatalf("Failed to read document: %s", describe(err))
		}

		if doc.Name != "John" {
			t.Fatalf("Expected result `John`, found `%s`", doc.Name)
		}
	}

	// now access it via AQL without waitForSync
	{
		cur, err := db.Query(driver.WithQueryCount(ctx), `FOR doc IN some_view SEARCH doc.name == "John" RETURN doc`, nil)
		if err != nil {
			t.Fatalf("Failed to query data using arangodsearch: %s", describe(err))
		} else if cur.Count() != 1 || !cur.HasMore() {
			t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
		}

		var doc UserDoc
		_, err = cur.ReadDocument(ctx, &doc)
		if err != nil {
			t.Fatalf("Failed to read document: %s", describe(err))
		}

		if doc.Name != "John" {
			t.Fatalf("Expected result `John`, found `%s`", doc.Name)
		}
	}
}
