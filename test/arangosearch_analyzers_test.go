//
// DISCLAIMER
//
// Copyright 2018-2024 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/util"
)

func newInt(v int) *int {
	return &v
}

func newInt64(v int64) *int64 {
	return &v
}

func newUInt64(v uint64) *uint64 {
	return &v
}

func newFloat64(v float64) *float64 {
	return &v
}

func newVersion(s driver.Version) *driver.Version {
	return &s
}

func newString(s string) *string {
	return &s
}

func newArangoSearchNGramStreamType(s driver.ArangoSearchNGramStreamType) *driver.ArangoSearchNGramStreamType {
	return &s
}

func fillPropertiesDefaults(t *testing.T, c driver.Client, props *driver.ArangoSearchAnalyzerProperties) {
	v, err := c.Version(nil)
	require.NoError(t, err)

	if v.Version.CompareTo("3.6") >= 0 {
		if props.StreamType == nil {
			props.StreamType = newArangoSearchNGramStreamType(driver.ArangoSearchNGramStreamBinary)
		}
		if props.StartMarker == nil {
			props.StartMarker = newString("")
		}
		if props.EndMarker == nil {
			props.EndMarker = newString("")
		}
	}
}

func TestArangoSearchAnalyzerEnsureAnalyzer(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_ensure"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	testCases := []struct {
		Name               string
		MinVersion         *driver.Version
		MaxVersion         *driver.Version
		Definition         driver.ArangoSearchAnalyzerDefinition
		ExpectedDefinition *driver.ArangoSearchAnalyzerDefinition
		Found              bool
		HasError           bool
		EnterpriseOnly     bool
	}{
		{
			Name: "create-my-identity",
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-identitfy",
				Type: driver.ArangoSearchAnalyzerTypeIdentity,
			},
		},
		{
			Name: "create-again-my-identity",
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-identitfy",
				Type: driver.ArangoSearchAnalyzerTypeIdentity,
			},
			Found: true,
		},
		{
			Name: "create-again-my-identity-diff-type",
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-identitfy",
				Type: driver.ArangoSearchAnalyzerTypeDelimiter,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Delimiter: "äöü",
				},
			},
			HasError: true,
		},
		{
			Name: "create-my-delimiter",
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-delimiter",
				Type: driver.ArangoSearchAnalyzerTypeDelimiter,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Delimiter: "äöü",
				},
			},
		},
		{
			Name:       "create-my-ngram-3.5",
			MinVersion: newVersion("3.5"),
			MaxVersion: newVersion("3.6"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-ngram",
				Type: driver.ArangoSearchAnalyzerTypeNGram,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Min:              newInt64(1),
					Max:              newInt64(14),
					PreserveOriginal: util.NewType(false),
				},
			},
		},
		{
			Name:       "create-my-ngram-3.6",
			MinVersion: newVersion("3.6"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-ngram",
				Type: driver.ArangoSearchAnalyzerTypeNGram,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Min:              newInt64(1),
					Max:              newInt64(14),
					PreserveOriginal: util.NewType(false),
				},
			},
			ExpectedDefinition: &driver.ArangoSearchAnalyzerDefinition{
				Name: "my-ngram",
				Type: driver.ArangoSearchAnalyzerTypeNGram,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Min:              newInt64(1),
					Max:              newInt64(14),
					PreserveOriginal: util.NewType(false),

					// Check defaults for 3.6
					StartMarker: newString(""),
					EndMarker:   newString(""),
					StreamType:  newArangoSearchNGramStreamType(driver.ArangoSearchNGramStreamBinary),
				},
			},
		},
		{
			Name:       "create-my-ngram-3.6-custom",
			MinVersion: newVersion("3.6"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-ngram-custom",
				Type: driver.ArangoSearchAnalyzerTypeNGram,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Min:              newInt64(1),
					Max:              newInt64(14),
					PreserveOriginal: util.NewType(false),
					StartMarker:      newString("^"),
					EndMarker:        newString("^"),
					StreamType:       newArangoSearchNGramStreamType(driver.ArangoSearchNGramStreamUTF8),
				},
			},
		},
		{
			Name:       "create-pipeline-analyzer",
			MinVersion: newVersion("3.8"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-pipeline",
				Type: driver.ArangoSearchAnalyzerTypePipeline,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Pipeline: []driver.ArangoSearchAnalyzerPipeline{
						{
							Type: driver.ArangoSearchAnalyzerTypeNGram,
							Properties: driver.ArangoSearchAnalyzerProperties{
								Min:              newInt64(1),
								Max:              newInt64(14),
								PreserveOriginal: util.NewType(false),
								StartMarker:      newString("^"),
								EndMarker:        newString("^"),
								StreamType:       newArangoSearchNGramStreamType(driver.ArangoSearchNGramStreamUTF8),
							},
						},
					},
				},
			},
		},
		{
			Name:       "create-aql-analyzer",
			MinVersion: newVersion("3.8"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-aql",
				Type: driver.ArangoSearchAnalyzerTypeAQL,
				Properties: driver.ArangoSearchAnalyzerProperties{
					QueryString:       `FOR year IN [ 2011, 2012, 2013 ] FOR quarter IN [ 1, 2, 3, 4 ] RETURN { year, quarter, formatted: CONCAT(quarter, " / ", year)}`,
					CollapsePositions: util.NewType(true),
					KeepNull:          util.NewType(false),
					BatchSize:         util.NewType(10),
					ReturnType:        driver.ArangoSearchAnalyzerAQLReturnTypeString.New(),
					MemoryLimit:       util.NewType(1024 * 1024),
				},
			},
		},
		{
			Name:       "create-geopoint",
			MinVersion: newVersion("3.8"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-geopoint",
				Type: driver.ArangoSearchAnalyzerTypeGeoPoint,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Options: &driver.ArangoSearchAnalyzerGeoOptions{
						MaxCells: util.NewType(20),
						MinLevel: util.NewType(4),
						MaxLevel: util.NewType(23),
					},
					Latitude:  []string{},
					Longitude: []string{},
				},
			},
		},
		{
			Name:       "create-geojson",
			MinVersion: newVersion("3.8"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-geojson",
				Type: driver.ArangoSearchAnalyzerTypeGeoJSON,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Options: &driver.ArangoSearchAnalyzerGeoOptions{
						MaxCells: util.NewType(20),
						MinLevel: util.NewType(4),
						MaxLevel: util.NewType(23),
					},
					Type: driver.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
		},
		{
			Name:       "create-geo_s2",
			MinVersion: newVersion("3.10.5"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-geo_s2",
				Type: driver.ArangoSearchAnalyzerTypeGeoS2,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Format: driver.FormatLatLngInt.New(),
					Options: &driver.ArangoSearchAnalyzerGeoOptions{
						MaxCells: util.NewType(20),
						MinLevel: util.NewType(4),
						MaxLevel: util.NewType(23),
					},
					Type: driver.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
			ExpectedDefinition: &driver.ArangoSearchAnalyzerDefinition{
				Name: "my-geo_s2",
				Type: driver.ArangoSearchAnalyzerTypeGeoS2,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Format: driver.FormatLatLngInt.New(),
					Options: &driver.ArangoSearchAnalyzerGeoOptions{
						MaxCells: util.NewType(20),
						MinLevel: util.NewType(4),
						MaxLevel: util.NewType(23),
					},
					Type: driver.ArangoSearchAnalyzerGeoJSONTypeShape.New(),
				},
			},
			EnterpriseOnly: true,
		},
		{
			Name:       "create-segmentation",
			MinVersion: newVersion("3.9"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-segmentation",
				Type: driver.ArangoSearchAnalyzerTypeSegmentation,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Break: driver.ArangoSearchBreakTypeAll,
					Case:  driver.ArangoSearchCaseUpper,
				},
			},
		},
		{
			Name:       "create-collation",
			MinVersion: newVersion("3.9"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-collation",
				Type: driver.ArangoSearchAnalyzerTypeCollation,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Locale: "en_US.utf-8",
				},
			},
			ExpectedDefinition: &driver.ArangoSearchAnalyzerDefinition{
				Name: "my-collation",
				Type: driver.ArangoSearchAnalyzerTypeCollation,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Locale: "en_US",
				},
			},
		},
		{
			Name:       "create-stopWords",
			MinVersion: newVersion("3.9"),
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-stopWords",
				Type: driver.ArangoSearchAnalyzerTypeStopwords,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Hex: util.NewType(true),
					Stopwords: []string{
						"616e64",
						"746865",
					},
				},
			},
			ExpectedDefinition: &driver.ArangoSearchAnalyzerDefinition{
				Name: "my-stopWords",
				Type: driver.ArangoSearchAnalyzerTypeStopwords,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Hex: util.NewType(true),
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
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-minhash",
				Type: driver.ArangoSearchAnalyzerTypeMinhash,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Analyzer: &driver.ArangoSearchAnalyzerDefinition{
						Type: driver.ArangoSearchAnalyzerTypeStopwords,
						Properties: driver.ArangoSearchAnalyzerProperties{
							Hex: util.NewType(true),
							Stopwords: []string{
								"616e64",
								"746865",
							},
						},
					},
					NumHashes: newUInt64(2),
				},
			},
			ExpectedDefinition: &driver.ArangoSearchAnalyzerDefinition{
				Name: "my-minhash",
				Type: driver.ArangoSearchAnalyzerTypeMinhash,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Analyzer: &driver.ArangoSearchAnalyzerDefinition{
						Type: driver.ArangoSearchAnalyzerTypeStopwords,
						Properties: driver.ArangoSearchAnalyzerProperties{
							Hex: util.NewType(true),
							Stopwords: []string{
								"616e64",
								"746865",
							},
						},
					},
					NumHashes: newUInt64(2),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.MinVersion != nil {
				if testCase.MaxVersion == nil {
					skipBelowVersion(c, *testCase.MinVersion, t)
				} else {
					skipVersionNotInRange(c, *testCase.MinVersion, *testCase.MaxVersion, t)
				}
			}
			if testCase.EnterpriseOnly {
				skipNoEnterprise(t)
			}

			existed, a, err := db.EnsureAnalyzer(ctx, testCase.Definition)

			if testCase.HasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.Found, existed)
			if a != nil {
				var def driver.ArangoSearchAnalyzerDefinition
				if testCase.ExpectedDefinition != nil {
					def = *testCase.ExpectedDefinition
				} else {
					def = testCase.Definition
				}

				require.Equal(t, a.Name(), def.Name)
				require.Equal(t, a.Type(), def.Type)
				require.Equal(t, a.UniqueName(), dbname+"::"+def.Name)
				require.Equal(t, a.Database(), db)
				aSerialized, err := json.Marshal(a.Properties())
				require.NoError(t, err)
				defSerialized, err := json.Marshal(def.Properties)
				require.NoError(t, err)
				require.Equalf(t, a.Properties(), def.Properties, "expected %s, got %s", string(aSerialized), string(defSerialized))
			}
		})
	}
}

func ensureAnalyzer(ctx context.Context, db driver.Database, definition driver.ArangoSearchAnalyzerDefinition, t *testing.T) driver.ArangoSearchAnalyzer {
	_, a, err := db.EnsureAnalyzer(ctx, definition)
	require.NoError(t, err)
	return a
}

func TestArangoSearchAnalyzerGet(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: util.NewType(false),
		},
	}
	ensureAnalyzer(ctx, db, def, t)
	fillPropertiesDefaults(t, c, &def.Properties)

	a, err := db.Analyzer(ctx, aname)

	require.NoError(t, err)
	require.NotNil(t, a)
	require.Equal(t, a.Name(), def.Name)
	require.Equal(t, a.Type(), def.Type)
	require.Equal(t, a.UniqueName(), dbname+"::"+def.Name)
	require.Equal(t, a.Database(), db)
	require.Equal(t, a.Properties(), def.Properties)

	_, err = db.Analyzer(ctx, "does-not-exist")
	require.Error(t, err)
}

func TestArangoSearchAnalyzerGetAll(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get_all"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: util.NewType(false),
		},
	}
	ensureAnalyzer(ctx, db, def, t)
	fillPropertiesDefaults(t, c, &def.Properties)

	alist, err := db.Analyzers(ctx)
	require.NoError(t, err)
	require.NotNil(t, alist)
	require.NotEmpty(t, alist)

	found := false
	for _, a := range alist {
		if a.Name() == aname {
			require.Equal(t, a.Name(), def.Name)
			require.Equal(t, a.Type(), def.Type)
			require.Equal(t, a.UniqueName(), dbname+"::"+def.Name)
			require.Equal(t, a.Database(), db)
			require.Equal(t, a.Properties(), def.Properties)
			found = true
		}
	}

	require.True(t, found)
}

func TestArangoSearchAnalyzerRemove(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get_all"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: util.NewType(false),
		},
	}
	a := ensureAnalyzer(ctx, db, def, t)
	err := a.Remove(ctx, false)
	require.NoError(t, err)

	alist, err := db.Analyzers(ctx)
	require.NoError(t, err)
	require.NotNil(t, alist)
	require.NotEmpty(t, alist)

	// should not be found
	found := false
	for _, a := range alist {
		if a.Name() == aname {
			found = true
		}
	}

	require.False(t, found)
}
