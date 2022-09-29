//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetServerMetrics tests if Client.Metrics works at all
func TestGetServerMetrics(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()
	skipBelowVersion(c, "3.8", t)

	metrics, err := c.Metrics(ctx)
	require.NoError(t, err)
	require.Contains(t, string(metrics), "arangodb_client_connection_statistics_total_time")
}
