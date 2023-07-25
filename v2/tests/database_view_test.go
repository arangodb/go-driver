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

package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

// ensureArangoSearchView is a helper to check if an arangosearch view exists and create it if needed.
// It will fail the test when an error occurs.
func ensureArangoSearchView(ctx context.Context, db arangodb.Database, name string, options *arangodb.ArangoSearchViewProperties, t *testing.T) arangodb.ArangoSearchView {
	v, err := db.View(ctx, name)
	if shared.IsNotFound(err) {
		v, err = db.CreateArangoSearchView(ctx, name, options)
		require.NoError(t, err, "Failed to create arangosearch view '%s'", name)
	}
	require.NoError(t, err, "Failed to open view '%s'", name)
	result, err := v.ArangoSearchView()
	require.NoError(t, err, "Failed to open view '%s' as arangosearch view", name)
	return result
}

// checkLinkExists tests if a given collection is linked to the given arangosearch view
func checkLinkExists(ctx context.Context, view arangodb.ArangoSearchView, colName string, t *testing.T) bool {
	props, err := view.Properties(ctx)
	require.NoError(t, err, "Failed to get view properties")
	links := props.Links
	if _, exists := links[colName]; !exists {
		return false
	}
	return true
}

// tryAddArangoSearchLink is a helper that adds a link to a view and collection.
// It will fail the test when an error occurs and returns wether the link is actually there or not.
func tryAddArangoSearchLink(ctx context.Context, view arangodb.ArangoSearchView, colName string, t *testing.T) bool {
	addprop := arangodb.ArangoSearchViewProperties{
		Links: arangodb.ArangoSearchLinks{
			colName: arangodb.ArangoSearchElementProperties{},
		},
	}

	err := view.SetProperties(ctx, addprop)
	require.NoError(t, err, "Could not create link, view: %s, collection: %s", view.Name(), colName)
	return checkLinkExists(ctx, view, colName, t)
}

// Test_CreateArangoSearchView creates an arangosearch view and then checks that it exists.
func Test_CreateArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				name := "test_create_asview"
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{},
					},
				}
				v, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// View must exist now
				if found, err := db.ViewExists(ctx, name); err != nil {
					t.Errorf("ViewExists('%s') failed: %s", name, err.Error())
				} else if !found {
					t.Errorf("ViewExists('%s') return false, expected true", name)
				}
				// Check v.Name
				if actualName := v.Name(); actualName != name {
					t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, name)
				}
				// Check v properties
				p, err := v.Properties(ctx)
				require.NoError(t, err, "Properties failed")
				if len(p.Links) != 1 {
					t.Errorf("Expected 1 link, got %d", len(p.Links))
				}
			})
		})
	})
}

// Test_CreateArangoSearchViewInvalidLinks attempts to create an arangosearch view with invalid links and then checks that it does not exist.
func Test_CreateArangoSearchViewInvalidLinks(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {

				skipBelowVersion(client, ctx, "3.4", t)
				name := "test_create_inv_view"
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						"some_nonexistent_col": arangodb.ArangoSearchElementProperties{},
					},
				}
				_, err := db.CreateArangoSearchView(ctx, name, opts)
				if err == nil {
					t.Fatalf("Creating view did not fail")
				}
				// View must not exist now
				if found, err := db.ViewExists(ctx, name); err != nil {
					t.Errorf("ViewExists('%s') failed: %s", name, err.Error())
				} else if found {
					t.Errorf("ViewExists('%s') return true, expected false", name)
				}
				// Try to open view, must fail as well
				if v, err := db.View(ctx, name); !shared.IsNotFound(err) {
					t.Errorf("Expected NotFound error from View('%s'), got %s instead (%#v)", name, err.Error(), v)
				}
			})
		})
	})
}

// Test_CreateEmptyArangoSearchView creates an arangosearch view without any links.
func Test_CreateEmptyArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				name := "test_create_empty_asview"
				v, err := db.CreateArangoSearchView(ctx, name, nil)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// View must exist now
				if found, err := db.ViewExists(ctx, name); err != nil {
					t.Errorf("ViewExists('%s') failed: %s", name, err.Error())
				} else if !found {
					t.Errorf("ViewExists('%s') return false, expected true", name)
				}
				// Check v properties
				p, err := v.Properties(ctx)
				require.NoError(t, err, "Properties failed")
				if len(p.Links) != 0 {
					t.Errorf("Expected 0 links, got %d", len(p.Links))
				}
			})
		})
	})
}

// Test_CreateDuplicateArangoSearchView creates an arangosearch view twice and then checks that it exists.
func Test_CreateDuplicateArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				name := "test_create_dup_asview"
				_, err := db.CreateArangoSearchView(ctx, name, nil)
				require.NoError(t, err, "Failed to create view '%s'", name)

				// View must exist now
				if found, err := db.ViewExists(ctx, name); err != nil {
					t.Errorf("ViewExists('%s') failed: %s", name, err.Error())
				} else if !found {
					t.Errorf("ViewExists('%s') return false, expected true", name)
				}
				// Try to create again. Must fail
				if _, err := db.CreateArangoSearchView(ctx, name, nil); !shared.IsConflict(err) {
					require.NoError(t, err, "Expect a Conflict error from CreateArangoSearchView")
				}
			})
		})
	})
}

// Test_CreateArangoSearchViewThenRemoveCollection creates an arangosearch view
// with a link to an existing collection and the removes that collection.
func Test_CreateArangoSearchViewThenRemoveCollection(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)
				name := "test_create_view_then_rem_col"
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{},
					},
				}
				v, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// View must exist now
				if found, err := db.ViewExists(ctx, name); err != nil {
					t.Errorf("ViewExists('%s') failed: %s", name, err.Error())
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
					require.NoError(t, err, "Properties failed")
				}
				if len(p.Links) != 1 {
					t.Errorf("Expected 1 link, got %d", len(p.Links))
				}

				// Now delete the collection
				err = col.Remove(ctx)
				require.NoError(t, err, "Failed to remove collection '%s': %s", col.Name())

				// Re-check v properties
				p, err = v.Properties(ctx)
				if err != nil {
					require.NoError(t, err, "Properties failed")
				}
				if len(p.Links) != 0 {
					// TODO is the really the correct expected behavior.
					t.Errorf("Expected 0 links, got %d", len(p.Links))
				}
			})
		})
	})
}

// Test_AddCollectionMultipleViews creates a collection and two view. adds the collection to both views
// and checks if the links exist. The links are set via modifying properties.
func Test_AddCollectionMultipleViews(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				v1 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view1", nil, t)
				if !tryAddArangoSearchLink(ctx, v1, col.Name(), t) {
					t.Error("Link does not exists")
				}
				v2 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view2", nil, t)
				if !tryAddArangoSearchLink(ctx, v2, col.Name(), t) {
					t.Error("Link does not exists")
				}
			})
		})
	})
}

// Test_AddCollectionMultipleViews creates a collection and two view. adds the collection to both views
// and checks if the links exist. The links are set when creating the view.
func Test_AddCollectionMultipleViewsViaCreate(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{},
					},
				}
				v1 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view1", opts, t)
				if !checkLinkExists(ctx, v1, col.Name(), t) {
					t.Error("Link does not exists")
				}
				v2 := ensureArangoSearchView(ctx, db, "col_in_multi_view_view2", opts, t)
				if !checkLinkExists(ctx, v2, col.Name(), t) {
					t.Error("Link does not exists")
				}
			})
		})
	})
}

// Test_GetArangoSearchOptimizeTopK creates an ArangoSearch view with OptimizeTopK and checks if it is set.
func Test_GetArangoSearchOptimizeTopK(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.12.0", t)
				skipNoEnterprise(client, ctx, t)

				name := "test_get_asview"
				optimizeTopK := []string{"BM25(@doc) DESC", "TFIDF(@doc) DESC"}
				opts := &arangodb.ArangoSearchViewProperties{
					OptimizeTopK: optimizeTopK,
				}
				_, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// Get view
				v, err := db.View(ctx, name)
				require.NoError(t, err, "View('%s') failed", name)
				asv, err := v.ArangoSearchView()
				require.NoError(t, err, "ArangoSearchView() failed")
				// Check v.Name
				if actualName := v.Name(); actualName != name {
					t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, name)
				}
				// Check asv properties
				p, err := asv.Properties(ctx)
				require.NoError(t, err, "Properties failed")
				assert.Equal(t, optimizeTopK, p.OptimizeTopK)
			})
		})
	})
}

// Test_GetArangoSearchView creates an ArangoSearch view and then gets it again.
func Test_GetArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)
				name := "test_get_asview"
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{},
					},
				}
				_, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// Get view
				v, err := db.View(ctx, name)
				require.NoError(t, err, "View('%s') failed", name)
				asv, err := v.ArangoSearchView()
				require.NoError(t, err, "ArangoSearchView() failed")
				// Check v.Name
				if actualName := v.Name(); actualName != name {
					t.Errorf("Name() failed. Got '%s', expected '%s'", actualName, name)
				}
				// Check asv properties
				p, err := asv.Properties(ctx)
				require.NoError(t, err, "Properties failed")
				if len(p.Links) != 1 {
					t.Errorf("Expected 1 link, got %d", len(p.Links))
				}
				// Check indexes on collection
				indexes, err := col.Indexes(ctx)
				require.NoError(t, err, "Indexes() failed")
				if len(indexes) != 1 {
					// 1 is always added by the system
					t.Errorf("Expected 1 index, got %d", len(indexes))
				}
			})
		})
	})
}

func readAllViewsT(ctx context.Context, t *testing.T, db arangodb.Database) []arangodb.View {
	t.Helper()
	r, err := db.Views(ctx)
	require.NoError(t, err, "Views failed")

	result := make([]arangodb.View, 0)
	for {
		a, err := r.Read()
		if shared.IsNoMoreDocuments(err) {
			return result
		}
		require.NoError(t, err)
		result = append(result, a)
	}
}

// Test_GetArangoSearchViews creates several arangosearch views and then gets all of them.
func Test_GetArangoSearchViews(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)
				// Get views before adding some
				before := readAllViewsT(ctx, t, db)
				// Create views
				names := make([]string, 5)
				for i := 0; i < len(names); i++ {
					names[i] = fmt.Sprintf("test_get_views_%d", i)
					_, err := db.CreateArangoSearchView(ctx, names[i], nil)
					require.NoError(t, err, "Failed to create view '%s'", names[i])
				}
				// Get views
				after := readAllViewsT(ctx, t, db)
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
			})
		})
	})
}

// Test_RenameAndRemoveArangoSearchView creates an arangosearch view, renames it and then removes it.
func Test_RenameAndRemoveArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				name := "test_rename_view"
				renamedView := "test_rename_view_new"
				v, err := db.CreateArangoSearchView(ctx, name, nil)
				require.NoError(t, err)

				// View must exist now
				found, err := db.ViewExists(ctx, name)
				require.NoError(t, err)
				require.True(t, found)

				t.Run("rename view - single server only", func(t *testing.T) {
					requireMode(t, testModeSingle)

					// Rename view
					err = v.Rename(ctx, renamedView)
					require.NoError(t, err)
					require.Equal(t, renamedView, v.Name())

					// Renamed View must exist
					found, err = db.ViewExists(ctx, renamedView)
					require.NoError(t, err)
					require.True(t, found)
				})

				// Now remove it
				err = v.Remove(ctx)
				require.NoError(t, err)

				// View must not exist now
				found, err = db.ViewExists(ctx, name)
				require.NoError(t, err)
				require.False(t, found)

				t.Run("ensure renamed view not exist - single server only", func(t *testing.T) {
					requireMode(t, testModeSingle)

					found, err = db.ViewExists(ctx, renamedView)
					require.NoError(t, err)
					require.False(t, found)
				})

			})
		})
	})
}

// Test_UseArangoSearchView tries to create a view and actually use it in
// an AQL query.
func Test_UseArangoSearchView(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.4", t)

				ensureArangoSearchView(ctx, db, "some_view", &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{
							Fields: arangodb.ArangoSearchFields{
								"name": arangodb.ArangoSearchElementProperties{},
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

				insertBatch(t, ctx, col, nil, docs)

				// now access it via AQL with waitForSync
				{
					cur, err := db.Query(ctx, `FOR doc IN some_view SEARCH doc.name == "John" OPTIONS {waitForSync:true} RETURN doc`, &arangodb.QueryOptions{
						Count: true,
					})
					require.NoError(t, err, "Failed to query data using arangodsearch")
					if cur.Count() != 1 || !cur.HasMore() {
						t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
					}

					var doc UserDoc
					_, err = cur.ReadDocument(ctx, &doc)
					require.NoError(t, err, "Failed to read document")

					if doc.Name != "John" {
						t.Fatalf("Expected result `John`, found `%s`", doc.Name)
					}
				}

				// now access it via AQL without waitForSync
				{
					cur, err := db.Query(ctx, `FOR doc IN some_view SEARCH doc.name == "John" RETURN doc`, &arangodb.QueryOptions{
						Count: true,
					})
					require.NoError(t, err, "Failed to query data using arangodsearch")
					if cur.Count() != 1 || !cur.HasMore() {
						t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
					}

					var doc UserDoc
					_, err = cur.ReadDocument(ctx, &doc)
					require.NoError(t, err, "Failed to read document")

					if doc.Name != "John" {
						t.Fatalf("Expected result `John`, found `%s`", doc.Name)
					}
				}
			})
		})
	})
}

// Test_UseArangoSearchViewWithNested tries to create a view with nested fields and actually use it in an AQL query.
func Test_UseArangoSearchViewWithNested(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {

				skipBelowVersion(client, ctx, "3.10", t)
				skipNoEnterprise(client, ctx, t)

				ensureArangoSearchView(ctx, db, "some_nested_view", &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{
							Fields: arangodb.ArangoSearchFields{
								"dimensions": arangodb.ArangoSearchElementProperties{
									Nested: arangodb.ArangoSearchFields{
										"type":  arangodb.ArangoSearchElementProperties{},
										"value": arangodb.ArangoSearchElementProperties{},
									},
								},
							},
						},
					},
				}, t)

				type dimension struct {
					Type  string `json:"type"`
					Value int    `json:"value"`
				}

				type nestedFieldsDoc struct {
					Name       string      `json:"name"`
					Dimensions []dimension `json:"dimensions,omitempty"`
				}
				docs := []nestedFieldsDoc{
					{
						Name: "John",
						Dimensions: []dimension{
							{"height", 10},
							{"weight", 80},
						},
					},
					{
						Name: "Jakub",
						Dimensions: []dimension{
							{"height", 25},
							{"weight", 80},
						},
					},
					{
						Name: "Marek",
						Dimensions: []dimension{
							{"height", 30},
							{"weight", 80},
						},
					},
				}

				insertBatch(t, ctx, col, nil, docs)

				// now access it via AQL with waitForSync
				{
					query := "FOR doc IN some_nested_view SEARCH doc.dimensions[? FILTER CURRENT.type == \"height\" AND CURRENT.value > 20] OPTIONS {waitForSync:true} RETURN doc"
					cur, err := db.Query(ctx, query, &arangodb.QueryOptions{
						Count: true,
					})
					require.NoError(t, err, "Failed to query data using arangodsearch")
					if cur.Count() != 2 || !cur.HasMore() {
						t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
					}
				}
			})
		})
	})
}

// Test_UseArangoSearchViewWithPipelineAnalyzer tries to create a view and analyzer and then actually use it in an AQL query.
func Test_UseArangoSearchViewWithPipelineAnalyzer(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.8", t)

				analyzer := arangodb.AnalyzerDefinition{
					Name: "custom_analyzer",
					Type: arangodb.ArangoSearchAnalyzerTypePipeline,
					Properties: arangodb.ArangoSearchAnalyzerProperties{
						Pipeline: []arangodb.ArangoSearchAnalyzerPipeline{
							{
								Type: arangodb.ArangoSearchAnalyzerTypeNGram,
								Properties: arangodb.ArangoSearchAnalyzerProperties{
									Min:              newT[int64](2),
									Max:              newT[int64](2),
									PreserveOriginal: newBool(false),
									StreamType:       newT(arangodb.ArangoSearchNGramStreamUTF8),
								},
							},
							{
								Type: arangodb.ArangoSearchAnalyzerTypeNorm,
								Properties: arangodb.ArangoSearchAnalyzerProperties{
									Locale: "en",
									Case:   arangodb.ArangoSearchCaseLower,
								},
							},
						},
					},
					Features: []arangodb.ArangoSearchFeature{
						arangodb.ArangoSearchFeatureFrequency,
						arangodb.ArangoSearchFeaturePosition,
						arangodb.ArangoSearchFeatureNorm,
					},
				}
				existed, _, err := db.EnsureAnalyzer(ctx, &analyzer)
				require.NoError(t, err)
				require.False(t, existed)

				ensureArangoSearchView(ctx, db, "some_view_with_analyzer", &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{
							Fields: arangodb.ArangoSearchFields{
								"name": arangodb.ArangoSearchElementProperties{
									Analyzers: []string{"custom_analyzer"},
								},
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
						12,
					},
					{
						"Helmut",
						17,
					},
				}

				insertBatch(t, ctx, col, nil, docs)

				// now access it via AQL with waitForSync
				{
					cur, err := db.Query(ctx, `FOR doc IN some_view_with_analyzer SEARCH NGRAM_MATCH(doc.name, 'john', 0.75, 'custom_analyzer')  OPTIONS {waitForSync:true} RETURN doc`, &arangodb.QueryOptions{
						Count: true,
					})
					require.NoError(t, err, "Failed to query data using arangosearch")
					if cur.Count() != 1 || !cur.HasMore() {
						t.Fatalf("Wrong number of return values: expected 1, found %d", cur.Count())
					}

					var doc UserDoc
					_, err = cur.ReadDocument(ctx, &doc)
					require.NoError(t, err, "Failed to read document")

					if doc.Name != "John" {
						t.Fatalf("Expected result `John`, found `%s`", doc.Name)
					}
				}
			})
		})
	})
}

// Test_GetArangoSearchView creates an arangosearch view and then gets it again.
func Test_ArangoSearchViewProperties35(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.7.1", t)
				commitInterval := int64(100)
				name := "test_get_asview_35"
				sortField := "foo"
				storedValuesFields := []string{"now", "is", "the", "time"}
				storedValuesCompression := arangodb.PrimarySortCompressionNone
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{},
					},
					CommitInterval: &commitInterval,
					PrimarySort: []arangodb.ArangoSearchPrimarySortEntry{{
						Field:     sortField,
						Ascending: newBool(false),
					}},
					StoredValues: []arangodb.StoredValue{{
						Fields:      storedValuesFields,
						Compression: storedValuesCompression,
					}},
				}
				_, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err, "Failed to create view '%s'", name)
				// Get view
				v, err := db.View(ctx, name)
				require.NoError(t, err, "View('%s') failed", name)
				asv, err := v.ArangoSearchView()
				require.NoError(t, err, "ArangoSearchView() failed")
				// Check asv properties
				p, err := asv.Properties(ctx)
				require.NoError(t, err, "Properties failed")
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
			})
		})
	})
}

// Test_ArangoSearchPrimarySort
func Test_ArangoSearchPrimarySort(t *testing.T) {
	ctx := context.Background()

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.5", t)

				boolTrue := true
				boolFalse := false

				testCases := []struct {
					Name              string
					InAscending       *bool
					ExpectedAscending *bool
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
				}

				for _, testCase := range testCases {
					t.Run(testCase.Name, func(t *testing.T) {
						// Create the view with given parameters
						opts := &arangodb.ArangoSearchViewProperties{
							Links: arangodb.ArangoSearchLinks{
								col.Name(): arangodb.ArangoSearchElementProperties{},
							},
							PrimarySort: []arangodb.ArangoSearchPrimarySortEntry{{
								Field:     "foo",
								Ascending: testCase.InAscending,
							}},
						}

						name := fmt.Sprintf("%s-view", testCase.Name)

						if _, err := db.CreateArangoSearchView(ctx, name, opts); err != nil {

							if !shared.IsArangoErrorWithCode(err, testCase.ErrorCode) {
								require.NoError(t, err, "Failed to create view '%s'", name)
							} else {
								// end test here
								return
							}
						}

						// Get view
						v, err := db.View(ctx, name)
						require.NoError(t, err, "View('%s') failed", name)
						asv, err := v.ArangoSearchView()
						require.NoError(t, err, "ArangoSearchView() failed")
						// Check asv properties
						p, err := asv.Properties(ctx)
						require.NoError(t, err, "Properties failed")
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
						}
					})
				}
			})
		})
	})
}

// Test_ArangoSearchViewProperties353 tests for custom analyzers.
func Test_ArangoSearchViewProperties353(t *testing.T) {
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				skipBelowVersion(client, ctx, "3.5.3", t)
				requireClusterMode(t)

				name := "test_get_asview_353"
				analyzerName := "myanalyzer"
				opts := &arangodb.ArangoSearchViewProperties{
					Links: arangodb.ArangoSearchLinks{
						col.Name(): arangodb.ArangoSearchElementProperties{
							AnalyzerDefinitions: []arangodb.AnalyzerDefinition{
								{
									Name: analyzerName,
									Type: arangodb.ArangoSearchAnalyzerTypeNorm,
									Properties: arangodb.ArangoSearchAnalyzerProperties{
										Locale: "en_US",
										Case:   arangodb.ArangoSearchCaseLower,
									},
									Features: []arangodb.ArangoSearchFeature{
										arangodb.ArangoSearchFeaturePosition,
										arangodb.ArangoSearchFeatureFrequency,
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
				require.Contains(t, p.Links, col.Name())
			})
		})
	})
}

func Test_ArangoSearchViewLinkAndStoredValueCache(t *testing.T) {
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				// feature was introduced in 3.9.5 and in 3.10.2:
				skipBelowVersion(client, ctx, "3.9.5", t)
				skipBetweenVersions(client, ctx, "3.10.0", "3.10.1", t)
				skipNoEnterprise(client, ctx, t)

				linkedColName := col.Name()

				name := "test_create_asview"
				opts := &arangodb.ArangoSearchViewProperties{
					StoredValues: []arangodb.StoredValue{
						{
							Fields: []string{"f1", "f2"},
							Cache:  newBool(true),
						},
					},
					Links: arangodb.ArangoSearchLinks{
						linkedColName: arangodb.ArangoSearchElementProperties{
							Cache: newBool(false),
						},
					},
				}
				v, err := db.CreateArangoSearchView(ctx, name, opts)
				require.NoError(t, err)

				// check props
				p, err := v.Properties(ctx)
				require.NoError(t, err)
				require.Len(t, p.StoredValues, 1)
				require.Equal(t, newBool(true), p.StoredValues[0].Cache)
				linkedColumnProps := p.Links[linkedColName]
				require.NotNil(t, linkedColumnProps)
				require.Nil(t, linkedColumnProps.Cache)
				// update props: set to cached
				p.Links[linkedColName] = arangodb.ArangoSearchElementProperties{Cache: newBool(true)}
				err = v.SetProperties(ctx, p)
				require.NoError(t, err)

				// check updates applied
				p, err = v.Properties(ctx)
				require.NoError(t, err)
				linkedColumnProps = p.Links[linkedColName]
				require.NotNil(t, linkedColumnProps)
				require.Equal(t, newBool(true), linkedColumnProps.Cache)
			})
		})
	})
}

func Test_ArangoSearchViewInMemoryCache(t *testing.T) {
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {

				skipNoEnterprise(client, ctx, t)

				t.Run("primarySortCache", func(t *testing.T) {
					// feature was introduced in 3.9.5 and in 3.10.2:
					skipBelowVersion(client, ctx, "3.9.5", t)
					skipBetweenVersions(client, ctx, "3.10.0", "3.10.1", t)

					name := "test_create_asview"
					opts := &arangodb.ArangoSearchViewProperties{
						PrimarySortCache: newBool(true),
					}
					v, err := db.CreateArangoSearchView(ctx, name, opts)
					require.NoError(t, err)

					p, err := v.Properties(ctx)
					require.NoError(t, err)
					// bug in arangod: the primarySortCache field is not returned in response. Fixed only in 3.9.6+:
					t.Run("must-be-returned-in-response", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.9.6", t)
						require.Equal(t, newBool(true), p.PrimarySortCache)
					})
				})

				t.Run("primaryKeyCache", func(t *testing.T) {
					// feature was introduced in 3.9.6 and 3.10.2:
					skipBelowVersion(client, ctx, "3.9.6", t)
					skipBetweenVersions(client, ctx, "3.10.0", "3.10.1", t)

					name := "test_view_"
					opts := &arangodb.ArangoSearchViewProperties{
						PrimaryKeyCache: newBool(true),
					}
					v, err := db.CreateArangoSearchView(ctx, name, opts)
					require.NoError(t, err)

					p, err := v.Properties(ctx)
					require.NoError(t, err)
					require.Equal(t, newBool(true), p.PrimaryKeyCache)
				})
			})
		})
	})
}
