//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

func Test_License(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
			version := skipBelowVersion(client, ctx, "3.10.0", t)

			license, err := client.GetLicense(ctx)
			require.NoError(t, err)
			if license.Version > 0 {
				assert.Contains(t,
					[]arangodb.LicenseStatus{
						arangodb.LicenseStatusGood,
						arangodb.LicenseStatusExpiring,
						arangodb.LicenseStatusReadOnly,
						arangodb.LicenseStatusExpired,
					},
					license.Status,
					"license status should be a known Enterprise status when version > 0",
				)
				assert.NotEmpty(t, license.Hash, "license hash should be present when version > 0")
				assert.GreaterOrEqual(t, license.Features.Expires, 0, "license expiry must be non-negative")
				assert.Nil(t, license.DiskUsage, "diskUsage should not be present when an Enterprise license is applied")
			} else if version.Version.CompareTo("3.12.5") >= 0 {
				require.NotNil(t, license.DiskUsage, "diskUsage should be present for Community/no-license state on 3.12.5+")
				assert.NotNil(t, license.DiskUsage.BytesLimit, "diskUsage.bytesLimit should be present")
				assert.NotNil(t, license.DiskUsage.BytesUsed, "diskUsage.bytesUsed should be present")
				assert.Equalf(t, arangodb.LicenseStatus(""), license.Status, "license status should be empty when diskUsage is returned")
				assert.Empty(t, license.License, "license string should be empty when diskUsage is returned")
			}

			if license.DiskUsage != nil {
				assert.Equalf(t, 0, license.Version, "license version should be 0 when diskUsage is returned")
				assert.NotNil(t, license.DiskUsage.Status, "diskUsage.status should be present when diskUsage is returned")
				if license.DiskUsage.Status != nil {
					assert.NotEmpty(t, *license.DiskUsage.Status, "diskUsage.status should not be empty when diskUsage is returned")
				}
			}
		})
	})
}
