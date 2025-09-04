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
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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

func defaultAdminLogEntriesOptions() *AdminLogEntriesOptions {
	return &AdminLogEntriesOptions{
		Start:  0,
		Offset: 0,
		Upto:   "info",
		Sort:   "asc",
	}
}

func (c *clientAdmin) formServerLogEntriesParams(opts *AdminLogEntriesOptions) ([]connection.RequestModifier, error) {
	var mods []connection.RequestModifier
	if opts == nil {
		opts = defaultAdminLogEntriesOptions()
	}

	if opts.Level != nil && opts.Upto != "" {
		return nil, errors.New("parameters 'level' and 'upto' cannot be used together")
	}

	if opts.Upto != "" {
		mods = append(mods, connection.WithQuery("upto", opts.Upto))
	}
	if opts.Level != nil && *opts.Level != "" {
		mods = append(mods, connection.WithQuery("level", *opts.Level))
	}
	if opts.Size != nil {
		mods = append(mods, connection.WithQuery("size", fmt.Sprintf("%d", *opts.Size)))
	}
	if opts.Search != nil && *opts.Search != "" {
		mods = append(mods, connection.WithQuery("search", *opts.Search))
	}
	if opts.Sort != "" {
		mods = append(mods, connection.WithQuery("sort", opts.Sort))
	}
	if opts.ServerId != nil && *opts.ServerId != "" {
		mods = append(mods, connection.WithQuery("serverId", *opts.ServerId))
	}
	if opts.Start >= 0 {
		mods = append(mods, connection.WithQuery("start", strconv.Itoa(opts.Start)))
	}
	if opts.Offset >= 0 {
		mods = append(mods, connection.WithQuery("offset", strconv.Itoa(opts.Offset)))
	}
	return mods, nil
}

// Logs retrieve logs from server in ArangoDB 3.8.0+ format
func (c *clientAdmin) Logs(ctx context.Context, queryParams *AdminLogEntriesOptions) (AdminLogEntriesResponse, error) {
	url := connection.NewUrl("_admin", "log", "entries")

	var response struct {
		shared.ResponseStruct   `json:",inline"`
		AdminLogEntriesResponse `json:",inline"`
	}
	mods, err := c.formServerLogEntriesParams(queryParams)
	if err != nil {
		return AdminLogEntriesResponse{}, err
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response, mods...)
	if err != nil {
		return AdminLogEntriesResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.AdminLogEntriesResponse, nil
	default:
		return AdminLogEntriesResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// DeleteLogLevels is for reset the server log levels from server in ArangoDB 3.12.1+ format
func (c *clientAdmin) DeleteLogLevels(ctx context.Context, serverId *string) (LogLevelResponse, error) {
	url := connection.NewUrl("_admin", "log", "level")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		LogLevelResponse      `json:",inline"`
	}

	var mods []connection.RequestModifier
	if serverId != nil {
		mods = append(mods, connection.WithQuery("serverId", *serverId))
	}

	resp, err := connection.CallDelete(ctx, c.client.connection, url, &response, mods...)
	if err != nil {
		return LogLevelResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.LogLevelResponse, nil
	default:
		return LogLevelResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// GetStructuredLogSettings returns the server's current structured log settings in ArangoDB 3.12.0+ format.
func (c *clientAdmin) GetStructuredLogSettings(ctx context.Context) (LogSettingsOptions, error) {
	url := connection.NewUrl("_admin", "log", "structured")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		LogSettingsOptions    `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return LogSettingsOptions{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.LogSettingsOptions, nil
	default:
		return LogSettingsOptions{}, response.AsArangoErrorWithCode(code)
	}
}

func formStructuredLogParams(opt *LogSettingsOptions) map[string]bool {
	params := map[string]bool{}
	if opt.Database != nil {
		params["database"] = *opt.Database
	}
	if opt.Url != nil {
		params["url"] = *opt.Url
	}
	if opt.Username != nil {
		params["username"] = *opt.Username
	}
	return params
}

// UpdateStructuredLogSettings modifies and returns the server's current structured log settings in ArangoDB 3.12.0+ format.
func (c *clientAdmin) UpdateStructuredLogSettings(ctx context.Context, opts *LogSettingsOptions) (LogSettingsOptions, error) {
	url := connection.NewUrl("_admin", "log", "structured")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		LogSettingsOptions    `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, formStructuredLogParams(opts))
	if err != nil {
		return LogSettingsOptions{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.LogSettingsOptions, nil
	default:
		return LogSettingsOptions{}, response.AsArangoErrorWithCode(code)
	}
}

// Get a list of the most recent requests with a timestamp and the endpoint in ArangoDB 3.12.5+ format.
func (c *clientAdmin) GetRecentAPICalls(ctx context.Context, dbName string) (ApiCallsResponse, error) {
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_admin", "server", "api-calls")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                ApiCallsResponse `json:"result"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ApiCallsResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return ApiCallsResponse{}, response.AsArangoErrorWithCode(code)
	}
}
