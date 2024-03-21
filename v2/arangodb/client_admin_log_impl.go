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

// LogLevels is a map of topics to log level.
type LogLevels map[string]string

// LogLevelsGetOptions describes log levels get options.
type LogLevelsGetOptions struct {
	// serverID describes log levels for a specific server ID.
	ServerID ServerID
}

// LogLevelsSetOptions describes log levels set options.
type LogLevelsSetOptions struct {
	// serverID describes log levels for a specific server ID.
	ServerID ServerID
}

// GetLogLevels returns log levels for topics.
func (c *clientAdmin) GetLogLevels(ctx context.Context, opts *LogLevelsGetOptions) (LogLevels, error) {
	url := connection.NewUrl("_admin", "log", "level")

	var response LogLevels
	var mods []connection.RequestModifier
	if opts != nil {
		if len(opts.ServerID) > 0 {
			mods = append(mods, connection.WithQuery("serverId", string(opts.ServerID)))
		}
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response, mods...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		var r *shared.ResponseStruct // nil
		return nil, r.AsArangoErrorWithCode(code)
	}
}

// SetLogLevels sets log levels for a given topics.
func (c *clientAdmin) SetLogLevels(ctx context.Context, logLevels LogLevels, opts *LogLevelsSetOptions) error {
	url := connection.NewUrl("_admin", "log", "level")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	var mods []connection.RequestModifier
	if opts != nil {
		if len(opts.ServerID) > 0 {
			mods = append(mods, connection.WithQuery("serverId", string(opts.ServerID)))
		}
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, logLevels, mods...)
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
