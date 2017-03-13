//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
	"time"

	"github.com/arangodb/go-driver/util"
)

// NewClient creates a new Client based on the given config setting.
func NewClient(config ClientConfig) (Client, error) {
	if config.Connection == nil {
		return nil, WithStack(InvalidArgumentError{Message: "Connection is not set"})
	}
	conn := config.Connection
	if config.Authentication != nil {
		var err error
		conn, err = newAuthenticatedConnection(conn, config.Authentication)
		if err != nil {
			return nil, WithStack(err)
		}
	}
	c := &client{
		conn: conn,
	}
	if config.SynchronizeEndpointsInterval > 0 {
		go c.autoSynchronizeEndpoints(config.SynchronizeEndpointsInterval)
	}
	return c, nil
}

// client implements the Client interface.
type client struct {
	conn Connection
}

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

// SynchronizeEndpoints fetches all endpoints from an ArangoDB cluster and updates the
// connection to use those endpoints.
// When this client is connected to a single server, nothing happens.
// When this client is connected to a cluster of servers, the connection will be updated to reflect
// the layout of the cluster.
func (c *client) SynchronizeEndpoints(ctx context.Context) error {
	role, err := c.role(ctx)
	if err != nil {
		return WithStack(err)
	}
	if role == "SINGLE" {
		// Standalone server, do nothing
		return nil
	}

	// Cluster mode, fetch endpoints
	ch, err := c.clusterHealth(ctx)
	if err != nil {
		return WithStack(err)
	}
	var endpoints []string
	for _, sh := range ch.Health {
		if sh.Role != "Coordinator" {
			continue
		}
		ep := util.FixupEndpointURLScheme(sh.Endpoint)
		endpoints = append(endpoints, ep)
	}

	// Update connection
	if err := c.conn.UpdateEndpoints(endpoints); err != nil {
		return WithStack(err)
	}

	return nil
}

// autoSynchronizeEndpoints performs automatic endpoint synchronization.
func (c *client) autoSynchronizeEndpoints(interval time.Duration) {
	for {
		// SynchronizeEndpoints endpoints
		c.SynchronizeEndpoints(nil)

		// Wait a bit
		time.Sleep(interval)
	}
}

type roleResponse struct {
	Role string `json:"role,omitempty"`
}

// role returns the role of the server that answers the request.
func (c *client) role(ctx context.Context) (string, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/role")
	if err != nil {
		return "", WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	var data roleResponse
	if err := resp.ParseBody("", &data); err != nil {
		return "", WithStack(err)
	}
	return data.Role, nil
}

type healthResponse struct {
	Health map[string]serverHealth `json:"Health,omitempty"`
}

type serverHealth struct {
	Endpoint string `json:"Endpoint,omitempty"`
	Role     string `json:"Role,omitempty"`
	Status   string `json:"Status,omitempty"`
}

// clusterHealth returns the health information about a cluster.
func (c *client) clusterHealth(ctx context.Context) (healthResponse, error) {
	req, err := c.conn.NewRequest("GET", "_admin/cluster/health")
	if err != nil {
		return healthResponse{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return healthResponse{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return healthResponse{}, WithStack(err)
	}
	var data healthResponse
	if err := resp.ParseBody("", &data); err != nil {
		return healthResponse{}, WithStack(err)
	}
	return data, nil
}
