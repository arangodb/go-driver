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
)

type ClientAdmin interface {
	ClientAdminLog
	ClientAdminBackup
	ClientAdminLicense
	ClientAdminCluster

	// ServerMode returns the current mode in which the server/cluster is operating.
	// This call needs ArangoDB 3.3 and up.
	ServerMode(ctx context.Context) (ServerMode, error)

	// SetServerMode changes the current mode in which the server/cluster is operating.
	// This call needs a client that uses JWT authentication.
	// This call needs ArangoDB 3.3 and up.
	SetServerMode(ctx context.Context, mode ServerMode) error

	// CheckAvailability checks if the particular server is available.
	// Use ClientAdminCluster.Health() to fetch the Endpoint list.
	// For ActiveFailover, it will return an error (503 code) if the server is not the leader.
	CheckAvailability(ctx context.Context, serverEndpoint string) error
}

type ClientAdminLog interface {
	// GetLogLevels returns log levels for topics.
	GetLogLevels(ctx context.Context, opts *LogLevelsGetOptions) (LogLevels, error)

	// SetLogLevels sets log levels for a given topics.
	SetLogLevels(ctx context.Context, logLevels LogLevels, opts *LogLevelsSetOptions) error

	// Logs retrieve logs from server in ArangoDB 3.8.0+ format
	Logs(ctx context.Context, queryParams *AdminLogEntriesOptions) (AdminLogEntriesResponse, error)
}

type ClientAdminLicense interface {
	// GetLicense returns license of an ArangoDB deployment.
	GetLicense(ctx context.Context) (License, error)

	// SetLicense Set a new license for an Enterprise Edition instance.
	// Can be called on single servers, Coordinators, and DB-Servers.
	SetLicense(ctx context.Context, license string, force bool) error
}

type AdminLogEntriesOptions struct {
	// Upto log level
	Upto string `json:"upto"` // (default: "info")

	// Returns all log entries of log level level.
	//  Note that the query parameters upto and level are mutually exclusive.
	Level *string `json:"level,omitempty"`

	// Start position
	Start int `json:"start"` // (default: 0)

	// Restricts the result to at most size log entries.
	Size *int `json:"size,omitempty"`

	// Offset position
	Offset int `json:"offset"` // (default: 0)

	// Only return the log entries containing the text specified in search.
	Search *string `json:"search,omitempty"`

	// Sort the log entries either ascending (if sort is asc) or
	// descending (if sort is desc) according to their id values.
	Sort string `json:"sort,omitempty"` // (default: "asc")

	// Returns all log entries of the specified server.
	//  If no serverId is given, the asked server will reply.
	// This parameter is only meaningful on Coordinators.
	ServerId *string `json:"serverId,omitempty"`
}

type AdminLogEntriesResponse struct {
	// Total number of log entries
	Total int `json:"total"`

	// List of log messages
	Messages []MessageObject `json:"messages"`
}

type MessageObject struct {
	Id int `json:"id"`
	// Log topic
	Topic string `json:"topic"`
	// Log level
	Level string `json:"level"`
	// Current date and time
	Date string `json:"date"`
	// Log message
	Message string `json:"message"`
}
