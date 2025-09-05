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
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
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
