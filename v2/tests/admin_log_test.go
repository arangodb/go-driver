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
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// Test_LogLevels tests log levels.
func Test_LogLevels(t *testing.T) {
	// This test cannot run subtests parallel, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {

			logLevels, err := client.GetLogLevels(ctx, nil)
			require.NoError(t, err)
			if len(logLevels) == 0 {
				t.Skip("test can not proceed without log levels")
			}

			var topic, level string
			for topic, level = range logLevels {
				// Get a first topic from the map of topics.
				break
			}

			level = changeLogLevel(level)
			logLevels[topic] = level
			err = client.SetLogLevels(ctx, logLevels, nil)
			require.NoError(t, err)

			newLogLevels, err := client.GetLogLevels(ctx, nil)
			require.NoError(t, err)
			require.Equal(t, logLevels, newLogLevels)
		})
	}, wrapOpts)
}

// Test_LogLevelsForServers tests log levels for on specific server.
func Test_LogLevelsForServers(t *testing.T) {
	requireMode(t, testModeCluster, testModeResilientSingle)

	// This test cannot run subtests parallel, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			skipBelowVersion(client, ctx, "3.10.2", t)

			WaitForHealthyCluster(t, client, time.Minute, false)
			health, err := client.Health(ctx)
			require.NoError(t, err)

			changed := 0
			servers := make(map[arangodb.ServerID]arangodb.LogLevels)
			for serverID, serverHealth := range health.Health {
				if serverHealth.Role == arangodb.ServerRoleAgent {
					continue
				}

				opts := arangodb.LogLevelsGetOptions{
					ServerID: serverID,
				}

				logLevels, err := client.GetLogLevels(ctx, &opts)
				require.NoError(t, err)

				if changed == 0 {
					// Change log level for a random topic, but only for one server.
					changed++
					for randomTopic, level := range logLevels {
						logLevels[randomTopic] = changeLogLevel(level)
						optsSet := arangodb.LogLevelsSetOptions{
							ServerID: serverID,
						}

						err = client.SetLogLevels(ctx, logLevels, &optsSet)
						require.NoError(t, err)
						break
					}
				}
				servers[serverID] = logLevels
			}
			require.Greater(t, len(servers), 0, "no servers found", servers)
			require.Equal(t, 1, changed, "only one server should change log levels")

			// Check if log levels have changed for a specific server.
			for serverID, health := range health.Health {
				if health.Role == arangodb.ServerRoleAgent {
					continue
				}

				opts := arangodb.LogLevelsGetOptions{
					ServerID: serverID,
				}

				result, err := client.GetLogLevels(ctx, &opts)
				require.NoError(t, err)

				s, ok := servers[serverID]
				require.True(t, ok)
				require.Equal(t, s, result)
			}
		})
	}, wrapOpts)
}

// Change log level from DEBUG to INFO or from something else to DEBUG.
func changeLogLevel(l string) string {
	if l != "DEBUG" {
		return "DEBUG"
	}

	return "INFO"
}

func Test_Logs(t *testing.T) {
	// This test cannot run subtests parallel, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
			skipBelowVersion(client, ctx, "3.8.0", t)

			logsResp, err := client.Logs(ctx, &arangodb.AdminLogEntriesOptions{
				Start:  0,
				Offset: 0,
				Upto:   "3",
				Sort:   "asc",
			})
			require.NoError(t, err)
			require.NotNil(t, logsResp)

			_, err = client.Logs(ctx, &arangodb.AdminLogEntriesOptions{
				Start:  0,
				Offset: 0,
				Upto:   "3",
				Sort:   "asc",
				Level:  utils.NewType("DEBUG"),
			})
			require.Error(t, err)
		})
	}, wrapOpts)
}

func Test_DeleteLogLevels(t *testing.T) {
	// This test cannot run subtests parallel, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
			skipBelowVersion(client, ctx, "3.12.1", t)
			// Role check
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole: %s", serverRole)

			var serverId *string
			if serverRole == arangodb.ServerRoleCoordinator {
				serverID, err := client.ServerID(ctx)
				require.NoError(t, err)
				serverId = &serverID
			}

			logsResp, err := client.DeleteLogLevels(ctx, serverId)
			require.NoError(t, err)
			require.NotNil(t, logsResp)
		})
	}, wrapOpts)
}

func Test_StructuredLogSettings(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
			skipBelowVersion(client, ctx, "3.12.0", t)

			opts := arangodb.LogSettingsOptions{
				Database: utils.NewType(true),
			}
			modifiedResp, err := client.UpdateStructuredLogSettings(ctx, &opts)
			require.NoError(t, err)
			require.NotEmpty(t, modifiedResp)
			require.NotNil(t, modifiedResp.Database)
			require.Equal(t, *modifiedResp.Database, *opts.Database)

			getResp, err := client.GetStructuredLogSettings(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, getResp)
			require.NotNil(t, getResp.Database)
			require.Equal(t, *getResp.Database, *opts.Database)
		})
	})
}

func Test_GetRecentAPICalls(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
			skipBelowVersion(client, ctx, "3.12.5-2", t)

			resp, err := client.Version(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, resp)
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)

			recentApisResp, err := client.GetRecentAPICalls(ctx, db.Name())
			require.NoError(t, err)
			require.NotEmpty(t, recentApisResp)
		})
	})
}

func Test_GetMetrics(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {

			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)
			// Role check
			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			t.Logf("ServerRole: %s", serverRole)

			var serverId *string
			if serverRole == arangodb.ServerRoleCoordinator {
				serverID, err := client.ServerID(ctx)
				require.NoError(t, err)
				serverId = &serverID
			}

			metricsResp, err := client.GetMetrics(ctx, db.Name(), serverId)
			require.NoError(t, err)
			require.NotEmpty(t, metricsResp)
		})
	})
}
