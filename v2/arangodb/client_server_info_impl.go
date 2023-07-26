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
	"net/http"

	"github.com/arangodb/go-driver/v2/arangodb/shared"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/connection"
)

// ServerRole is the role of an arangod server
type ServerRole string

const (
	// ServerRoleSingle indicates that the server is a single-server instance
	ServerRoleSingle ServerRole = "Single"
	// ServerRoleSingleActive indicates that the server is a the leader of a single-server resilient pair
	ServerRoleSingleActive ServerRole = "SingleActive"
	// ServerRoleSinglePassive indicates that the server is a a follower of a single-server resilient pair
	ServerRoleSinglePassive ServerRole = "SinglePassive"
	// ServerRoleDBServer indicates that the server is a dbserver within a cluster
	ServerRoleDBServer ServerRole = "DBServer"
	// ServerRoleCoordinator indicates that the server is a coordinator within a cluster
	ServerRoleCoordinator ServerRole = "Coordinator"
	// ServerRoleAgent indicates that the server is an agent within a cluster
	ServerRoleAgent ServerRole = "Agent"
	// ServerRoleUndefined indicates that the role of the server cannot be determined
	ServerRoleUndefined ServerRole = "Undefined"
)

// ConvertServerRole returns go-driver server role based on ArangoDB role.
func ConvertServerRole(arangoDBRole string) ServerRole {
	switch arangoDBRole {
	case "SINGLE":
		return ServerRoleSingle
	case "PRIMARY":
		return ServerRoleDBServer
	case "COORDINATOR":
		return ServerRoleCoordinator
	case "AGENT":
		return ServerRoleAgent
	default:
		return ServerRoleUndefined
	}
}

func newClientServerInfo(client *client) *clientServerInfo {
	return &clientServerInfo{
		client: client,
	}
}

var _ ClientServerInfo = &clientServerInfo{}

type clientServerInfo struct {
	client *client
}

func (c clientServerInfo) Version(ctx context.Context) (VersionInfo, error) {
	return c.VersionWithOptions(ctx, nil)
}

func (c clientServerInfo) VersionWithOptions(ctx context.Context, opts *GetVersionOptions) (VersionInfo, error) {
	url := connection.NewUrl("_api", "version")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		VersionInfo
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response, opts.modifyRequest)
	if err != nil {
		return VersionInfo{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.VersionInfo, nil
	default:
		return VersionInfo{}, response.AsArangoErrorWithCode(code)
	}
}

// ServerRole returns the role of the server that answers the request.
func (c clientServerInfo) ServerRole(ctx context.Context) (ServerRole, error) {
	url := connection.NewUrl("_admin", "server", "role")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Role                  string `json:"role,omitempty"`
		Mode                  string `json:"mode,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ServerRoleUndefined, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Fallthrough.
	default:
		return ServerRoleUndefined, response.AsArangoErrorWithCode(code)
	}

	role := ConvertServerRole(response.Role)
	if role != ServerRoleSingle {
		return role, nil
	}

	if response.Mode != "resilient" {
		// Single server mode.
		return role, nil
	}

	// Active fail-over mode.
	if err := c.echo(ctx); err != nil {
		if shared.IsNoLeader(err) {
			return ServerRoleSinglePassive, nil
		}

		return ServerRoleUndefined, errors.WithStack(err)
	}

	return ServerRoleSingleActive, nil
}

func (c clientServerInfo) ServerID(ctx context.Context) (string, error) {
	url := connection.NewUrl("_admin", "server", "id")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ID                    string `json:"id,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ID, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

// echo returns what is sent to the server.
func (c clientServerInfo) echo(ctx context.Context) error {
	url := connection.NewUrl("_admin", "echo")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	// Velocypack requires non-empty body for versions < 3.11.
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response, connection.WithBody("echo"))
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

type GetVersionOptions struct {
	// If true, additional details will be returned in response
	// Default false
	Details *bool
}

func (o *GetVersionOptions) modifyRequest(r connection.Request) error {
	if o == nil {
		return nil
	}
	if o.Details != nil {
		r.AddQuery("details", boolToString(*o.Details))
	}
	return nil
}
