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
)

type serverModeResponse struct {
	Mode ServerMode `json:"mode"`
}

type serverModeRequest struct {
	Mode ServerMode `json:"mode"`
}

type clusterAdvertisedEndpointRequest struct {
	Endpoint string `json:"endpoint"`
}

// ServerMode returns the current mode in which the server/cluster is operating.
// This call needs ArangoDB 3.3 and up.
func (c *client) ServerMode(ctx context.Context) (ServerMode, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/mode")
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	var result serverModeResponse
	if err := resp.ParseBody("", &result); err != nil {
		return "", WithStack(err)
	}
	return result.Mode, nil
}

// SetServerMode changes the current mode in which the server/cluster is operating.
// This call needs a client that uses JWT authentication.
// This call needs ArangoDB 3.3 and up.
func (c *client) SetServerMode(ctx context.Context, mode ServerMode) error {
	req, err := c.conn.NewRequest("PUT", "_admin/server/mode")
	if err != nil {
		return WithStack(err)
	}
	input := serverModeRequest{
		Mode: mode,
	}
	req, err = req.SetBody(input)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Shutdown a specific server, optionally removing it from its cluster.
func (c *client) Shutdown(ctx context.Context, removeFromCluster bool) error {
	req, err := c.conn.NewRequest("DELETE", "_admin/shutdown")
	if err != nil {
		return WithStack(err)
	}
	if removeFromCluster {
		req.SetQuery("remove_from_cluster", "1")
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// SetAdvertisedEndpoint updates the endpoint specific advertised endpoint
func (c *client) SetAdvertisedEndpoint(ctx context.Context, endpoint string) error {
	req, err := c.conn.NewRequest("PUT", "_admin/cluster/advertised-endpoint")
	if err != nil {
		return WithStack(err)
	}
	body := clusterAdvertisedEndpointRequest{
		Endpoint: endpoint,
	}
	_, err = req.SetBody(&body)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// AdvertisedEndpoint returns the endpoint specific advertised endpoint
func (c *client) AdvertisedEndpoint(ctx context.Context) (string, error) {
	req, err := c.conn.NewRequest("GET", "_admin/cluster/advertised-endpoint")
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	var body clusterAdvertisedEndpointRequest
	if err := resp.ParseBody("", &body); err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	return body.Endpoint, nil
}
