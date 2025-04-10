//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_EnsureInvertedIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					skipBelowVersion(client, ctx, "3.10", t)

					type testCase struct {
						IsEE       bool
						minVersion arangodb.Version
						Options    arangodb.InvertedIndexOptions
					}

					testCases := []testCase{
						{
							IsEE: false,
							Options: arangodb.InvertedIndexOptions{
								Name: "inverted-opt",
								PrimarySort: &arangodb.PrimarySort{
									Fields: []arangodb.PrimarySortEntry{
										{Field: "test1", Ascending: true},
										{Field: "test2", Ascending: false},
									},
									Compression: arangodb.PrimarySortCompressionLz4,
								},
								Fields: []arangodb.InvertedIndexField{
									{
										Name:     "test1",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency},
										Nested:   nil},
									{
										Name:     "test2",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency, arangodb.ArangoSearchFeaturePosition},
										Nested:   nil},
								},
							},
						},
						{
							IsEE: false,
							Options: arangodb.InvertedIndexOptions{
								Name: "inverted-overwrite-tracklistpositions",
								PrimarySort: &arangodb.PrimarySort{
									Fields: []arangodb.PrimarySortEntry{
										{Field: "test1", Ascending: true},
										{Field: "test2", Ascending: false},
									},
									Compression: arangodb.PrimarySortCompressionLz4,
								},
								Fields: []arangodb.InvertedIndexField{
									{
										Name:     "test1-overwrite",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency},
										Nested:   nil, TrackListPositions: true},
									{
										Name:     "test2",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency, arangodb.ArangoSearchFeaturePosition},
										Nested:   nil},
								},
								TrackListPositions: false,
							},
						},
						{
							IsEE: true,
							Options: arangodb.InvertedIndexOptions{
								Name: "inverted-opt-nested",
								PrimarySort: &arangodb.PrimarySort{
									Fields: []arangodb.PrimarySortEntry{
										{Field: "test1", Ascending: true},
										{Field: "test2", Ascending: false},
									},
									Compression: arangodb.PrimarySortCompressionLz4,
								},
								Fields: []arangodb.InvertedIndexField{
									{
										Name:     "field1",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency},
										Nested:   nil},
									{
										Name:     "field2",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency, arangodb.ArangoSearchFeaturePosition},
										Nested: []arangodb.InvertedIndexNestedField{
											{
												Name: "some-nested-field",
												Nested: []arangodb.InvertedIndexNestedField{
													{Name: "test"},
													{Name: "bas", Nested: []arangodb.InvertedIndexNestedField{
														{Name: "a", Features: nil},
													}},
													{Name: "kas", Nested: []arangodb.InvertedIndexNestedField{
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
							minVersion: arangodb.Version("3.12.0"),
							Options: arangodb.InvertedIndexOptions{
								Name: "inverted-opt-optimize-top-k",
								PrimarySort: &arangodb.PrimarySort{
									Fields: []arangodb.PrimarySortEntry{
										{Field: "field1", Ascending: true},
									},
									Compression: arangodb.PrimarySortCompressionLz4,
								},
								Fields: []arangodb.InvertedIndexField{
									{
										Name:     "field1",
										Features: []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency},
									},
								},
								OptimizeTopK: []string{"BM25(@doc) DESC", "TFIDF(@doc) DESC"},
							},
						},
					}

					for _, tc := range testCases {
						t.Run(tc.Options.Name, func(t *testing.T) {
							if tc.IsEE {
								skipNoEnterprise(client, ctx, t)
							}
							if len(tc.minVersion) > 0 {
								skipBelowVersion(client, ctx, tc.minVersion, t)
							}

							idx, created, err := col.EnsureInvertedIndex(ctx, &tc.Options)
							require.NoError(t, err)
							require.True(t, created)

							requireIdxEquality := func(invertedIdx arangodb.IndexResponse) {
								require.Equal(t, arangodb.InvertedIndexType, idx.Type)
								require.Equal(t, tc.Options.Name, idx.Name)
								require.Equal(t, tc.Options.PrimarySort, idx.InvertedIndex.PrimarySort)
								require.Equal(t, tc.Options.Fields, idx.InvertedIndex.Fields)
								require.Equal(t, tc.Options.TrackListPositions, idx.InvertedIndex.TrackListPositions)

								t.Run("optimizeTopK", func(t *testing.T) {
									skipBelowVersion(client, ctx, "3.12.0", t)
									// OptimizeTopK can be nil or []string{} depends on the version, so it better to check length.
									if len(tc.Options.OptimizeTopK) > 0 || len(idx.InvertedIndex.OptimizeTopK) > 0 {
										require.Equal(t, tc.Options.OptimizeTopK, idx.InvertedIndex.OptimizeTopK)
									}
								})
							}
							requireIdxEquality(idx)

							indexes, err := col.Indexes(ctx)
							require.NoError(t, err)
							require.NotNil(t, indexes)
							assert.True(t, slices.ContainsFunc(indexes, func(i arangodb.IndexResponse) bool {
								return i.ID == idx.ID && i.Name == tc.Options.Name
							}))

							err = col.DeleteIndex(ctx, idx.Name)
							require.NoError(t, err)
						})
					}
				})
			})
		})
	})
}
