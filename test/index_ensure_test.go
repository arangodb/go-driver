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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

// TestEnsureFullTextIndex creates a collection with a full text index.
func TestEnsureFullTextIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureFullTextIndexOptions{
		nil,
		{MinLength: 2},
		{MinLength: 20},
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
		{GeoJSON: true},
		{GeoJSON: false},
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

// TestEnsureGeoIndexLegacyPolygons creates a collection with a Geo index and additional LegacyPolygons options.
func TestEnsureGeoIndexLegacyPolygons(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.10", t)

	db := ensureDatabase(ctx, c, "index_geo_LegacyPolygons_test", nil, t)
	col := ensureCollection(ctx, db, fmt.Sprintf("persistent_index_options_test_"), nil, t)

	options := &driver.EnsureGeoIndexOptions{
		LegacyPolygons: true,
	}
	idx, created, err := col.EnsureGeoIndex(ctx, []string{"age"}, options)
	if err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}
	require.True(t, created)
	require.Equal(t, driver.GeoIndex, idx.Type())
	require.True(t, idx.LegacyPolygons())

	idxDefault, created, err := col.EnsureGeoIndex(ctx, []string{"name"}, nil)
	if err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}
	require.True(t, created)
	require.Equal(t, driver.GeoIndex, idx.Type())
	require.False(t, idxDefault.LegacyPolygons())
}

// TestEnsureHashIndex creates a collection with a hash index.
func TestEnsureHashIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureHashIndexOptions{
		nil,
		{Unique: true, Sparse: false},
		{Unique: true, Sparse: true},
		{Unique: false, Sparse: false},
		{Unique: false, Sparse: true},
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
		{Unique: true, Sparse: false},
		{Unique: true, Sparse: true},
		{Unique: false, Sparse: false},
		{Unique: false, Sparse: true},
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

		// Index must exist now
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

		// Index must not exist now
		if found, err := col.IndexExists(nil, idx.Name()); err != nil {
			t.Fatalf("Failed to check index '%s' exists: %s", idx.Name(), describe(err))
		} else if found {
			t.Errorf("Index '%s' does exist, expected it not to exist", idx.Name())
		}
	}
}

// TestEnsurePersistentIndexOptions creates a collection with a persistent index and additional options.
func TestEnsurePersistentIndexOptions(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.10", t)

	db := ensureDatabase(ctx, c, "index_persistent_options_test", nil, t)
	col := ensureCollection(ctx, db, fmt.Sprintf("persistent_index_options_test_"), nil, t)

	options := &driver.EnsurePersistentIndexOptions{
		StoredValues: []string{"extra1", "extra2"},
		CacheEnabled: true,
	}
	idx, created, err := col.EnsurePersistentIndex(ctx, []string{"age", "name"}, options)
	if err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}

	require.True(t, created)
	require.Equal(t, driver.PersistentIndex, idx.Type())

	require.NotNil(t, idx.StoredValues())
	require.Len(t, idx.StoredValues(), 2)
	require.Equal(t, "extra1", idx.StoredValues()[0])
	require.Equal(t, "extra2", idx.StoredValues()[1])

	require.True(t, idx.CacheEnabled())
}

// TestEnsureSkipListIndex creates a collection with a skiplist index.
func TestEnsureSkipListIndex(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)

	testOptions := []*driver.EnsureSkipListIndexOptions{
		nil,
		{Unique: true, Sparse: false, NoDeduplicate: true},
		{Unique: true, Sparse: true, NoDeduplicate: true},
		{Unique: false, Sparse: false, NoDeduplicate: false},
		{Unique: false, Sparse: true, NoDeduplicate: false},
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

	// Create index with expireAfter = 0
	idx, created, err = col.EnsureTTLIndex(nil, "createdAt", 0, nil)
	if err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}
	if !created {
		t.Error("Expected created to be true, got false")
	}
	if idxType := idx.Type(); idxType != driver.TTLIndex {
		t.Errorf("Expected TTLIndex, found `%s`", idxType)
	}
	if idx.ExpireAfter() != 0 {
		t.Errorf("Expected ExpireAfter to be 0, found `%d`", idx.ExpireAfter())
	}
}

// TestEnsureZKDIndex creates a collection with a ZKD index.
func TestEnsureZKDIndex(t *testing.T) {
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.9.0"))

	db := ensureDatabase(ctx, c, "index_test", nil, t)
	col := ensureCollection(ctx, db, fmt.Sprintf("zkd_index_test"), nil, t)

	f1 := "field-zkd-index_1"
	f2 := "field-zkd-index_2"

	idx, created, err := col.EnsureZKDIndex(ctx, []string{f1, f2}, nil)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, driver.ZKDIndex, idx.Type())
	assert.Contains(t, idx.Fields(), f1)
	assert.Contains(t, idx.Fields(), f2)

	err = idx.Remove(nil)
	require.NoError(t, err)
}

// TestEnsureZKDIndexWithOptions creates a collection with a ZKD index and additional options
func TestEnsureZKDIndexWithOptions(t *testing.T) {
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.9.0"))

	db := ensureDatabase(ctx, c, "index_test", nil, t)
	col := ensureCollection(ctx, db, fmt.Sprintf("zkd_index_opt_test"), nil, t)

	f1 := "field-zkd-index1-opt"
	f2 := "field-zkd-index2-opt"

	opt := driver.EnsureZKDIndexOptions{
		Name: "zkd-opt",
	}

	idx, created, err := col.EnsureZKDIndex(ctx, []string{f1, f2}, &opt)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, driver.ZKDIndex, idx.Type())
	require.Equal(t, opt.Name, idx.UserName())
	assert.Contains(t, idx.Fields(), f1)
	assert.Contains(t, idx.Fields(), f2)

	err = idx.Remove(nil)
	require.NoError(t, err)
}

// TestEnsureInvertedIndex creates a collection with an inverted index
func TestEnsureInvertedIndex(t *testing.T) {
	ctx := context.Background()

	c := createClientFromEnv(t, true)
	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.10.0"))

	db := ensureDatabase(ctx, c, "index_test", nil, t)
	col := ensureCollection(ctx, db, fmt.Sprintf("inverted_index_opt_test"), nil, t)

	type testCase struct {
		IsEE       bool
		minVersion driver.Version
		Options    driver.InvertedIndexOptions
	}
	testCases := []testCase{
		{
			IsEE: false,
			Options: driver.InvertedIndexOptions{
				Name: "inverted-opt",
				PrimarySort: driver.InvertedIndexPrimarySort{
					Fields: []driver.ArangoSearchPrimarySortEntry{
						{Field: "test1", Ascending: newBool(true)},
						{Field: "test2", Ascending: newBool(false)},
					},
					Compression: driver.PrimarySortCompressionLz4,
				},
				Fields: []driver.InvertedIndexField{
					{Name: "test1", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency}, Nested: nil},
					{Name: "test2", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency, driver.ArangoSearchAnalyzerFeaturePosition}, TrackListPositions: false, Nested: nil},
				},
			},
		},
		{
			IsEE: true,
			Options: driver.InvertedIndexOptions{
				Name: "inverted-opt-nested",
				PrimarySort: driver.InvertedIndexPrimarySort{
					Fields: []driver.ArangoSearchPrimarySortEntry{
						{Field: "test1", Ascending: newBool(true)},
						{Field: "test2", Ascending: newBool(false)},
					},
					Compression: driver.PrimarySortCompressionLz4,
				},
				Fields: []driver.InvertedIndexField{
					{Name: "field1", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency}, Nested: nil},
					{Name: "field2", Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency, driver.ArangoSearchAnalyzerFeaturePosition}, TrackListPositions: false,
						Nested: []driver.InvertedIndexField{
							{
								Name: "some-nested-field",
								Nested: []driver.InvertedIndexField{
									{Name: "test"},
									{Name: "bas", Nested: []driver.InvertedIndexField{
										{Name: "a", Features: nil},
									}},
									{Name: "kas", Nested: []driver.InvertedIndexField{
										{Name: "b", TrackListPositions: true},
										{Name: "c"},
									}},
								},
							},
						},
					},
				},
			},
		},
		{
			IsEE:       true,
			minVersion: driver.Version("3.11.0"),
			Options: driver.InvertedIndexOptions{
				Name: "inverted-opt-optimize-top-k",
				PrimarySort: driver.InvertedIndexPrimarySort{
					Fields: []driver.ArangoSearchPrimarySortEntry{
						{Field: "field1", Ascending: newBool(true)},
					},
					Compression: driver.PrimarySortCompressionLz4,
				},
				Fields: []driver.InvertedIndexField{
					{
						Name:     "field1",
						Features: []driver.ArangoSearchAnalyzerFeature{driver.ArangoSearchAnalyzerFeatureFrequency},
					},
				},
				OptimizeTopK: []string{"BM25(@doc) DESC", "TFIDF(@doc) DESC"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Options.Name, func(t *testing.T) {
			if tc.IsEE {
				skipNoEnterprise(t)
			}
			if len(tc.minVersion) > 0 {
				skipBelowVersion(c, tc.minVersion, t)
			}

			requireIdxEquality := func(invertedIdx driver.Index) {
				require.Equal(t, driver.InvertedIndex, invertedIdx.Type())
				require.Equal(t, tc.Options.Name, invertedIdx.UserName())
				require.Equal(t, tc.Options.PrimarySort, invertedIdx.InvertedIndexOptions().PrimarySort)
				require.Equal(t, tc.Options.Fields, invertedIdx.InvertedIndexOptions().Fields)

				t.Run("optimizeTopK", func(t *testing.T) {
					skipBelowVersion(c, "3.11.0", t)
					// OptimizeTopK can be nil or []string{} depends on the version, so it better to check length.
					if len(tc.Options.OptimizeTopK) > 0 || len(invertedIdx.InvertedIndexOptions().OptimizeTopK) > 0 {
						require.Equal(t, tc.Options.OptimizeTopK, invertedIdx.InvertedIndexOptions().OptimizeTopK)
					}
				})
			}

			idx, created, err := col.EnsureInvertedIndex(ctx, &tc.Options)
			require.NoError(t, err)
			require.True(t, created)
			requireIdxEquality(idx)

			col.Indexes(ctx)
			idx, err = col.Index(ctx, tc.Options.Name)
			require.NoError(t, err)
			requireIdxEquality(idx)

			err = idx.Remove(ctx)
			require.NoError(t, err)
		})
	}
}
