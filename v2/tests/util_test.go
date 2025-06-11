//
// DISCLAIMER
//
// Copyright 2021-2024 ArangoDB GmbH, Cologne, Germany
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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func getTestMode() string {
	return strings.TrimSpace(os.Getenv("TEST_MODE"))
}

type mode string

const (
	testModeCluster         mode = "cluster"
	testModeResilientSingle mode = "resilientsingle"
	testModeSingle          mode = "single"
)

// requireMode skips current test if it is not in given modes.
func requireMode(t testing.TB, modes ...mode) {
	testMode := getTestMode()
	for _, mode := range modes {
		if testMode == string(mode) {
			return
		}
	}

	t.Skipf("test is in \"%s\" mode, but it requires one of \"%s\"", testMode, modes)
}

func requireClusterMode(t testing.TB) {
	requireMode(t, testModeCluster)
}

func requireSingleMode(t testing.TB) {
	requireMode(t, testModeSingle)
}

func requireResilientSingleMode(t testing.TB) {
	requireMode(t, testModeResilientSingle)
}

func skipResilientSingleMode(t testing.TB) {
	requireMode(t, testModeCluster, testModeSingle)
}

func requireExtraDBFeatures(t testing.TB) {
	if os.Getenv("ENABLE_DATABASE_EXTRA_FEATURES") != "true" {
		t.Skip("Skipping test, extra database features are not enabled")
	}
}

func skipNoEnterprise(c arangodb.Client, ctx context.Context, t testing.TB) {
	version, err := c.Version(ctx)
	require.NoError(t, err)

	if !version.IsEnterprise() {
		t.Skip("Skipping test, no enterprise version")
	}
}

// skipFromVersion skips test if DB version is equal or above given version
func skipFromVersion(c arangodb.Client, ctx context.Context, version arangodb.Version, t testing.TB) arangodb.VersionInfo {
	x, err := c.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", err)
	}
	if x.Version.CompareTo(version) > 0 || x.Version.CompareTo(version) == 0 {
		t.Skipf("Skipping above version '%s', got version '%s'", version, x.Version)
	}
	return x
}

func skipBelowVersion(c arangodb.Client, ctx context.Context, version arangodb.Version, t testing.TB) arangodb.VersionInfo {
	x, err := c.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", err)
	}
	if x.Version.CompareTo(version) < 0 {
		t.Skipf("Skipping below version '%s', got version '%s'", version, x.Version)
	}
	return x
}

// skipBetweenVersions skips test if DB version is in interval (close-ended)
func skipBetweenVersions(c arangodb.Client, ctx context.Context, minVersion, maxVersion arangodb.Version, t *testing.T) arangodb.VersionInfo {
	x, err := c.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", err)
	}
	if x.Version.CompareTo(minVersion) >= 0 && x.Version.CompareTo(maxVersion) <= 0 {
		t.Skipf("Skipping between version '%s' and '%s': got version '%s'", minVersion, maxVersion, x.Version)
	}
	return x
}


// skipVersionNotInRange skips the test if the current server version is less than
// the min version or higher/equal max version
func skipVersionNotInRange(c arangodb.Client, ctx context.Context, minVersion, maxVersion arangodb.Version, t testing.TB) arangodb.VersionInfo {
	x, err := c.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", err)
	}
	if x.Version.CompareTo(minVersion) < 0 {
		t.Skipf("Skipping below version '%s', got version '%s'", minVersion, x.Version)
	}
	if x.Version.CompareTo(maxVersion) >= 0 {
		t.Skipf("Skipping above version '%s', got version '%s'", maxVersion, x.Version)
	}
	return x
}
