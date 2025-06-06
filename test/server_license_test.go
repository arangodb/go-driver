//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

// Test_License tests ArangoDB license.
func Test_License(t *testing.T) {
	c := createClient(t, nil)
	ctx := context.Background()
	skipVersionNotInRange(c, "3.10.0", "3.12.4", t)

	version, err := c.Version(ctx)
	require.NoError(t, err)

	license, err := c.GetLicense(ctx)
	require.NoError(t, err)

	if version.IsEnterprise() {
		assert.Equalf(t, driver.LicenseStatusExpiring, license.Status, "by default status should be expiring")
		assert.Equalf(t, 1, license.Version, "excpected version should be 1")
	} else {
		assert.Equalf(t, driver.LicenseStatus(""), license.Status, "license status should be empty")
		assert.Equalf(t, 0, license.Version, "license version should be empty")
	}
}
