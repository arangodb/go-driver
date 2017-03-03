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

import "context"

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
	return &client{
		conn: conn,
	}, nil
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
