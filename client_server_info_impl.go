//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import (
	"context"
)

// Version returns version information from the connected database server.
func (c *client) Version(ctx context.Context) (VersionInfo, error) {
	req, err := c.conn.NewRequest("GET", "_api/version")
	if err != nil {
		return VersionInfo{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return VersionInfo{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return VersionInfo{}, WithStack(err)
	}
	var data VersionInfo
	if err := resp.ParseBody("", &data); err != nil {
		return VersionInfo{}, WithStack(err)
	}
	return data, nil
}

// ServerRoleInfo contains information on the role played by a single ArangoDB server.
type roleResponse struct {
	// Role of the server within a cluster
	Role string `json:"role,omitempty"`
	Mode string `json:"mode,omitempty"`
}

// AsServerRole converts the response into a ServerRole
func (r roleResponse) AsServerRole() ServerRole {
	switch r.Role {
	case "SINGLE":
		switch r.Mode {
		case "resilient":
			return ServerRoleSingleResilient
		default:
			return ServerRoleSingle
		}
	case "PRIMARY":
		return ServerRoleDBServer
	case "COORDINATOR":
		return ServerRoleCoordinator
	case "AGENT":
		return ServerRoleAgent
	case "UNDEFINED":
		return ServerRoleUndefined
	default:
		return ServerRoleUndefined
	}
}

// ServerRole returns the role of the server that answers the request.
func (c *client) ServerRole(ctx context.Context) (ServerRole, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/role")
	if err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	var data roleResponse
	if err := resp.ParseBody("", &data); err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	return data.AsServerRole(), nil
}
