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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

type validateQueryTest struct {
	Query         string
	ExpectSuccess bool
}

type profileQueryTest struct {
	Query    string
	BindVars map[string]interface{}
}

func prepareQueryDatabase(t *testing.T, ctx context.Context, c driver.Client, name string) (driver.Database, func(t *testing.T)) {
	db := ensureDatabase(ctx, c, name, nil, t)

	// Create data set
	collectionData := map[string][]interface{}{
		"books": {
			Book{Title: "Book 01"},
			Book{Title: "Book 02"},
			Book{Title: "Book 03"},
			Book{Title: "Book 04"},
			Book{Title: "Book 05"},
			Book{Title: "Book 06"},
			Book{Title: "Book 07"},
			Book{Title: "Book 08"},
			Book{Title: "Book 09"},
			Book{Title: "Book 10"},
			Book{Title: "Book 11"},
			Book{Title: "Book 12"},
			Book{Title: "Book 13"},
			Book{Title: "Book 14"},
			Book{Title: "Book 15"},
			Book{Title: "Book 16"},
			Book{Title: "Book 17"},
			Book{Title: "Book 18"},
			Book{Title: "Book 19"},
			Book{Title: "Book 20"},
		},
		"users": {
			UserDoc{Name: "John", Age: 13},
			UserDoc{Name: "Jake", Age: 25},
			UserDoc{Name: "Clair", Age: 12},
			UserDoc{Name: "Johnny", Age: 42},
			UserDoc{Name: "Blair", Age: 67},
		},
	}
	for colName, colDocs := range collectionData {
		col := ensureCollection(ctx, db, colName, nil, t)
		if _, _, err := col.CreateDocuments(ctx, colDocs); err != nil {
			require.NoError(t, err)
		}
	}

	return db, func(t *testing.T) {
		require.NoError(t, db.Remove(ctx))
	}
}

// TestValidateQuery validates several AQL queries.
func TestValidateQuery(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "validate_query_test", nil, t)

	db, clean := prepareQueryDatabase(t, ctx, c, "validate_query_test")
	defer clean(t)

	// Setup tests
	tests := []validateQueryTest{
		{
			Query:         "FOR d IN books SORT d.Title RETURN d",
			ExpectSuccess: true,
		},
		{
			Query:         "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			ExpectSuccess: true,
		},
		{
			Query:         "FOR u IN users FILTER u.age>>>100 SORT u.name RETURN u",
			ExpectSuccess: false,
		},
		{
			Query:         "",
			ExpectSuccess: false,
		},
	}

	// Run tests for every context alternative
	for i, test := range tests {
		t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
			err := db.ValidateQuery(ctx, test.Query)
			if test.ExpectSuccess {
				if err != nil {
					t.Errorf("Expected success in query %d (%s), got '%s'", i, test.Query, describe(err))
					return
				}
			} else {
				if err == nil {
					t.Errorf("Expected error in query %d (%s), got '%s'", i, test.Query, describe(err))
					return
				}
			}
		})
	}
}

// TestValidateQuery validates several AQL queries.
func TestValidateQueryOptionShardIds(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	_, err := c.Cluster(ctx)

	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		db := ensureDatabase(ctx, c, "validate_query_options_test", nil, t)
		col := ensureCollection(ctx, db, "c", nil, t)

		db, clean := prepareQueryDatabase(t, ctx, c, "validate_query_options_test")
		defer clean(t)

		t.Run(fmt.Sprintf("Real shards"), func(t *testing.T) {
			shards, err := col.Shards(ctx, true)
			for sk := range shards.Shards {
				ctx = driver.WithQueryShardIds(nil, []string{string(sk)})
				_, err = db.Query(ctx, "FOR doc in c RETURN doc", map[string]interface{}{})
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("Fake shards"), func(t *testing.T) {
			ctx = driver.WithQueryShardIds(nil, []string{"s1"})
			_, err = db.Query(ctx, "FOR doc in c RETURN doc", map[string]interface{}{})
			require.NotNil(t, err)
		})
	}

	return

}

// TestProfileQuery profile several AQL queries.
func TestProfileQuery(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "validate_query_test", nil, t)

	db, clean := prepareQueryDatabase(t, ctx, c, "validate_query_test")
	defer clean(t)

	// Setup tests
	tests := []profileQueryTest{
		{
			Query: "FOR d IN books SORT d.Title RETURN d",
		},
		{
			Query: "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars: map[string]interface{}{
				"title": "Book 16",
			},
		},
	}

	t.Run("Without profile", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				r, err := db.Query(ctx, test.Query, test.BindVars)
				require.NoError(t, err)

				_, ok, err := r.Extra().GetPlanRaw()
				require.NoError(t, err)
				require.False(t, ok)

				_, ok, err = r.Extra().GetProfileRaw()
				require.NoError(t, err)
				require.False(t, ok)
			})
		}
	})

	t.Run("Without profile set to default", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				newCtx := driver.WithQueryProfile(ctx)
				r, err := db.Query(newCtx, test.Query, test.BindVars)
				require.NoError(t, err)

				_, ok, err := r.Extra().GetPlanRaw()
				require.NoError(t, err)
				require.False(t, ok)

				_, ok, err = r.Extra().GetProfileRaw()
				require.NoError(t, err)
				require.True(t, ok)
			})
		}
	})

	t.Run("Without profile set to 1", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				newCtx := driver.WithQueryProfile(ctx, 1)
				r, err := db.Query(newCtx, test.Query, test.BindVars)
				require.NoError(t, err)

				_, ok, err := r.Extra().GetPlanRaw()
				require.NoError(t, err)
				require.False(t, ok)

				_, ok, err = r.Extra().GetProfileRaw()
				require.NoError(t, err)
				require.True(t, ok)
			})
		}
	})

	t.Run("Without profile set to 2", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				newCtx := driver.WithQueryProfile(ctx, 2)
				r, err := db.Query(newCtx, test.Query, test.BindVars)
				require.NoError(t, err)

				_, ok, err := r.Extra().GetPlanRaw()
				require.NoError(t, err)
				require.True(t, ok)

				_, ok, err = r.Extra().GetProfileRaw()
				require.NoError(t, err)
				require.True(t, ok)
			})
		}
	})
}

// TestForceOneShardAttributeValue test ForceOneShardAttributeValue query attribute.
func TestForceOneShardAttributeValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := createClientFromEnv(t, true)

	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.9.0")).Cluster().Enterprise()

	db := ensureDatabase(ctx, c, "force_one_shard_attribute_value", nil, t)

	db, clean := prepareQueryDatabase(t, ctx, c, "force_one_shard_attribute_value")
	defer clean(t)

	// Setup tests
	tests := []profileQueryTest{
		{
			Query: "FOR d IN books SORT d.Title RETURN d",
		},
		{
			Query: "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars: map[string]interface{}{
				"title": "Book 16",
			},
		},
	}

	t.Run("With ForceOneShardAttributeValue", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				nCtx := driver.WithQueryForceOneShardAttributeValue(ctx, "value")
				_, err := db.Query(nCtx, test.Query, test.BindVars)
				require.NoError(t, err)
			})
		}
	})
}

// TestFillBlockCache test FillBlockCache query attribute
func TestFillBlockCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := createClientFromEnv(t, true)

	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.8.1")).Cluster().Enterprise()

	db := ensureDatabase(ctx, c, "fill_block_cache", nil, t)

	db, clean := prepareQueryDatabase(t, ctx, c, "fill_block_cache")
	defer clean(t)

	// Setup tests
	tests := []profileQueryTest{
		{
			Query: "FOR d IN books SORT d.Title RETURN d",
		},
		{
			Query: "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars: map[string]interface{}{
				"title": "Book 16",
			},
		},
	}

	t.Run("With FillBlockCache enabled", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				nCtx := driver.WithQueryFillBlockCache(ctx, true)
				_, err := db.Query(nCtx, test.Query, test.BindVars)
				require.NoError(t, err)
			})
		}
	})

	t.Run("With FillBlockCache disabled", func(t *testing.T) {
		for i, test := range tests {
			t.Run(fmt.Sprintf("Run %d", i), func(t *testing.T) {
				nCtx := driver.WithQueryFillBlockCache(ctx, false)
				_, err := db.Query(nCtx, test.Query, test.BindVars)
				require.NoError(t, err)
			})
		}
	})
}
