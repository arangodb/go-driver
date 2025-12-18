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
//

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_DefaultIndexes(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					indexes, err := col.Indexes(ctx)
					require.NoError(t, err)
					require.NotNil(t, indexes)
					require.Equal(t, 1, len(indexes))
					assert.Equal(t, arangodb.PrimaryIndexType, indexes[0].Type)
				})
			})
		})
	})
}

func Test_DefaultEdgeIndexes(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, &arangodb.CreateCollectionPropertiesV2{Type: utils.NewType(arangodb.CollectionTypeEdge)}, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					indexes, err := col.Indexes(ctx)
					require.NoError(t, err)
					require.NotNil(t, indexes)
					require.Equal(t, 2, len(indexes))

					assert.True(t, slices.ContainsFunc(indexes, func(i arangodb.IndexResponse) bool {
						return i.Type == arangodb.PrimaryIndexType
					}))

					assert.True(t, slices.ContainsFunc(indexes, func(i arangodb.IndexResponse) bool {
						return i.Type == arangodb.EdgeIndexType
					}))
				})
			})
		})
	})
}

func Test_EnsurePersistentIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					var testOptions = []struct {
						ShouldBeCreated bool
						ExpectedNoIdx   int
						Fields          []string
						Opts            *arangodb.CreatePersistentIndexOptions
					}{
						// default options
						{true, 2, []string{"age", "name"}, nil},
						// same as default
						{false, 2, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(false), Sparse: utils.NewType(false)}},

						// unique
						{true, 3, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(true), Sparse: utils.NewType(false)}},
						{false, 3, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(true), Sparse: utils.NewType(false)}},

						{true, 4, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(true), Sparse: utils.NewType(true)}},
						{false, 4, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(true), Sparse: utils.NewType(true)}},

						{true, 5, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(false), Sparse: utils.NewType(true)}},
						{false, 5, []string{"age", "name"},
							&arangodb.CreatePersistentIndexOptions{Unique: utils.NewType(false), Sparse: utils.NewType(true)}},
					}

					for _, testOpt := range testOptions {
						idx, created, err := col.EnsurePersistentIndex(ctx, testOpt.Fields, testOpt.Opts)
						require.NoError(t, err)
						require.Equal(t, created, testOpt.ShouldBeCreated)
						require.Equal(t, arangodb.PersistentIndexType, idx.Type)
						if testOpt.Opts != nil {
							require.Equal(t, testOpt.Opts.Unique, idx.Unique)
							require.Equal(t, testOpt.Opts.Sparse, idx.Sparse)
						} else {
							require.False(t, *idx.Unique)
							require.False(t, *idx.Sparse)
						}
						assert.ElementsMatch(t, idx.RegularIndex.Fields, testOpt.Fields)

						indexes, err := col.Indexes(ctx)
						require.NoError(t, err)
						require.NotNil(t, indexes)
						assert.True(t, slices.ContainsFunc(indexes, func(i arangodb.IndexResponse) bool {
							return i.ID == idx.ID
						}))
						require.Equal(t, testOpt.ExpectedNoIdx, len(indexes))
					}

					t.Run("Create Persistent index with Cache", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.10", t)

						fields := []string{"year", "type"}
						storedValues := []string{"extra1", "extra2"}

						options := &arangodb.CreatePersistentIndexOptions{
							StoredValues: storedValues,
							CacheEnabled: utils.NewType(true),
						}

						idx, created, err := col.EnsurePersistentIndex(ctx, fields, options)
						require.NoError(t, err)
						require.True(t, created)
						require.Equal(t, arangodb.PersistentIndexType, idx.Type)
						require.True(t, *idx.RegularIndex.CacheEnabled)
						assert.ElementsMatch(t, idx.RegularIndex.Fields, fields)
						assert.ElementsMatch(t, idx.RegularIndex.StoredValues, storedValues)

					})
				})
			})
		})
	})
}

func Test_EnsurePersistentIndexDeduplicate(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					doc := struct {
						Tags []string `json:"tags"`
					}{
						Tags: []string{"a", "a", "b"},
					}

					t.Run("Create index with Deduplicate OFF", func(t *testing.T) {
						idx, created, err := col.EnsurePersistentIndex(ctx, []string{"tags[*]"}, &arangodb.CreatePersistentIndexOptions{
							Deduplicate: utils.NewType(false),
							Unique:      utils.NewType(true),
							Sparse:      utils.NewType(false),
						})
						require.NoError(t, err)
						require.True(t, created)
						require.False(t, *idx.RegularIndex.Deduplicate)
						require.Equal(t, arangodb.PersistentIndexType, idx.Type)

						_, err = col.CreateDocument(ctx, doc)
						require.Error(t, err)
						require.True(t, shared.IsConflict(err))

						err = col.DeleteIndexByID(ctx, idx.ID)
						require.NoError(t, err)
					})

					t.Run("Create index with Deduplicate ON", func(t *testing.T) {
						idx, created, err := col.EnsurePersistentIndex(ctx, []string{"tags[*]"}, &arangodb.CreatePersistentIndexOptions{
							Deduplicate: utils.NewType(true),
							Unique:      utils.NewType(true),
							Sparse:      utils.NewType(false),
						})
						require.NoError(t, err)
						require.True(t, created)
						require.True(t, *idx.RegularIndex.Deduplicate)
						require.Equal(t, arangodb.PersistentIndexType, idx.Type)

						_, err = col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						err = col.DeleteIndex(ctx, idx.Name)
						require.NoError(t, err)
					})
				})
			})
		})
	})
}

func Test_TTLIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, 4*time.Minute, func(ctx context.Context, _ testing.TB) {
					t.Run("Removing documents at a fixed period after creation", func(t *testing.T) {
						idx, created, err := col.EnsureTTLIndex(ctx, []string{"createdAt"}, 5, nil)
						require.NoError(t, err)
						defer func() {
							err := col.DeleteIndexByID(ctx, idx.ID)
							require.NoError(t, err)
						}()
						require.True(t, created)
						require.Equal(t, *idx.RegularIndex.ExpireAfter, 5)
						require.Equal(t, arangodb.TTLIndexType, idx.Type)

						doc := struct {
							CreatedAt int64 `json:"createdAt,omitempty"`
						}{
							CreatedAt: time.Now().Unix(),
						}

						meta, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						exist, err := col.DocumentExists(ctx, meta.Key)
						require.NoError(t, err)
						require.True(t, exist)

						// cleanup is made every 30 seconds by default
						withContextT(t, 65*time.Second, func(ctx context.Context, _ testing.TB) {
							for {
								exist, err := col.DocumentExists(ctx, meta.Key)
								require.NoError(t, err)
								if !exist {
									break
								}
								time.Sleep(1 * time.Second)
							}
						})

					})

					t.Run("Removing documents at certain points in time", func(t *testing.T) {
						idx, created, err := col.EnsureTTLIndex(ctx, []string{"expireDate"}, 0, nil)
						require.NoError(t, err)
						defer func() {
							err := col.DeleteIndexByID(ctx, idx.ID)
							require.NoError(t, err)
						}()
						require.True(t, created)
						require.Equal(t, *idx.RegularIndex.ExpireAfter, 0)
						require.Equal(t, arangodb.TTLIndexType, idx.Type)

						doc := struct {
							ExpireDate int64 `json:"expireDate,omitempty"`
						}{
							ExpireDate: time.Now().Add(5 * time.Second).Unix(),
						}

						meta, err := col.CreateDocument(ctx, doc)
						require.NoError(t, err)

						exist, err := col.DocumentExists(ctx, meta.Key)
						require.NoError(t, err)
						require.True(t, exist)

						// cleanup is made every 30 seconds by default
						withContextT(t, 65*time.Second, func(ctx context.Context, _ testing.TB) {
							for {
								exist, err := col.DocumentExists(ctx, meta.Key)
								require.NoError(t, err)
								if !exist {
									break
								}
								time.Sleep(1 * time.Second)
							}
						})
					})
				})
			})
		})
	})
}

func Test_EnsureGeoIndexIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {

					t.Run("Test GeoJSON opts", func(t *testing.T) {
						var testOptions = []arangodb.CreateGeoIndexOptions{
							{GeoJSON: utils.NewType(true)},
							{GeoJSON: utils.NewType(false)},
						}
						for _, testOpt := range testOptions {
							idx, created, err := col.EnsureGeoIndex(ctx, []string{"geo"}, &testOpt)
							require.NoError(t, err)
							require.True(t, created)
							require.Equal(t, arangodb.GeoIndexType, idx.Type)
							require.Equal(t, testOpt.GeoJSON, idx.RegularIndex.GeoJSON)
						}
					})

					t.Run("Test LegacyPolygons opts", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.10", t)
						var testOptions = []struct {
							ExpectedLegacyPolygons bool
							ExpectedGeoJSON        bool
							Fields                 []string
							Opts                   *arangodb.CreateGeoIndexOptions
						}{
							{
								true,
								false,
								[]string{"geoOld1"},
								&arangodb.CreateGeoIndexOptions{LegacyPolygons: utils.NewType(true)},
							},
							{
								false,
								false,
								[]string{"geoOld2"},
								&arangodb.CreateGeoIndexOptions{LegacyPolygons: utils.NewType(false)},
							},
							{
								false,
								true,
								[]string{"geoOld3"},
								&arangodb.CreateGeoIndexOptions{GeoJSON: utils.NewType(true), LegacyPolygons: utils.NewType(false)},
							},
							{
								false,
								false,
								[]string{"geoOld4"},
								&arangodb.CreateGeoIndexOptions{GeoJSON: utils.NewType(false), LegacyPolygons: utils.NewType(false)},
							},
						}

						for _, testOpt := range testOptions {
							idx, created, err := col.EnsureGeoIndex(ctx, testOpt.Fields, testOpt.Opts)
							require.NoError(t, err)
							require.True(t, created)
							require.Equal(t, arangodb.GeoIndexType, idx.Type)
							assert.Equal(t, testOpt.ExpectedGeoJSON, *idx.RegularIndex.GeoJSON)
							assert.Equal(t, testOpt.ExpectedLegacyPolygons, *idx.RegularIndex.LegacyPolygons)
							assert.ElementsMatch(t, idx.RegularIndex.Fields, testOpt.Fields)
						}
					})
				})
			})
		})
	})
}

func Test_NamedIndexes(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					clientVersion, _ := client.Version(ctx)
					t.Logf("Arangodb Version: %s", clientVersion.Version)
					docs := []map[string]interface{}{
						{
							"pername":      "persistent-name",
							"geo":          []float64{12.9716, 77.5946},
							"createdAt":    time.Now().Unix(),
							"mkd":          1.23,
							"mkd-prefixed": 4.56,
							"prefix":       "p1",
							"vectorfield":  []float64{0.1, 0.2, 0.3},
							"text":         "first document",
						},
						{
							"pername":      "persistent-name-2",
							"geo":          []float64{13.0827, 80.2707},
							"createdAt":    time.Now().Unix(),
							"mkd":          2.34,
							"mkd-prefixed": 5.67,
							"prefix":       "p2",
							"vectorfield":  []float64{0.4, 0.5, 0.6},
							"text":         "second document",
						},
					}

					_, err := col.CreateDocuments(ctx, docs)
					require.NoError(t, err)

					var namedIndexTestCases = []struct {
						Name           string
						CreateCallback func(col arangodb.Collection, name string) (arangodb.IndexResponse, error)
						MinVersion     arangodb.Version
					}{
						{
							Name: "Persistent",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsurePersistentIndex(ctx, []string{"pername"}, &arangodb.CreatePersistentIndexOptions{
									Name: name,
								})
								return idx, err
							},
						},
						{
							Name: "Geo",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsureGeoIndex(ctx, []string{"geo"}, &arangodb.CreateGeoIndexOptions{
									Name: name,
								})
								return idx, err
							},
						},
						{
							Name: "TTL",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsureTTLIndex(ctx, []string{"createdAt"}, 3600, &arangodb.CreateTTLIndexOptions{
									Name: name,
								})
								return idx, err
							},
						},
						{
							Name: "MKD",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsureMDIIndex(ctx, []string{"mkd"}, &arangodb.CreateMDIIndexOptions{
									Name:            name,
									FieldValueTypes: arangodb.MDIDoubleFieldType,
								})
								return idx, err
							},
							MinVersion: "3.12",
						},
						{
							Name: "MKD-Prefixed",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsureMDIPrefixedIndex(ctx, []string{"mkd-prefixed"}, &arangodb.CreateMDIPrefixedIndexOptions{
									CreateMDIIndexOptions: arangodb.CreateMDIIndexOptions{
										Name:            name,
										FieldValueTypes: arangodb.MDIDoubleFieldType,
									},
									PrefixFields: []string{"prefix"},
								})
								return idx, err
							},
							MinVersion: "3.12",
						},
						{
							Name:       "Inverted",
							MinVersion: "3.10",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								idx, _, err := col.EnsureInvertedIndex(ctx, &arangodb.InvertedIndexOptions{
									Name: name,
									Fields: []arangodb.InvertedIndexField{
										{
											Name: name,
										},
									},
								})
								if clientVersion.Version.CompareTo("3.12.7") >= 0 {
									require.Equal(t, 0.4, *idx.InvertedIndex.ConsolidationPolicy.MaxSkewThreshold)
									require.Equal(t, 0.5, *idx.InvertedIndex.ConsolidationPolicy.MinDeletionRatio)
								}
								return idx, err
							},
						},
						{
							Name:       "Vector",
							MinVersion: "3.12.4",
							CreateCallback: func(col arangodb.Collection, name string) (arangodb.IndexResponse, error) {
								params := &arangodb.VectorParams{
									Dimension: utils.NewType(3),
									Metric:    utils.NewType(arangodb.VectorMetricCosine),
									NLists:    utils.NewType(1),
								}
								idx, _, err := col.EnsureVectorIndex(ctx, []string{"vectorfield"},
									params, &arangodb.CreateVectorIndexOptions{Name: &name})
								return idx, err
							},
						},
					}

					for _, testCase := range namedIndexTestCases {
						t.Run(fmt.Sprintf("Test named index: %s", testCase.Name), func(t *testing.T) {
							if testCase.MinVersion != "" {
								skipBelowVersion(client, ctx, testCase.MinVersion, t)
							}

							idx, err := testCase.CreateCallback(col, testCase.Name)
							require.NoError(t, err, "failed to create %s index", testCase.Name)
							require.Equal(t, testCase.Name, idx.Name)
							defer func() {
								if idx.ID != "" {
									_ = col.DeleteIndexByID(ctx, idx.ID) // Ignore errors in tests
								}
							}()
							indexes, err := col.Indexes(ctx)
							require.NoError(t, err)
							require.NotNil(t, indexes)
							assert.True(t, slices.ContainsFunc(indexes, func(i arangodb.IndexResponse) bool {
								return i.ID == idx.ID && i.Name == testCase.Name
							}))
						})
					}
				})
			})
		})
	})
}

func Test_EnsureVectorIndex(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
					skipBelowVersion(client, ctx, "3.12.4", t)
					dimension := 3
					metric := arangodb.VectorMetricCosine
					nLists := 1 // or 2, but <= number of docs

					params := &arangodb.VectorParams{
						Dimension: &dimension,
						Metric:    &metric,
						NLists:    &nLists,
					}

					// Vector indexes require documents to be present for training
					// Create sample documents with embeddings
					docs := []map[string]interface{}{
						{"embedding": []float64{0.1, 0.2, 0.3}, "text": "first document"},
						{"embedding": []float64{0.4, 0.5, 0.6}, "text": "second document"},
						{"embedding": []float64{0.7, 0.8, 0.9}, "text": "third document"},
					}

					_, err := col.CreateDocuments(ctx, docs)
					require.NoError(t, err, "failed to create sample documents for vector index training")

					t.Run("Create Vector Index", func(t *testing.T) {
						idx, created, err := col.EnsureVectorIndex(
							ctx,
							[]string{"embedding"},
							params,
							&arangodb.CreateVectorIndexOptions{
								Name: utils.NewType("my_vector_index"),
							},
						)
						require.NoError(t, err)
						require.True(t, created, "index should be created on first call")
						require.Equal(t, arangodb.VectorIndexType, idx.Type)
						require.NotNil(t, idx.VectorIndex)
						require.Equal(t, dimension, *idx.VectorIndex.Dimension)
						require.Equal(t, metric, *idx.VectorIndex.Metric)
					})

					t.Run("Create the same index again", func(t *testing.T) {
						idx, created, err := col.EnsureVectorIndex(
							ctx,
							[]string{"embedding"},
							params,
							nil,
						)
						require.NoError(t, err)
						defer func() {
							if idx.ID != "" {
								_ = col.DeleteIndexByID(ctx, idx.ID) // Ignore errors in cleanup
							}
						}()
						require.False(t, created, "index should already exist")
						require.Equal(t, arangodb.VectorIndexType, idx.Type)
					})

					t.Run("Invalid Vector Index Params", func(t *testing.T) {
						invalidParams := &arangodb.VectorParams{Dimension: utils.NewType(-1)}
						_, _, err := col.EnsureVectorIndex(ctx, []string{"embedding"}, invalidParams, nil)
						require.Error(t, err, "Should fail with invalid dimension")
					})

					var idx arangodb.IndexResponse

					t.Run("Create Vector Index with storedValues", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.12.7", t)
						options := &arangodb.CreateVectorIndexOptions{
							StoredValues: []string{"text"},
						}
						var err error
						idx, _, err = col.EnsureVectorIndex(ctx, []string{"embedding"}, params, options)
						require.NoError(t, err)
						require.Equal(t, arangodb.VectorIndexType, idx.Type)
					})

					if idx.ID == "" {
						t.Skip("Index not created, skipping dependent tests")
					}
					defer func() {
						if idx.ID != "" {
							_ = col.DeleteIndexByID(ctx, idx.ID) // Ignore errors in cleanup
						}
					}()
					// Run explain in a separate subtest
					t.Run("storedValues_are_used_for_filter", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.12.7", t)

						query := fmt.Sprintf(
							"FOR d IN `%s`\n"+
								"  SORT APPROX_NEAR_COSINE(d.embedding, @vector) DESC\n"+
								"  LIMIT 1\n"+
								"  FILTER d.text == @text\n"+
								"  RETURN d",
							col.Name(),
						)

						bindVars := map[string]interface{}{
							"text":   "first document",
							"vector": []float64{0.1, 0.2, 0.3},
						}

						explain, err := db.ExplainQuery(ctx, query, bindVars, nil)
						require.NoError(t, err)
						require.Contains(t, explain.Plan.Rules, "use-vector-index")

						found := false
						for _, node := range explain.Plan.NodesRaw {
							if t, ok := node["type"].(string); ok && t == "EnumerateNearVectorNode" {
								found = true
								break
							}
						}
						if !found {
							t.Logf("Execution plan: %+v", explain.Plan)
						}
						require.True(t, found)
					})

					t.Run("vector_index_with_storedValues_and_indexHint_is_used", func(t *testing.T) {
						skipBelowVersion(client, ctx, "3.12.7", t)
						// indexHint and forceIndexHint for vector indexes supported by 3.12.7+
						// Query using indexHint + forceIndexHint
						query := `
					FOR d IN @@col OPTIONS {
					  indexHint: [@idxName],
					  forceIndexHint: true
					}
					SORT APPROX_NEAR_COSINE(d.embedding, @vector) DESC
					LIMIT 1
					RETURN d
					`
						bindVars := map[string]interface{}{
							"@col":    col.Name(),
							"idxName": idx.Name,
							"vector":  []float64{0.1, 0.2, 0.3},
						}

						// 3. Explain query
						explain, err := db.ExplainQuery(ctx, query, bindVars, nil)
						require.NoError(t, err)

						// 4. Assert vector index is used
						require.Contains(t, explain.Plan.Rules, "use-vector-index")

						// 5. Assert EnumerateNearVectorNode exists
						found := false
						for _, node := range explain.Plan.NodesRaw {
							if nodeType, ok := node["type"].(string); ok && nodeType == "EnumerateNearVectorNode" {
								found = true
								break
							}
						}
						if !found {
							t.Logf("Execution plan: %+v", explain.Plan)
						}
						require.True(t, found, "expected EnumerateNearVectorNode in execution plan")
					})

				})
			})
		})
	})
}
