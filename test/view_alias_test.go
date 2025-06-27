//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/util"
)

// ensureArangoSearchView is a helper to check if an arangosearch view exists and create it if needed.
// It will fail the test when an error occurs.
func ensureArangoSearchAliasView(ctx context.Context, db driver.Database, name string, options *driver.ArangoSearchAliasViewProperties, t testEnv) driver.ArangoSearchViewAlias {
	v, err := db.View(ctx, name)
	if driver.IsNotFound(err) {
		v, err = db.CreateArangoSearchAliasView(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create arangosearch view '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open view '%s': %s", name, describe(err))
	}
	result, err := v.ArangoSearchViewAlias()
	if err != nil {
		t.Fatalf("Failed to open view '%s' as arangosearch view: %s", name, describe(err))
	}
	return result
}

// TestSearchViewsAlias tests the arangosearch view alias methods
func TestSearchViewsAlias(t *testing.T) {
	ctx := context.Background()
	c := createClient(t, nil)
	skipBelowVersion(c, "3.10", t)
	skipBelowVersion(c, "3.10", t)
	db := ensureDatabase(ctx, c, "search_view_test_basic", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	nameAlias := "test_add_collection_view_alias"
	nameCol := "col_in_alias_view"
	nameInvInd := "inv_index_alias_view"

	col := ensureCollection(ctx, db, nameCol, nil, t)
	v := ensureArangoSearchAliasView(ctx, db, nameAlias, nil, t)

	p, err := v.Properties(ctx)
	require.NoError(t, err)
	require.Equal(t, p.Type, driver.ViewTypeArangoSearchAlias)
	require.Equal(t, p.Name, nameAlias)
	require.Len(t, p.Indexes, 0)

	_, err = v.ArangoSearchView()
	require.Error(t, err)

	indexOpt := driver.InvertedIndexOptions{
		Name: nameInvInd,
		PrimarySort: driver.InvertedIndexPrimarySort{
			Fields: []driver.ArangoSearchPrimarySortEntry{
				{Field: "test1", Ascending: util.NewType(true)},
				{Field: "test2", Ascending: util.NewType(false)},
			},
			Compression: driver.PrimarySortCompressionLz4,
		},
		Fields: []driver.InvertedIndexField{
			{Name: "field1", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency}, Nested: nil},
			{Name: "field2", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency, driver.ArangoSearchAnalyzerFeaturePosition}, TrackListPositions: false, Nested: nil},
		},
	}
	idx, created, err := col.EnsureInvertedIndex(ctx, &indexOpt)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, nameInvInd, idx.UserName())

	opt := driver.ArangoSearchAliasViewProperties{
		Indexes: []driver.ArangoSearchAliasIndex{
			{
				Collection: nameCol,
				Index:      nameInvInd,
			},
		},
	}
	p, err = v.SetProperties(ctx, opt)
	require.NoError(t, err)
	require.Equal(t, p.Type, driver.ViewTypeArangoSearchAlias)
	require.Equal(t, p.Name, nameAlias)
	require.Len(t, p.Indexes, 1)
	require.Equal(t, p.Indexes[0].Collection, nameCol)
	require.Equal(t, p.Indexes[0].Index, nameInvInd)

	p, err = v.Properties(ctx)
	require.NoError(t, err)
	require.Equal(t, p.Type, driver.ViewTypeArangoSearchAlias)
	require.Equal(t, p.Name, nameAlias)
	require.Len(t, p.Indexes, 1)

	views, err := db.Views(ctx)
	require.NoError(t, err)
	require.Len(t, views, 1)

	exist, err := db.ViewExists(ctx, nameAlias)
	require.NoError(t, err)
	require.True(t, exist)

	vv, err := db.View(ctx, nameAlias)
	require.NoError(t, err)
	require.Equal(t, vv.Name(), nameAlias)

	err = v.Remove(ctx)
	require.NoError(t, err)

	views, err = db.Views(ctx)
	require.NoError(t, err)
	require.Len(t, views, 0)
}
