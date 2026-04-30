//
// DISCLAIMER
//
// Copyright 2023-2026 ArangoDB GmbH, Cologne, Germany
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// Test_DecoderBytes gets plain text response from the server (Prometheus metrics via Client APIs).
func Test_DecoderBytes(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				vi, err := client.Version(ctx)
				require.NoError(t, err)

				serverRole, err := client.ServerRole(ctx)
				require.NoError(t, err)
				var serverId *string
				if serverRole == arangodb.ServerRoleCoordinator {
					sid, err := client.ServerID(ctx)
					require.NoError(t, err)
					serverId = &sid
				}

				var output []byte
				if vi.Version.Major() >= 4 {
					output, err = client.Metrics(ctx, db.Name(), serverId)
				} else {
					output, err = client.GetMetrics(ctx, db.Name(), serverId)
				}
				require.NoError(t, err)
				require.NotNil(t, output)
				// Metric names differ between …/metrics/v2 (3.x) and …/metrics (4.0+); both should expose ArangoDB Prometheus metrics.
				assert.Contains(t, string(output), "arangodb_")
			})
		})
	})
}
