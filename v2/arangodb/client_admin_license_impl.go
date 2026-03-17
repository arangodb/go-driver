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
	// LicenseStatusGood - The license is valid for more than 1 week.
	LicenseStatusGood LicenseStatus = "good"

	// LicenseStatusExpired - Not returned by current server versions; kept for backward compatibility.
	// With managed/activation licenses the server transitions good → read-only without "expired".
	LicenseStatusExpired LicenseStatus = "expired"

	// LicenseStatusExpiring - The license is close to expiry (less than 1 week).
	LicenseStatusExpiring LicenseStatus = "expiring"

	// LicenseStatusReadOnly - The license has expired; the instance is restricted to read-only mode.
	LicenseStatusReadOnly LicenseStatus = "read-only"
)

// LicenseDiskUsage describes Community Edition dataset limit usage details.
// Present only when no Enterprise Edition license is applied.
type LicenseDiskUsage struct {
	BytesLimit           *uint64 `json:"bytesLimit,omitempty"`
	BytesUsed            *uint64 `json:"bytesUsed,omitempty"`
	LimitReached         *bool   `json:"limitReached,omitempty"`
	SecondsUntilReadOnly *int    `json:"secondsUntilReadOnly,omitempty"`
	SecondsUntilShutDown *int    `json:"secondsUntilShutDown,omitempty"`
	Status               *string `json:"status,omitempty"`
}

// License describes license information.
type License struct {
	// Features describe properties of the license.
	Features LicenseFeatures `json:"features"`

	// License is an encrypted license key in Base64 encoding.
	License string `json:"license,omitempty"`

	// Status is a status of a license.
	Status LicenseStatus `json:"status,omitempty"`

	// Version is a version of a license.
	Version int `json:"version"`

	// Hash The hash value of the license.
	Hash string `json:"hash,omitempty"`

	// DiskUsage provides dataset usage/limit information in Community Edition.
	// It is typically absent when an Enterprise Edition license is applied.
	DiskUsage *LicenseDiskUsage `json:"diskUsage,omitempty"`

	// Upgrading indicates that a license update is currently being propagated.
	// This field is optional and may be omitted by the server.
	Upgrading *bool `json:"upgrading,omitempty"`
}

func (c *clientAdmin) GetLicense(ctx context.Context) (License, error) {
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

func (c *clientAdmin) SetLicense(ctx context.Context, license string, force bool) error {
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	var modifiers []connection.RequestModifier
	if force {
		modifiers = append(modifiers, connection.WithQuery("force", "true"))
	}

	resp, err := connection.CallPut(ctx, c.client.connection, "_admin/license", &response, license, modifiers...)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}
