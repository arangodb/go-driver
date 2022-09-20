package test

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver"

	"github.com/stretchr/testify/require"
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
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.10", t)
	skipBelowVersion(c, "3.10", t)
	db := ensureDatabase(ctx, c, "search_view_test_basic", nil, t)

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
				{Field: "test1", Ascending: newBool(true)},
				{Field: "test2", Ascending: newBool(false)},
			},
			Compression: driver.PrimarySortCompressionLz4,
		},
		Fields: []driver.InvertedIndexField{
			{Name: "field1", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency}, Nested: nil},
			{Name: "field2", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeaturePosition}, TrackListPositions: false, Nested: nil},
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
