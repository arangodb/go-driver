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
// Author Jakub Wierzbowski
//

package tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
)

func Test_EnsureInvertedIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContext(30*time.Second, func(ctx context.Context) error {

					type testCase struct {
						IsEE    bool
						Options arangodb.InvertedIndexOptions
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
									{Name: "test1", Features: []arangodb.AnalyzerFeature{arangodb.AnalyzerFeatureFrequency}, Nested: nil},
									{Name: "test2", Features: []arangodb.AnalyzerFeature{arangodb.AnalyzerFeatureFrequency, arangodb.AnalyzerFeaturePosition}, Nested: nil},
								},
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
									{Name: "field1", Features: []arangodb.AnalyzerFeature{arangodb.AnalyzerFeatureFrequency}, Nested: nil},
									{Name: "field2", Features: []arangodb.AnalyzerFeature{arangodb.AnalyzerFeatureFrequency, arangodb.AnalyzerFeaturePosition},
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
					}

					for _, tc := range testCases {
						t.Run(tc.Options.Name, func(t *testing.T) {
							if tc.IsEE {
								skipNoEnterprise(client, ctx, t)
							}

							idx, created, err := col.EnsureInvertedIndex(ctx, &tc.Options)
							require.NoError(t, err)
							require.True(t, created)

							requireIdxEquality := func(invertedIdx arangodb.IndexResponse) {
								require.Equal(t, arangodb.InvertedIndexType, idx.Type)
								require.Equal(t, tc.Options.Name, idx.Name)
								require.Equal(t, tc.Options.PrimarySort, idx.InvertedIndex.PrimarySort)
								require.Equal(t, tc.Options.Fields, idx.InvertedIndex.Fields)
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

					return nil
				})
			})
		})
	})
}
