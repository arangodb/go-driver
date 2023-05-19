//
// DISCLAIMER
//
// Copyright 2021-2023 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

// TestServerLogs tests if logs are parsed.
func TestServerLogs(t *testing.T) {
	c := createClientFromEnv(t, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.8.0"))

	logs, err := c.Logs(ctx)
	require.NoError(t, err)
	for _, l := range logs.Messages {
		if strings.Contains(l.Message, "is ready for business") {
			t.Logf("Line `is ready for business` found in logs")
			return
		}
	}

	t.Fatalf("Line `is ready for business` not found in logs")
}

// Test_LogLevels tests log levels.
func Test_LogLevels(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	result, err := c.GetLogLevels(ctx, nil)
	require.NoError(t, err)

	if len(result) == 0 {
		t.Skip("test can not proceed without log levels")
	}
	var topic, level string
	for topic, level = range result {
		// Get first topic from map of topics.
		break
	}

	level = changeLogLevel(level)
	result[topic] = level
	err = c.SetLogLevels(ctx, result, nil)
	require.NoError(t, err)

	result1, err := c.GetLogLevels(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, result, result1)
}

// Test_LogLevelsForServers tests log levels for on specific server.
func Test_LogLevelsForServers(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()
	skipBelowVersion(c, "3.10.2", t)
	skipNoCluster(c, t)

	cl, err := c.Cluster(ctx)
	require.NoError(t, err)

	health, err := cl.Health(ctx)
	require.NoError(t, err)

	var changed int
	servers := make(map[driver.ServerID]driver.LogLevels)
	for serverID, health := range health.Health {
		if health.Role == driver.ServerRoleAgent {
			continue
		}

		opts := driver.LogLevelsGetOptions{
			ServerID: serverID,
		}

		logLevels, err := c.GetLogLevels(ctx, &opts)
		require.NoError(t, err)

		if changed == 0 {
			// Change log level for random topic, but only for one server.
			changed++
			for randomTopic, level := range logLevels {
				logLevels[randomTopic] = changeLogLevel(level)
				opts := driver.LogLevelsSetOptions{
					ServerID: serverID,
				}

				err = c.SetLogLevels(ctx, logLevels, &opts)
				require.NoError(t, err)

				break
			}
		}

		servers[serverID] = logLevels
	}
	require.Equal(t, 1, changed, "only one server should change log levels")

	// Check if log levels have changed for a specific server.
	for serverID, health := range health.Health {
		if health.Role == driver.ServerRoleAgent {
			continue
		}

		opts := driver.LogLevelsGetOptions{
			ServerID: serverID,
		}

		result, err := c.GetLogLevels(ctx, &opts)
		require.NoError(t, err)

		s, ok := servers[serverID]
		require.True(t, ok)

		require.Equal(t, s, result)
	}
}

// Change log level from DEBUG to INFO or from something else to DEBUG.
func changeLogLevel(l string) string {
	if l != "DEBUG" {
		return "DEBUG"
	}

	return "INFO"
}
