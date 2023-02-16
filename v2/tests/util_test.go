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

package tests

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/stretchr/testify/require"
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

func requireMode(t *testing.T, mode mode) {
	if getTestMode() != string(mode) {
		t.Skipf("the test requires %s mode", mode)
	}
}

func requireClusterMode(t *testing.T) {
	requireMode(t, testModeCluster)
}

func requireSingleMode(t *testing.T) {
	requireMode(t, testModeSingle)
}

func requireResilientSingleMode(t *testing.T) {
	requireMode(t, testModeResilientSingle)
}

func skipNoEnterprise(c arangodb.Client, ctx context.Context, t *testing.T) {
	version, err := c.Version(ctx)
	require.NoError(t, err)

	if !version.IsEnterprise() {
		t.Skip("Skipping test, no enterprise version")
	}
}

func skipBelowVersion(c arangodb.Client, ctx context.Context, version arangodb.Version, t *testing.T) arangodb.VersionInfo {
	x, err := c.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", err)
	}
	if x.Version.CompareTo(version) < 0 {
		t.Skipf("Skipping below version '%s', got version '%s'", version, x.Version)
	}
	return x
}
