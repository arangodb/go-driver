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
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

func EnsureVersion(t *testing.T, ctx context.Context, c driver.Client) VersionCheck {
	version, err := c.Version(ctx)
	if err != nil {
		require.NoError(t, err, "Version check failed")
	}

	return VersionCheck{
		t:          t,
		version:    version.Version,
		enterprise: version.IsEnterprise(),
	}
}

type VersionCheck struct {
	t          *testing.T
	version    driver.Version
	enterprise bool
}

func (v VersionCheck) MinimumVersion(version driver.Version) VersionCheck {
	v.t.Logf("Minimum version required: %s", version)
	if v.version.CompareTo(version) < 0 {
		v.t.Skipf("Required version ArangoDB(%s) >= Expected(%s)", v.version, version)
	}
	return v
}

func (v VersionCheck) MaximumVersion(version driver.Version) VersionCheck {
	v.t.Logf("Maximum version required: %s", version)
	if v.version.CompareTo(version) > 0 {
		v.t.Skipf("Required version ArangoDB(%s) <= Expected(%s)", v.version, version)
	}
	return v
}

func (v VersionCheck) AboveVersion(version driver.Version) VersionCheck {
	v.t.Logf("Above version required: %s", version)
	if v.version.CompareTo(version) <= 0 {
		v.t.Skipf("Required version ArangoDB(%s) > Expected(%s)", v.version, version)
	}
	return v
}

func (v VersionCheck) BelowVersion(version driver.Version) VersionCheck {
	v.t.Logf("Below version required: %s", version)
	if v.version.CompareTo(version) >= 0 {
		v.t.Skipf("Required version ArangoDB(%s) < Expected(%s)", v.version, version)
	}
	return v
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
