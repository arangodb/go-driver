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
}

type ClientAdminLicense interface {
	// GetLicense returns license of an ArangoDB deployment.
	GetLicense(ctx context.Context) (License, error)

	// SetLicense Set a new license for an Enterprise Edition instance.
	// Can be called on single servers, Coordinators, and DB-Servers.
	SetLicense(ctx context.Context, license string, force bool) error
}
