//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_ArangoSearchAliasView(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					nameAlias := "test_add_collection_view_alias"
					nameInvInd1 := "inv_index_alias_view1"
					nameInvInd2 := "inv_index_alias_view2"

					view, err := db.CreateArangoSearchAliasView(ctx, nameAlias, nil)
					require.NoError(t, err, "Failed to create alias view '%s'", nameAlias)

					prop, err := view.Properties(ctx)
					require.NoError(t, err)
					require.Equal(t, prop.Type, arangodb.ViewTypeSearchAlias)
					require.Equal(t, prop.Name, nameAlias)
					require.Len(t, prop.Indexes, 0)

					_, err = view.ArangoSearchView()
					require.Error(t, err)

					t.Run("Add Inverted Index to the collection", func(t *testing.T) {
						idx, created, err := col.EnsureInvertedIndex(ctx, sampleIndex(nameInvInd1))
						require.NoError(t, err)
						require.True(t, created)
						require.Equal(t, nameInvInd1, idx.Name)

						idx2, created, err := col.EnsureInvertedIndex(ctx, sampleIndex(nameInvInd2))
						require.NoError(t, err)
						require.True(t, created)
						require.Equal(t, nameInvInd2, idx2.Name)
					})

					t.Run("Set properties of the view", func(t *testing.T) {
						opt := arangodb.ArangoSearchAliasViewProperties{
							Indexes: []arangodb.ArangoSearchAliasIndex{
								{
									Collection: col.Name(),
									Index:      nameInvInd1,
								},
							},
						}
						err = view.SetProperties(ctx, opt)
						require.NoError(t, err)
					})

					t.Run("Get properties of the view", func(t *testing.T) {
						prop, err = view.Properties(ctx)
						require.NoError(t, err)
						require.Equal(t, prop.Type, arangodb.ViewTypeSearchAlias)
						require.Equal(t, prop.Name, nameAlias)
						require.Len(t, prop.Indexes, 1)
						require.Equal(t, prop.Indexes[0].Collection, col.Name())

						views, err := db.ViewsAll(ctx)
						require.NoError(t, err)
						require.Len(t, views, 1)

						exist, err := db.ViewExists(ctx, nameAlias)
						require.NoError(t, err)
						require.True(t, exist)

						vv, err := db.View(ctx, nameAlias)
						require.NoError(t, err)
						require.Equal(t, vv.Name(), nameAlias)
					})

					t.Run("Update properties of the view", func(t *testing.T) {
						opt := arangodb.ArangoSearchAliasUpdateOpts{
							Indexes: []arangodb.ArangoSearchAliasIndex{
								{
									Collection: col.Name(),
									Index:      nameInvInd2,
								},
							},
							Operation: arangodb.ArangoSearchAliasOperationAdd,
						}
						err = view.UpdateProperties(ctx, opt)
						require.NoError(t, err)

						pr, err := view.Properties(ctx)
						require.NoError(t, err)
						require.Equal(t, pr.Type, arangodb.ViewTypeSearchAlias)
						require.Equal(t, pr.Name, nameAlias)
						require.Len(t, pr.Indexes, 2)
					})

					t.Run("Replace properties of the view", func(t *testing.T) {
						opt := arangodb.ArangoSearchAliasViewProperties{
							Indexes: []arangodb.ArangoSearchAliasIndex{
								{
									Collection: col.Name(),
									Index:      nameInvInd2,
								},
							},
						}
						err = view.SetProperties(ctx, opt)
						require.NoError(t, err)

						pr, err := view.Properties(ctx)
						require.NoError(t, err)
						require.Len(t, pr.Indexes, 1)
						require.Equal(t, pr.Indexes[0].Index, nameInvInd2)
					})

					t.Run("Remove the view", func(t *testing.T) {
						err = view.Remove(ctx)
						require.NoError(t, err)

						views, err := db.ViewsAll(ctx)
						require.NoError(t, err)
						require.Len(t, views, 0)
					})
				})
			})
		})
	})
}

func sampleIndex(nameInvInd string) *arangodb.InvertedIndexOptions {
	indexOpt := arangodb.InvertedIndexOptions{
		Name: nameInvInd,
		Fields: []arangodb.InvertedIndexField{
			{
				Name:               nameInvInd,
				Features:           []arangodb.ArangoSearchFeature{arangodb.ArangoSearchFeatureFrequency, arangodb.ArangoSearchFeaturePosition},
				TrackListPositions: false,
				Nested:             nil,
			},
		},
	}
	return &indexOpt
}
