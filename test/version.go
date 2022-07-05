//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

type mode string

const (
	cluster mode = "cluster"
	single  mode = "single"
)

func EnsureVersion(t *testing.T, ctx context.Context, c driver.Client) VersionCheck {
	version, err := c.Version(ctx)
	if err != nil {
		require.NoError(t, err, "Version check failed")
	}

	m := cluster

	_, err = c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		m = single
	} else if err != nil {
		require.NoError(t, err)
	}

	return VersionCheck{
		t:          t,
		version:    version.Version,
		enterprise: version.IsEnterprise(),
		mode:       m,
	}
}

type VersionCheck struct {
	t *testing.T

	version    driver.Version
	enterprise bool

	mode mode
}

func (v VersionCheck) CheckVersion(check VersionChecker) VersionCheck {
	v.t.Logf("Version check: %s", check.String(v.version))
	if !check.Check(v.version) {
		v.t.Skipf("Version check failed: %s", check.String(v.version))
	}

	return v
}

func AbovePatchRelease(version driver.Version) VersionChecker {
	currMinor := driver.Version(fmt.Sprintf("%d.%d.0", version.Major(), version.Minor()))
	nextMinor := driver.Version(fmt.Sprintf("%d.%d.0", version.Major(), version.Minor()+1))

	return LT.Than(currMinor).Or(GE.Than(nextMinor)).Or(LT.Than(nextMinor).And(GE.Than(version)))
}

func BelowPatchRelease(version driver.Version) VersionChecker {
	currMinor := driver.Version(fmt.Sprintf("%d.%d.0", version.Major(), version.Minor()))
	nextMinor := driver.Version(fmt.Sprintf("%d.%d.0", version.Major(), version.Minor()+1))

	return LT.Than(currMinor).Or(GE.Than(nextMinor)).Or(LT.Than(nextMinor).And(LT.Than(version)))
}

func MinimumVersion(version driver.Version) VersionChecker {
	return GE.Than(version)
}

func (v VersionCheck) Enterprise() VersionCheck {
	v.t.Logf("Enterprise version required")
	if !v.enterprise {
		v.t.Skipf("Required enterprise version")
	}
	return v
}

func (v VersionCheck) Community() VersionCheck {
	v.t.Logf("Community version required")
	if !v.enterprise {
		v.t.Skipf("Required community version")
	}
	return v
}

func (v VersionCheck) Cluster() VersionCheck {
	v.t.Logf("Cluster mode required")
	if v.mode != cluster {
		v.t.Skipf("Required cluster mode, got %s", v.mode)
	}
	return v
}

func (v VersionCheck) NotCluster() VersionCheck {
	v.t.Logf("Skipping cluster mode")
	if v.mode == cluster {
		v.t.Skipf("Test should not run on cluster")
	}
	return v
}
