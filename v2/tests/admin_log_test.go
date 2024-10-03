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
		Parallel: utils.NewT(false),
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
		Parallel: utils.NewT(false),
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
