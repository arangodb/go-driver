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
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_Users(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			user1Name := "kuba@example" + GenerateUUID("user-db")

			user2Name := "testOpts" + GenerateUUID("user-db")
			doc := UserDoc{
				Name: "Jakub",
				Age:  30,
			}

			t.Run("Test list users", func(t *testing.T) {
				users, err := client.Users(ctx)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(users), 1)
			})

			defer func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				err := client.RemoveUser(cleanupCtx, user1Name)
				if err != nil {
					t.Logf("Failed to delete user %s: %s ...", user1Name, err)
				}
			}()
			t.Run("Test created user", func(t *testing.T) {
				u, err := client.CreateUser(ctx, user1Name, nil)
				require.NoError(t, err)
				require.NotNil(t, u)
				require.Equal(t, user1Name, u.Name())
				require.True(t, u.IsActive())

				ur, err := client.User(ctx, user1Name)
				require.NoError(t, err)
				require.NotNil(t, ur)
				require.Equal(t, user1Name, ur.Name())
			})

			defer func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				err := client.RemoveUser(cleanupCtx, user2Name)
				if err != nil {
					t.Logf("Failed to delete user %s: %s ...", user2Name, err)
				}
			}()
			t.Run("Test created user with options", func(t *testing.T) {
				opts := &arangodb.UserOptions{
					Extra: doc,
				}

				u, err := client.CreateUser(ctx, user2Name, opts)
				require.NoError(t, err)
				require.NotNil(t, u)
				require.Equal(t, user2Name, u.Name())
				require.True(t, u.IsActive())

				ur, err := client.User(ctx, user2Name)
				require.NoError(t, err)
				require.NotNil(t, ur)
				require.Equal(t, user2Name, ur.Name())

				var docRead UserDoc
				require.NoError(t, ur.Extra(&docRead))
				require.Equal(t, doc, docRead)
			})

			t.Run("Test list users", func(t *testing.T) {
				users, err := client.Users(ctx)
				require.NoError(t, err)

				for _, u := range users {
					if u.Name() == user2Name {
						var docRead UserDoc
						require.NoError(t, u.Extra(&docRead))
						require.Equal(t, doc, docRead)
						require.True(t, u.IsActive())
					}
				}
			})

			t.Run("Test update user", func(t *testing.T) {
				opts := &arangodb.UserOptions{
					Active: utils.NewType(false),
				}

				u, err := client.UpdateUser(ctx, user1Name, opts)
				require.NoError(t, err)
				require.NotNil(t, u)
				require.False(t, u.IsActive())
			})

			t.Run("Test replace user", func(t *testing.T) {
				opts := &arangodb.UserOptions{
					Extra: doc,
				}

				u, err := client.ReplaceUser(ctx, user1Name, opts)
				require.NoError(t, err)
				require.NotNil(t, u)

				// Active should be overwritten
				require.True(t, u.IsActive())

				ur, err := client.User(ctx, user1Name)
				require.NoError(t, err)
				require.NotNil(t, ur)

				var docRead UserDoc
				require.NoError(t, ur.Extra(&docRead))
				require.Equal(t, doc, docRead)
			})

			t.Run("Test remove user", func(t *testing.T) {
				require.NoError(t, client.RemoveUser(ctx, user1Name))
				require.NoError(t, client.RemoveUser(ctx, user2Name))
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_UserCreation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		uuid := GenerateUUID("user-creation")

		testCases := map[string]*arangodb.UserOptions{
			"jan1-" + uuid:      nil,
			"george-" + uuid:    {Password: "foo", Active: utils.NewType(false)},
			"candy-" + uuid:     {Password: "ARANGODB_DEFAULT_ROOT_PASSWORD", Active: utils.NewType(true)},
			"joe-" + uuid:       {Extra: map[string]interface{}{"key": "value", "x": 5}},
			"admin@api-" + uuid: nil,
			"測試用例-" + uuid:      nil,
			"測試用例@foo-" + uuid:  nil,
			"_-" + uuid:         nil,
			"/-" + uuid:         nil,
			"jakub/foo-" + uuid: nil,
		}

		for name, options := range testCases {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

				u, err := client.CreateUser(ctx, name, options)
				require.NoError(t, err)
				require.NotNil(t, u)
				require.Equal(t, name, u.Name())

				exist, err := client.UserExists(ctx, name)
				require.NoError(t, err)
				require.True(t, exist)

				ur, err := client.User(ctx, name)
				require.NoError(t, err)
				require.NotNil(t, ur)
				require.Equal(t, name, ur.Name())

				opts := &arangodb.UserOptions{
					Password: "test",
				}
				u, err = client.UpdateUser(ctx, name, opts)
				require.NoError(t, err)

				require.NoError(t, client.RemoveUser(ctx, name))
			})
		}
	})
}
