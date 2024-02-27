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
					Active: newBool(false),
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
	})
}

func Test_UserCreation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {

		testCases := map[string]*arangodb.UserOptions{
			"jan1":      nil,
			"george":    {Password: "foo", Active: newBool(false)},
			"candy":     {Password: "ARANGODB_DEFAULT_ROOT_PASSWORD", Active: newBool(true)},
			"joe":       {Extra: map[string]interface{}{"key": "value", "x": 5}},
			"admin@api": nil,
			"測試用例":      nil,
			"測試用例@foo":  nil,
			"_":         nil,
			"/":         nil,
			"jakub/foo": nil,
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
	}, WrapOptions{
		Parallel: newBool(false),
	})
}
