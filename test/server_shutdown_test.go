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
// Author Tomasz Mielech
//

package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServerShutdown tests the graceful shutdown on the coordinator.
func TestServerShutdown(t *testing.T) {
	enabled := os.Getenv("TEST_ENABLE_SHUTDOWN")

	if enabled != "on" && enabled != "1" {
		t.Skipf("TEST_ENABLE_SHUTDOWN is not set")
	}

	c := createClientFromEnv(t, true)
	ctx := context.Background()
	testing.Short()

	// It must be a cluster
	versionCheck := EnsureVersion(t, ctx, c).Cluster()

	// Check required version.
	if !isGracefulShutdownAvailable(versionCheck) {
		t.Skipf("Skipping because version %s is not sufficient", versionCheck.version)
	}

	// Shutdown the coordinator.
	err := c.ShutdownV2(ctx, false, true)
	require.NoError(t, err, "can not shutdown the coordinator")

	// Wait one minute for the coordinator shutdown.
	ctxTimeout, _ := context.WithTimeout(ctx, time.Minute)
	for {
		info, err := c.ShutdownInfoV2(ctxTimeout)
		require.NoError(t, err, "can not fetch shutdown progress information")
		if info.AllClear {
			break
		}
		require.NoError(t, ctx.Err(), "shutdown coordinator timeout")
	}
}

// isGracefulShutdownAvailable returns true since versions: v3.7.12, v3.8.1, v3.9.0.
func isGracefulShutdownAvailable(versionCheck VersionCheck) bool {
	if versionCheck.version.Major() > 3 {
		return true
	}

	minor := versionCheck.version.Minor()
	if minor < 7 {
		return false
	}

	if minor == 7 {
		if versionCheck.version.CompareTo("3.7.12") < 0 {
			return false
		}
	} else if minor == 8 {
		if versionCheck.version.CompareTo("3.8.1") < 0 {
			return false
		}
	}

	return true
}
