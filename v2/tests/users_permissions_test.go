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
//

package tests

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_UserPermission(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

				t.Run("Test root user", func(t *testing.T) {
					userRoot, err := client.User(ctx, "root")
					require.NoError(t, err)

					dbs, err := userRoot.AccessibleDatabases(ctx)
					require.NoError(t, err)
					require.Contains(t, dbs, db.Name())

					dbsFull, err := userRoot.AccessibleDatabasesFull(ctx)
					require.NoError(t, err)
					require.Contains(t, dbsFull, db.Name())
					require.Contains(t, dbsFull, "*")

					grant := dbsFull[db.Name()]
					require.Greater(t, len(grant.Collections), 1)
					require.Equal(t, grant.Permission, arangodb.GrantUndefined)

					dbAccess, err := userRoot.GetDatabaseAccess(ctx, "_system")
					require.NoError(t, err)
					require.Equal(t, dbAccess, arangodb.GrantReadWrite)

					colAccess, err := userRoot.GetCollectionAccess(ctx, "_system", "_users")
					require.NoError(t, err)
					require.Equal(t, colAccess, arangodb.GrantNone)
				})

				t.Run("Test custom user", func(t *testing.T) {
					WithCollection(t, db, nil, func(col arangodb.Collection) {
						userCustom, err := client.CreateUser(ctx, "custom"+GenerateUUID("user-db"), nil)
						require.NoError(t, err)
						require.NotNil(t, userCustom)

						dbAccess, err := userCustom.GetDatabaseAccess(ctx, db.Name())
						require.NoError(t, err)
						require.Equal(t, dbAccess, arangodb.GrantNone)

						colAccess, err := userCustom.GetCollectionAccess(ctx, db.Name(), col.Name())
						require.NoError(t, err)
						require.Equal(t, arangodb.GrantNone, colAccess)

						t.Run("Test grant DB access", func(t *testing.T) {
							require.NoError(t, userCustom.SetDatabaseAccess(ctx, db.Name(), arangodb.GrantReadOnly))

							dbAccess, err = userCustom.GetDatabaseAccess(ctx, db.Name())
							require.NoError(t, err)
							require.Equal(t, dbAccess, arangodb.GrantReadOnly)

							colAccess, err = userCustom.GetCollectionAccess(ctx, db.Name(), col.Name())
							require.NoError(t, err)
							require.Equal(t, arangodb.GrantReadOnly, colAccess)
						})

						t.Run("Test grant Collection access", func(t *testing.T) {
							require.NoError(t, userCustom.SetCollectionAccess(ctx, db.Name(), col.Name(), arangodb.GrantReadWrite))

							dbAccess, err = userCustom.GetDatabaseAccess(ctx, db.Name())
							require.NoError(t, err)
							require.Equal(t, dbAccess, arangodb.GrantReadOnly)

							colAccess, err = userCustom.GetCollectionAccess(ctx, db.Name(), col.Name())
							require.NoError(t, err)
							require.Equal(t, arangodb.GrantReadWrite, colAccess)

							require.NoError(t, userCustom.RemoveCollectionAccess(ctx, db.Name(), col.Name()))
							colAccess, err = userCustom.GetCollectionAccess(ctx, db.Name(), col.Name())
							require.NoError(t, err)
							require.Equal(t, arangodb.GrantReadOnly, colAccess)
						})

						t.Run("Test remove DB access", func(t *testing.T) {
							require.NoError(t, userCustom.RemoveDatabaseAccess(ctx, db.Name()))

							dbAccess, err = userCustom.GetDatabaseAccess(ctx, db.Name())
							require.NoError(t, err)
							require.Equal(t, dbAccess, arangodb.GrantNone)
						})
					})

				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}
