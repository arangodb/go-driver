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
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

func newInt64(v int64) *int64 {
	return &v
}

func TestArangoSearchAnalyzerEnsureAnalyzer(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_ensure"
	db := ensureDatabase(ctx, c, dbname, nil, t)

	testCases := []struct {
		Name       string
		Definition driver.ArangoSearchAnalyzerDefinition
		Found      bool
		HasError   bool
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
			Name: "create-my-ngram",
			Definition: driver.ArangoSearchAnalyzerDefinition{
				Name: "my-ngram",
				Type: driver.ArangoSearchAnalyzerTypeNGram,
				Properties: driver.ArangoSearchAnalyzerProperties{
					Min:              newInt64(1),
					Max:              newInt64(14),
					PreserveOriginal: newBool(false),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			existed, a, err := db.EnsureAnalyzer(ctx, testCase.Definition)

			if testCase.HasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.Found, existed)
			if a != nil {
				require.Equal(t, a.Name(), testCase.Definition.Name)
				require.Equal(t, a.Type(), testCase.Definition.Type)
				require.Equal(t, a.UniqueName(), dbname+"::"+testCase.Definition.Name)
				require.Equal(t, a.Database(), db)
				require.Equal(t, a.Properties(), testCase.Definition.Properties)
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
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: newBool(false),
		},
	}
	ensureAnalyzer(ctx, db, def, t)

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
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get_all"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: newBool(false),
		},
	}
	ensureAnalyzer(ctx, db, def, t)

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
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.5", t)
	ctx := context.Background()

	dbname := "analyzer_test_get_all"
	db := ensureDatabase(ctx, c, dbname, nil, t)
	aname := "my-ngram"
	def := driver.ArangoSearchAnalyzerDefinition{
		Name: aname,
		Type: driver.ArangoSearchAnalyzerTypeNGram,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Min:              newInt64(1),
			Max:              newInt64(14),
			PreserveOriginal: newBool(false),
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
