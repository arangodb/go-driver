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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

// ensureArangoSearchView is a helper to check if an arangosearch view exists and create it if needed.
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

// checkLinkExists tests if a given collection is linked to the given arangosearch view
func checkLinkExists(ctx context.Context, view driver.ArangoSearchView, colName string, t testEnv) bool {
	props, err := view.Properties(ctx)
	if err != nil {
		t.Fatalf("Failed to get view properties: %s", describe(err))
	}
	links := props.Links
	if _, exists := links[colName]; !exists {
		return false
	}
	return true
}

// tryAddArangoSearchLink is a helper that adds a link to a view and collection.
// It will fail the test when an error occurs and returns wether the link is actually there or not.
func tryAddArangoSearchLink(ctx context.Context, db driver.Database, view driver.ArangoSearchView, colName string, t testEnv) bool {
	addprop := driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			colName: driver.ArangoSearchElementProperties{},
		},
	}
	if err := view.SetProperties(ctx, addprop); err != nil {
		t.Fatalf("Could not create link, view: %s, collection: %s, error: %s", view.Name(), colName, describe(err))
	}
	return checkLinkExists(ctx, view, colName, t)
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
	name := "test_create_inv_view"
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

// TestCreateArangoSearchViewThenRemoveCollection creates an arangosearch view
// with a link to an existing collection and the removes that collection.
func TestCreateArangoSearchViewThenRemoveCollection(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	col := ensureCollection(ctx, db, "someViewTmpCol", nil, t)
	name := "test_create_view_then_rem_col"
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"someViewTmpCol": driver.ArangoSearchElementProperties{},
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

	// Now delete the collection
	if err := col.Remove(ctx); err != nil {
		t.Fatalf("Failed to remove collection '%s': %s", col.Name(), describe(err))
	}

	// Re-check v properties
	p, err = v.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %s", describe(err))
	}
	if len(p.Links) != 0 {
		// TODO is the really the correct expected behavior.
		t.Errorf("Expected 0 links, got %d", len(p.Links))
	}
}

// TestAddCollectionMultipleViews creates a collection and two view. adds the collection to both views
// and checks if the links exist. The links are set via modifying properties.
func TestAddCollectionMultipleViews(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "col_in_multi_view_test", nil, t)
	ensureCollection(ctx, db, "col_in_multi_view", nil, t)
	v1 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view1", nil, t)
	if !tryAddArangoSearchLink(ctx, db, v1, "col_in_multi_view", t) {
		t.Error("Link does not exists")
	}
	v2 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view2", nil, t)
	if !tryAddArangoSearchLink(ctx, db, v2, "col_in_multi_view", t) {
		t.Error("Link does not exists")
	}
}

// TestAddCollectionMultipleViews creates a collection and two view. adds the collection to both views
// and checks if the links exist. The links are set when creating the view.
func TestAddCollectionMultipleViewsViaCreate(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t)
	db := ensureDatabase(ctx, c, "col_in_multi_view_create_test", nil, t)
	ensureCollection(ctx, db, "col_in_multi_view_create", nil, t)
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"col_in_multi_view_create": driver.ArangoSearchElementProperties{},
		},
	}
	v1 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view1", opts, t)
	if !checkLinkExists(ctx, v1, "col_in_multi_view_create", t) {
		t.Error("Link does not exists")
	}
	v2 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view2", opts, t)
	if !checkLinkExists(ctx, v2, "col_in_multi_view_create", t) {
		t.Error("Link does not exists")
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

// TestUseArangoSearchView tries to create a view and actually use it in
// an AQL query.
func TestUseArangoSearchView(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
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
		{
			"John",
			23,
		},
		{
			"Alice",
			43,
		},
		{
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

// TestGetArangoSearchView creates an arangosearch view and then gets it again.
func TestArangoSearchViewProperties35(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.7.1", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	ensureCollection(ctx, db, "someCol", nil, t)
	commitInterval := int64(100)
	sortDir := driver.ArangoSearchSortDirectionDesc
	name := "test_get_asview_35"
	sortField := "foo"
	storedValuesFields := []string{"now", "is", "the", "time"}
	storedValuesCompression := driver.PrimarySortCompressionNone
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			"someCol": driver.ArangoSearchElementProperties{},
		},
		CommitInterval: &commitInterval,
		PrimarySort: []driver.ArangoSearchPrimarySortEntry{{
			Field:     sortField,
			Direction: &sortDir,
		}},
		StoredValues: []driver.StoredValue{{
			Fields:      storedValuesFields,
			Compression: storedValuesCompression,
		}},
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
	// Check asv properties
	p, err := asv.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %s", describe(err))
	}
	if p.CommitInterval == nil || *p.CommitInterval != commitInterval {
		t.Error("CommitInterval was not set properly")
	}
	if len(p.PrimarySort) != 1 {
		t.Fatalf("Primary sort expected length: %d, found %d", 1, len(p.PrimarySort))
	} else {
		ps := p.PrimarySort[0]
		if ps.Field != sortField {
			t.Errorf("Primary Sort field is wrong: %s, expected %s", ps.Field, sortField)
		}
	}

	if len(p.StoredValues) != 1 {
		t.Fatalf("StoredValues expected length: %d, found %d", 1, len(p.StoredValues))
	} else {
		sv := p.StoredValues[0]
		if !assert.Equal(t, sv.Fields, storedValuesFields) {
			t.Errorf("StoredValues field is wrong: %s, expected %s", sv.Fields, storedValuesFields)
		} else if sv.Compression != storedValuesCompression {
			t.Errorf("StoredValues Compression is wrong: %s, expected %s", sv.Compression, storedValuesCompression)
		}
	}
}

// TestArangoSearchPrimarySort
func TestArangoSearchPrimarySort(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	ensureCollection(ctx, db, "primary_col_sort", nil, t)

	boolTrue := true
	boolFalse := false
	directionAsc := driver.ArangoSearchSortDirectionAsc
	directionDesc := driver.ArangoSearchSortDirectionDesc

	testCases := []struct {
		Name              string
		InAscending       *bool
		ExpectedAscending *bool
		InDirection       *driver.ArangoSearchSortDirection
		ExpectedDirection *driver.ArangoSearchSortDirection
		ErrorCode         int
	}{
		{
			Name:      "NoneSet",
			ErrorCode: http.StatusBadRequest, // Bad Parameter
		},
		{
			Name:              "AscTrue",
			InAscending:       &boolTrue,
			ExpectedAscending: &boolTrue,
		},
		{
			Name:              "AscFalse",
			InAscending:       &boolFalse,
			ExpectedAscending: &boolFalse,
		},
		{
			Name:              "DirAsc",
			InDirection:       &directionAsc,
			ExpectedAscending: &boolTrue, // WAT!? Setting direction = asc returns asc = true
		},
		{
			Name:              "DirDesc",
			InDirection:       &directionDesc,
			ExpectedAscending: &boolFalse,
		},
		{
			Name:        "SetBothAsc",
			InDirection: &directionAsc,
			InAscending: &boolTrue,
			ErrorCode:   http.StatusBadRequest,
		},
		{
			Name:        "SetBothDesc",
			InDirection: &directionDesc,
			InAscending: &boolFalse,
			ErrorCode:   http.StatusBadRequest,
		},
		{
			Name:        "DirAscAscFalse",
			InDirection: &directionAsc,
			InAscending: &boolTrue,
			ErrorCode:   http.StatusBadRequest,
		},
		{
			Name:        "DirDescAscTrue",
			InDirection: &directionAsc,
			InAscending: &boolTrue,
			ErrorCode:   http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Create the view with given parameters
			opts := &driver.ArangoSearchViewProperties{
				Links: driver.ArangoSearchLinks{
					"primary_col_sort": driver.ArangoSearchElementProperties{},
				},
				PrimarySort: []driver.ArangoSearchPrimarySortEntry{{
					Field:     "foo",
					Ascending: testCase.InAscending,
					Direction: testCase.InDirection,
				}},
			}

			name := fmt.Sprintf("%s-view", testCase.Name)

			if _, err := db.CreateArangoSearchView(ctx, name, opts); err != nil {

				if !driver.IsArangoErrorWithCode(err, testCase.ErrorCode) {
					t.Fatalf("Failed to create view '%s': %s", name, describe(err))
				} else {
					// end test here
					return
				}
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
			// Check asv properties
			p, err := asv.Properties(ctx)
			if err != nil {
				t.Fatalf("Properties failed: %s", describe(err))
			}
			if len(p.PrimarySort) != 1 {
				t.Fatalf("Primary sort expected length: %d, found %d", 1, len(p.PrimarySort))
			} else {
				ps := p.PrimarySort[0]
				if ps.Ascending == nil {
					if testCase.ExpectedAscending != nil {
						t.Errorf("Expected Ascending to be nil")
					}
				} else {
					if testCase.ExpectedAscending == nil {
						t.Errorf("Expected Ascending to be non nil")
					} else if ps.GetAscending() != *testCase.ExpectedAscending {
						t.Errorf("Expected Ascending to be %t, found %t", *testCase.ExpectedAscending, ps.GetAscending())
					}
				}

				if ps.Direction == nil {
					if testCase.ExpectedDirection != nil {
						t.Errorf("Expected Direction to be nil")
					}
				} else {
					if testCase.ExpectedDirection == nil {
						t.Errorf("Expected Direction to be non nil")
					} else if ps.GetDirection() != *testCase.ExpectedDirection {
						t.Errorf("Expected Direction to be %s, found %s", string(*testCase.ExpectedDirection), string(ps.GetDirection()))
					}
				}
			}
		})
	}
}

func newBool(v bool) *bool {
	return &v
}

// TestArangoSearchViewProperties353 tests for custom analyzers.
func TestArangoSearchViewProperties353(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5.3", t)
	skipNoCluster(c, t)
	db := ensureDatabase(ctx, c, "view_test", nil, t)
	colname := "someCol"
	ensureCollection(ctx, db, colname, nil, t)
	name := "test_get_asview_353"
	analyzerName := "myanalyzer"
	opts := &driver.ArangoSearchViewProperties{
		Links: driver.ArangoSearchLinks{
			colname: driver.ArangoSearchElementProperties{
				AnalyzerDefinitions: []driver.ArangoSearchAnalyzerDefinition{
					{
						Name: analyzerName,
						Type: driver.ArangoSearchAnalyzerTypeNorm,
						Properties: driver.ArangoSearchAnalyzerProperties{
							Locale: "en_US",
							Case:   driver.ArangoSearchCaseLower,
						},
						Features: []driver.ArangoSearchAnalyzerFeature{
							driver.ArangoSearchAnalyzerFeaturePosition,
							driver.ArangoSearchAnalyzerFeatureFrequency,
						},
					},
				},
				IncludeAllFields: newBool(true),
				InBackground:     newBool(false),
			},
		},
	}
	_, err := db.CreateArangoSearchView(ctx, name, opts)
	require.NoError(t, err)
	// Get view
	v, err := db.View(ctx, name)
	require.NoError(t, err)
	asv, err := v.ArangoSearchView()
	require.NoError(t, err)
	// Check asv properties
	p, err := asv.Properties(ctx)
	require.NoError(t, err)
	require.Contains(t, p.Links, colname)

	// get cluster inventory
	cluster, err := c.Cluster(ctx)
	require.NoError(t, err)
	inv, err := cluster.DatabaseInventory(ctx, db)
	require.NoError(t, err)
	p2, found := inv.ViewByName(name)
	require.True(t, found)

	require.Contains(t, p2.Links, colname)
	link := p2.Links[colname]
	require.Len(t, link.AnalyzerDefinitions, 2)
	analyzer := &link.AnalyzerDefinitions[1]
	require.EqualValues(t, analyzer.Name, analyzerName)
	require.EqualValues(t, analyzer.Type, driver.ArangoSearchAnalyzerTypeNorm)
	require.Len(t, analyzer.Features, 2)
	require.Contains(t, analyzer.Features, driver.ArangoSearchAnalyzerFeatureFrequency)
	require.Contains(t, analyzer.Features, driver.ArangoSearchAnalyzerFeaturePosition)
	require.EqualValues(t, analyzer.Properties.Locale, "en_US")
	require.EqualValues(t, analyzer.Properties.Case, driver.ArangoSearchCaseLower)
	require.Equal(t, newBool(true), link.IncludeAllFields)
}
