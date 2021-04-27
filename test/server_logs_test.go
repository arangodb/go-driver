//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
