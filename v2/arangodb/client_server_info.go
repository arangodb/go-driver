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

package arangodb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type ClientServerInfo interface {
	// Version returns version information from the connected database server.
	// Use WithDetails to configure a context that will include additional details in the return VersionInfo.
	Version(ctx context.Context) (VersionInfo, error)
}

// VersionInfo describes the version of a database server.
type VersionInfo struct {
	// This will always contain "arango"
	Server string `json:"server,omitempty"`
	//  The server version string. The string has the format "major.minor.sub".
	// Major and minor will be numeric, and sub may contain a number or a textual version.
	Version Version `json:"version,omitempty"`
	// Type of license of the server
	License string `json:"license,omitempty"`
	// Optional additional details. This is returned only if the context is configured using WithDetails.
	Details map[string]interface{} `json:"details,omitempty"`
}

func (v *VersionInfo) IsEnterprise() bool {
	return v.License == "enterprise"
}

// String creates a string representation of the given VersionInfo.
func (v VersionInfo) String() string {
	result := fmt.Sprintf("%s, version %s, license %s", v.Server, v.Version, v.License)
	if len(v.Details) > 0 {
		lines := make([]string, 0, len(v.Details))
		for k, v := range v.Details {
			lines = append(lines, fmt.Sprintf("%s: %v", k, v))
		}
		sort.Strings(lines)
		result = result + "\n" + strings.Join(lines, "\n")
	}
	return result
}
