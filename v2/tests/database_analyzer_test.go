//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_Analyzers(t *testing.T) {
	testCases := []struct {
		Name               string
		MinVersion         *arangodb.Version
		Definition         arangodb.AnalyzerDefinition
		ExpectedDefinition *arangodb.AnalyzerDefinition
		Found              bool
		HasError           bool
		EnterpriseOnly     bool
	}{
		{
			Name: "create-my-identity",
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-identitfy",
				Type: arangodb.ArangoSearchAnalyzerTypeIdentity,
			},
		},
		{
			Name: "create-again-my-identity",
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-identitfy",
				Type: arangodb.ArangoSearchAnalyzerTypeIdentity,
			},
			Found: true,
		},
		{
			Name: "create-again-my-identity-diff-type",
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-identitfy",
				Type: arangodb.ArangoSearchAnalyzerTypeDelimiter,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Delimiter: "äöü",
				},
			},
			HasError: true,
		},
		{
			Name:       "create-my-multi-delimiters",
			MinVersion: newVersion("3.12"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-multidelimiters",
				Type: arangodb.ArangoSearchAnalyzerTypeMultiDelimiter,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Delimiters: []string{"ö", "ü"},
				},
			},
			HasError: false,
		},
		{
			Name: "create-my-delimiter",
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-delimiter",
				Type: arangodb.ArangoSearchAnalyzerTypeDelimiter,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Delimiter: "äöü",
				},
			},
		},
		{
			Name:       "create-my-ngram-3.6",
			MinVersion: newVersion("3.6"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-ngram",
				Type: arangodb.ArangoSearchAnalyzerTypeNGram,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Min:              utils.NewType[int64](1),
					Max:              utils.NewType[int64](14),
					PreserveOriginal: utils.NewType(false),
				},
			},
			ExpectedDefinition: &arangodb.AnalyzerDefinition{
				Name: "my-ngram",
				Type: arangodb.ArangoSearchAnalyzerTypeNGram,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Min:              utils.NewType[int64](1),
					Max:              utils.NewType[int64](14),
					PreserveOriginal: utils.NewType(false),

					// Check defaults for 3.6
					StartMarker: utils.NewType(""),
					EndMarker:   utils.NewType(""),
					StreamType:  utils.NewType(arangodb.ArangoSearchNGramStreamBinary),
				},
			},
		},
		{
			Name:       "create-my-ngram-3.6-custom",
			MinVersion: newVersion("3.6"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-ngram-custom",
				Type: arangodb.ArangoSearchAnalyzerTypeNGram,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Min:              utils.NewType[int64](1),
					Max:              utils.NewType[int64](14),
					PreserveOriginal: utils.NewType(false),
					StartMarker:      utils.NewType("^"),
					EndMarker:        utils.NewType("^"),
					StreamType:       utils.NewType(arangodb.ArangoSearchNGramStreamUTF8),
				},
			},
		},
		{
			Name:       "create-pipeline-analyzer",
			MinVersion: newVersion("3.8"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-pipeline",
				Type: arangodb.ArangoSearchAnalyzerTypePipeline,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Pipeline: []arangodb.ArangoSearchAnalyzerPipeline{
						{
							Type: arangodb.ArangoSearchAnalyzerTypeNGram,
							Properties: arangodb.ArangoSearchAnalyzerProperties{
								Min:              utils.NewType[int64](1),
								Max:              utils.NewType[int64](14),
								PreserveOriginal: utils.NewType(false),
								StartMarker:      utils.NewType("^"),
								EndMarker:        utils.NewType("^"),
								StreamType:       utils.NewType(arangodb.ArangoSearchNGramStreamUTF8),
							},
						},
					},
				},
			},
		},
		{
			Name:       "create-aql-analyzer",
			MinVersion: newVersion("3.8"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-aql",
				Type: arangodb.ArangoSearchAnalyzerTypeAQL,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					QueryString:       `FOR year IN [ 2011, 2012, 2013 ] FOR quarter IN [ 1, 2, 3, 4 ] RETURN { year, quarter, formatted: CONCAT(quarter, " / ", year)}`,
					CollapsePositions: utils.NewType(true),
					KeepNull:          utils.NewType(false),
					BatchSize:         utils.NewType(10),
					ReturnType:        arangodb.ArangoSearchAnalyzerAQLReturnTypeString.New(),
					MemoryLimit:       utils.NewType(1024 * 1024),
				},
			},
		},
		{
			Name:       "create-geopoint",
			MinVersion: newVersion("3.8"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-geopoint",
				Type: arangodb.ArangoSearchAnalyzerTypeGeoPoint,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Options: &arangodb.ArangoSearchAnalyzerGeoOptions{
						MaxCells: utils.NewType(20),
						MinLevel: utils.NewType(4),
						MaxLevel: utils.NewType(23),
					},
					Latitude:  []string{},
					Longitude: []string{},
				},
			},
		},
		{
			Name:       "create-geojson",
			MinVersion: newVersion("3.8"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-geojson",
				Type: arangodb.ArangoSearchAnalyzerTypeGeoJSON,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Options: &arangodb.ArangoSearchAnalyzerGeoOptions{
						MaxCells: utils.NewType(20),
						MinLevel: utils.NewType(4),
						MaxLevel: utils.NewType(23),
					},
					Type: arangodb.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
		},
		{
			Name:       "create-geo_s2",
			MinVersion: newVersion("3.10.5"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-geo_s2",
				Type: arangodb.ArangoSearchAnalyzerTypeGeoS2,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Format: utils.NewType(arangodb.ArangoSearchFormatLatLngInt),
					Options: &arangodb.ArangoSearchAnalyzerGeoOptions{
						MaxCells: utils.NewType(20),
						MinLevel: utils.NewType(4),
						MaxLevel: utils.NewType(23),
					},
					Type: arangodb.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
			ExpectedDefinition: &arangodb.AnalyzerDefinition{
				Name: "my-geo_s2",
				Type: arangodb.ArangoSearchAnalyzerTypeGeoS2,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Format: utils.NewType(arangodb.ArangoSearchFormatLatLngInt),
					Options: &arangodb.ArangoSearchAnalyzerGeoOptions{
						MaxCells: utils.NewType(20),
						MinLevel: utils.NewType(4),
						MaxLevel: utils.NewType(23),
					},
					Type: arangodb.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
			EnterpriseOnly: true,
		},
		{
			Name:       "create-segmentation",
			MinVersion: newVersion("3.9"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-segmentation",
				Type: arangodb.ArangoSearchAnalyzerTypeSegmentation,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Break: arangodb.ArangoSearchBreakTypeAll,
					Case:  arangodb.ArangoSearchCaseUpper,
				},
			},
		},
		{
			Name:       "create-collation",
			MinVersion: newVersion("3.9"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-collation",
				Type: arangodb.ArangoSearchAnalyzerTypeCollation,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Locale: "en_US.utf-8",
				},
			},
			ExpectedDefinition: &arangodb.AnalyzerDefinition{
				Name: "my-collation",
				Type: arangodb.ArangoSearchAnalyzerTypeCollation,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Locale: "en_US",
				},
			},
		},
		{
			Name:       "create-stopWords",
			MinVersion: newVersion("3.9"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-stopWords",
				Type: arangodb.ArangoSearchAnalyzerTypeStopwords,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Hex: utils.NewType(true),
					Stopwords: []string{
						"616e64",
						"746865",
					},
				},
			},
			ExpectedDefinition: &arangodb.AnalyzerDefinition{
				Name: "my-stopWords",
				Type: arangodb.ArangoSearchAnalyzerTypeStopwords,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Hex: utils.NewType(true),
					Stopwords: []string{
						"616e64",
						"746865",
					},
				},
			},
		},
		{
			Name:           "my-minhash",
			MinVersion:     newVersion("3.10"),
			EnterpriseOnly: true,
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-minhash",
				Type: arangodb.ArangoSearchAnalyzerTypeMinhash,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Analyzer: &arangodb.AnalyzerDefinition{
						Type: arangodb.ArangoSearchAnalyzerTypeStopwords,
						Properties: arangodb.ArangoSearchAnalyzerProperties{
							Hex: utils.NewType(true),
							Stopwords: []string{
								"616e64",
								"746865",
							},
						},
					},
					NumHashes: utils.NewType[uint64](2),
				},
			},
			ExpectedDefinition: &arangodb.AnalyzerDefinition{
				Name: "my-minhash",
				Type: arangodb.ArangoSearchAnalyzerTypeMinhash,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					Analyzer: &arangodb.AnalyzerDefinition{
						Type: arangodb.ArangoSearchAnalyzerTypeStopwords,
						Properties: arangodb.ArangoSearchAnalyzerProperties{
							Hex: utils.NewType(true),
							Stopwords: []string{
								"616e64",
								"746865",
							},
						},
					},
					NumHashes: utils.NewType[uint64](2),
				},
			},
		},
		{
			Name:       "create-my-wildcard",
			MinVersion: newVersion("3.12"),
			Definition: arangodb.AnalyzerDefinition{
				Name: "my-wildcard",
				Type: arangodb.ArangoSearchAnalyzerTypeWildcard,
				Properties: arangodb.ArangoSearchAnalyzerProperties{
					NGramSize: 4,
				},
			},
			HasError: false,
		},
	}

	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			nonSuchAnalyzer, err := db.Analyzer(ctx, "no_such_analyzer")
			require.Error(t, err)
			require.True(t, shared.IsNotFound(err))
			require.Nil(t, nonSuchAnalyzer)

			for _, testCase := range testCases {
				t.Run(testCase.Name, func(t *testing.T) {
					if testCase.MinVersion != nil {
						skipBelowVersion(client, ctx, *testCase.MinVersion, t)
					}
					if testCase.EnterpriseOnly {
						skipNoEnterprise(client, ctx, t)
					}

					existed, ensuredA, err := db.EnsureAnalyzer(ctx, &testCase.Definition)

					if testCase.HasError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}

					require.Equal(t, testCase.Found, existed)
					if ensuredA != nil {
						var def arangodb.AnalyzerDefinition
						if testCase.ExpectedDefinition != nil {
							def = *testCase.ExpectedDefinition
						} else {
							def = testCase.Definition
						}

						checkAnalyzer(t, db, def, ensuredA)

						// try to find the same analyzer via reading all analyzers
						list := readAllAnalyzersT(ctx, t, db)
						found := false
						for _, listedA := range list {
							if listedA.UniqueName() == ensuredA.UniqueName() {
								found = true
								checkAnalyzer(t, db, def, listedA)
							}
						}
						require.True(t, found)

						// try to find the same analyzer by normal GET
						gotA, err := db.Analyzer(ctx, ensuredA.Name())
						require.NoError(t, err)
						require.NotNil(t, gotA)
						checkAnalyzer(t, db, def, gotA)
					}
				})
			}
		})
	})
}

func Test_AnalyzerRemove(t *testing.T) {
	def := arangodb.AnalyzerDefinition{
		Name: "my-delimiter",
		Type: arangodb.ArangoSearchAnalyzerTypeDelimiter,
		Properties: arangodb.ArangoSearchAnalyzerProperties{
			Delimiter: "äöü",
		},
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			ctx := context.Background()

			_, a, err := db.EnsureAnalyzer(ctx, &def)
			require.NoError(t, err)

			// delete and check it was deleted (use force to delete it even if it is in use)
			err = a.Remove(ctx, true)
			require.NoError(t, err)

			shouldBeRemoved, err := db.Analyzer(ctx, a.Name())
			require.Error(t, err)
			require.True(t, shared.IsNotFound(err))
			require.Nil(t, shouldBeRemoved)
		})
	})
}

func readAllAnalyzersT(ctx context.Context, t *testing.T, db arangodb.Database) []arangodb.Analyzer {
	t.Helper()

	r, err := db.Analyzers(ctx)
	require.NoError(t, err)

	result := make([]arangodb.Analyzer, 0)
	for {
		a, err := r.Read()
		if shared.IsNoMoreDocuments(err) {
			return result
		}
		require.NoError(t, err)
		result = append(result, a)
	}
}

func checkAnalyzer(t *testing.T, db arangodb.Database, def arangodb.AnalyzerDefinition, actual arangodb.Analyzer) {
	t.Helper()

	require.Equal(t, def.Name, actual.Name())
	require.Equal(t, def.Type, actual.Type())
	require.Equal(t, db.Name()+"::"+def.Name, actual.UniqueName())
	require.Equal(t, db, actual.Database())
	actualSerialized, err := json.Marshal(actual.Definition().Properties)
	require.NoError(t, err)
	defSerialized, err := json.Marshal(def.Properties)
	require.NoError(t, err)
	require.Equalf(t, def.Properties, actual.Definition().Properties, "expected %s, got %s", string(defSerialized), string(actualSerialized))
}
