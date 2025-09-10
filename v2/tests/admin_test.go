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

package tests

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_ServerMode(t *testing.T) {
	// This test can not run sub-tests parallelly, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			serverMode, err := client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeReadOnly)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeReadOnly, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeDefault)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)
		})
	}, wrapOpts)
}

func Test_ServerID(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			if getTestMode() == string(testModeCluster) {
				id, err := client.ServerID(ctx)
				require.NoError(t, err, "ServerID failed")
				require.NotEmpty(t, id, "Expected ID to be non-empty")
			} else {
				_, err := client.ServerID(ctx)
				require.Error(t, err, "ServerID succeeded, expected error")
			}
		})
	})
}

func Test_Version(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			v, err := client.VersionWithOptions(context.Background(), &arangodb.GetVersionOptions{
				Details: utils.NewType(true),
			})
			require.NoError(t, err)
			require.NotEmpty(t, v.Version)
			require.NotEmpty(t, v.Server)
			require.NotEmpty(t, v.License)
			require.NotZero(t, len(v.Details))
		})
	})
}

func Test_GetSystemTime(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(context.Background(), "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)

			time, err := client.GetSystemTime(context.Background(), db.Name())
			require.NoError(t, err)
			require.NotEmpty(t, time)
			t.Logf("Current time in Unix timestamp with microsecond precision is:%f", time)
		})
	})
}

func Test_GetServerStatus(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(context.Background(), "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)

			resp, err := client.GetServerStatus(context.Background(), db.Name())
			require.NoError(t, err)
			require.NotEmpty(t, resp)
		})
	})
}

func Test_GetDeploymentSupportInfo(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {

			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			resp, err := client.GetDeploymentSupportInfo(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, resp)
			require.NotEmpty(t, resp.Date)
			require.NotEmpty(t, resp.Deployment)
			require.NotEmpty(t, resp.Deployment.Type)
			if serverRole == arangodb.ServerRoleCoordinator {
				require.NotEmpty(t, resp.Deployment.Servers)
			}
			if serverRole == arangodb.ServerRoleSingle {
				require.NotEmpty(t, resp.Host)
			}
		})
	})
}

func Test_GetStartupConfiguration(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			resp, err := client.GetStartupConfiguration(ctx)
			t.Logf("Error type: %T,  Error:%v\n", err, err)

			if err != nil {
				var arangoErr shared.ArangoError
				t.Logf("arangoErr code:%d", arangoErr.Code)
				if errors.As(err, &arangoErr) {
					if arangoErr.Code == 403 || arangoErr.Code == 500 {
						t.Skip("startup configuration API not enabled on this server")
					}
				}
			}
			require.NoError(t, err)
			require.NotEmpty(t, resp)

			configDesc, err := client.GetStartupConfigurationDescription(ctx)
			t.Logf("Error type: %T,  Error:%v\n", err, err)
			if err != nil {
				var arangoErr shared.ArangoError
				t.Logf("arangoErr code:%d", arangoErr.Code)
				if errors.As(err, &arangoErr) {
					if arangoErr.Code == 403 || arangoErr.Code == 500 {
						t.Skip("startup configuration description API not enabled on this server")
					}
				}
				require.NoError(t, err)
			}
			require.NotEmpty(t, configDesc)

			// Assert that certain well-known options exist
			_, hasEndpoint := configDesc["server.endpoint"]
			require.True(t, hasEndpoint, "expected server.endpoint option to be present")

			_, hasAuth := configDesc["server.authentication"]
			require.True(t, hasAuth, "expected server.authentication option to be present")

			// Optionally assert that each entry has a description
			for key, value := range configDesc {
				option, ok := value.(map[string]interface{})
				require.True(t, ok, "expected value for %s to be a map", key)

				_, hasDesc := option["description"]
				require.True(t, hasDesc, "expected option %s to have a description", key)
			}
		})
	})
}

func Test_ReloadRoutingTable(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			err = client.ReloadRoutingTable(ctx, db.Name())
			require.NoError(t, err)
		})
	})
}

func Test_ExecuteAdminScript(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		tests := []struct {
			name   string
			script string
		}{
			{
				name:   "ReturnObject",
				script: "return {hello: 'world'};",
			},
			{
				name: "ReturnNumber",
				script: `
                    var sum = 0;
                    for (var i = 1; i <= 5; i++) {
                        sum += i;
                    }
                    return sum;
                `,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := client.ExecuteAdminScript(ctx, db.Name(), tt.script)
				var arangoErr *shared.ArangoError
				if errors.As(err, &arangoErr) {
					t.Logf("arangoErr code:%d\n", arangoErr.Code)
					if arangoErr.Code == http.StatusNotFound {
						t.Skip("javascript.allow-admin-execute is disabled")
					}
				}
				require.NoError(t, err)

				switch v := result.(type) {
				case map[string]interface{}:
					t.Logf("Got object result: %+v", v)
					require.Contains(t, v, "hello")
				case float64:
					t.Logf("Got number result: %v", v)
					require.Equal(t, float64(15), v)
				default:
					t.Fatalf("Unexpected result type: %T, value: %+v", v, v)
				}
			})
		}
	})
}

func Test_CompactDatabases(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			resp, err := client.CompactDatabases(ctx, nil)
			t.Logf("Error type: %T,  Error:%v\n", err, err)

			if err != nil {
				var arangoErr shared.ArangoError
				t.Logf("arangoErr code:%d", arangoErr.Code)
				if errors.As(err, &arangoErr) {
					if arangoErr.Code == 403 || arangoErr.Code == 500 {
						t.Skip("The endpoint requires superuser access")
					}
				}
				require.NoError(t, err)
			}
			require.NoError(t, err)
			require.Empty(t, resp)

			opts := &arangodb.CompactOpts{
				ChangeLevel:            true,
				CompactBottomMostLevel: false,
			}
			resp, err = client.CompactDatabases(ctx, opts)
			if err != nil {
				var arangoErr shared.ArangoError
				t.Logf("arangoErr code:%d", arangoErr.Code)
				if errors.As(err, &arangoErr) {
					if arangoErr.Code == 403 || arangoErr.Code == 500 {
						t.Skip("The endpoint requires superuser access")
					}
				}
				require.NoError(t, err)
			}
			require.NoError(t, err)
			require.Empty(t, resp)
		})
	})
}
