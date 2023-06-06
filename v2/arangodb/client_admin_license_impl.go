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

package arangodb

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

// LicenseFeatures describes license's features.
type LicenseFeatures struct {
	// Expires is expiry date as Unix timestamp (seconds since January 1st, 1970 UTC).
	Expires int `json:"expires"`
}

// LicenseStatus describes license's status.
type LicenseStatus string

const (
	// LicenseStatusGood - The license is valid for more than 2 weeks.
	LicenseStatusGood LicenseStatus = "good"
	// LicenseStatusExpired - The license has expired. In this situation, no new Enterprise Edition features can be utilized.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusExpiring - The license is valid for less than 2 weeks.
	LicenseStatusExpiring LicenseStatus = "expiring"
	// LicenseStatusReadOnly - The license is expired over 2 weeks. The instance is now restricted to read-only mode.
	LicenseStatusReadOnly LicenseStatus = "read-only"
)

// License describes license information.
type License struct {
	// Features describes properties of the license.
	Features LicenseFeatures `json:"features"`
	// License is an encrypted license key in Base64 encoding.
	License string `json:"license"`
	// Status is a status of a license.
	Status LicenseStatus `json:"status"`
	// Version is a version of a license.
	Version int `json:"version"`
}

// GetLicense returns license of an ArangoDB deployment.
func (c clientAdmin) GetLicense(ctx context.Context) (License, error) {
	var response struct {
		shared.ResponseStruct `json:",inline"`
		License
	}
	resp, err := connection.CallGet(ctx, c.client.connection, "_admin/license", &response)
	if err != nil {
		return response.License, errors.WithMessage(err, "failed to get license")
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.License, nil
	default:
		return License{}, response.AsArangoErrorWithCode(code)
	}
}
