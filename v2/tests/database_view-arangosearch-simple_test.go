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

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_ArangoSearchSimple(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					viewName := GenerateUUID("test-view")

					opts := &arangodb.ArangoSearchViewProperties{
						CleanupIntervalStep: utils.NewType[int64](1),
						CommitInterval:      utils.NewType[int64](500),
					}

					clientVersion, _ := client.Version(ctx)
					t.Logf("Arangodb Version: %s", clientVersion.Version)

					view, err := db.CreateArangoSearchView(ctx, viewName, opts)
					require.NoError(t, err, "Failed to create alias view '%s'", viewName)

					prop, err := view.Properties(ctx)
					require.NoError(t, err)
					if clientVersion.Version.CompareTo("3.12.7") >= 0 {
						require.Equal(t, 0.4, *prop.ConsolidationPolicy.MaxSkewThreshold)
						require.Equal(t, 0.5, *prop.ConsolidationPolicy.MinDeletionRatio)
					}
					require.Equal(t, prop.Name, viewName)
					require.Equal(t, int64(1), *prop.CleanupIntervalStep)
					require.Equal(t, int64(500), *prop.CommitInterval)

					t.Run("Update properties of the view", func(t *testing.T) {
						opt := arangodb.ArangoSearchViewProperties{
							CommitInterval: utils.NewType[int64](200),
						}
						err = view.UpdateProperties(ctx, opt)
						require.NoError(t, err)

						pr, err := view.Properties(ctx)
						require.NoError(t, err)
						if clientVersion.Version.CompareTo("3.12.7") >= 0 {
							require.Equal(t, 0.4, *pr.ConsolidationPolicy.MaxSkewThreshold)
							require.Equal(t, 0.5, *pr.ConsolidationPolicy.MinDeletionRatio)
						}
						require.Equal(t, pr.Type, arangodb.ViewTypeArangoSearch)
						require.Equal(t, pr.Name, viewName)
						require.Equal(t, int64(1), *pr.CleanupIntervalStep)
						require.Equal(t, int64(200), *pr.CommitInterval)
					})

					t.Run("Replace properties of the view", func(t *testing.T) {
						opt := arangodb.ArangoSearchViewProperties{
							CommitInterval: utils.NewType[int64](300),
						}
						err = view.SetProperties(ctx, opt)
						require.NoError(t, err)

						pr, err := view.Properties(ctx)
						require.NoError(t, err)
						if clientVersion.Version.CompareTo("3.12.7") >= 0 {
							require.Equal(t, 0.4, *pr.ConsolidationPolicy.MaxSkewThreshold)
							require.Equal(t, 0.5, *pr.ConsolidationPolicy.MinDeletionRatio)
						}
						require.Equal(t, pr.Type, arangodb.ViewTypeArangoSearch)
						require.Equal(t, pr.Name, viewName)
						require.Equal(t, int64(300), *pr.CommitInterval)
						// check if the cleanup interval step is reverted to the default value (2)
						require.Equal(t, int64(2), *pr.CleanupIntervalStep)
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
