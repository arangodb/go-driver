//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"sort"
	"strings"
)

type ClientServerInfo interface {
	// Version returns version information from the connected database server.
	Version(ctx context.Context) (VersionInfo, error)

	// VersionWithOptions returns version information from the connected database server.
	VersionWithOptions(ctx context.Context, opts *GetVersionOptions) (VersionInfo, error)

	// ServerRole returns the role of the server that answers the request.
	ServerRole(ctx context.Context) (ServerRole, error)

	// ServerID Gets the ID of this server in the cluster.
	// An error is returned when calling this to a server that is not part of a cluster.
	ServerID(ctx context.Context) (string, error)
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

	// Optional additional details. This is returned only if details were requested
	Details map[string]interface{} `json:"details,omitempty"`
}

func (v VersionInfo) IsEnterprise() bool {
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
