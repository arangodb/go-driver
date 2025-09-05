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

	//GetSystemTime returns the current system time as a Unix timestamp with microsecond precision.
	GetSystemTime(ctx context.Context, dbName string) (float64, error)

	//GetServerStatus returns status information about the server.
	GetServerStatus(ctx context.Context, dbName string) (ServerStatusResponse, error)
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

type ServerStatusResponse struct {
	// The server type (e.g., "arango")
	Server string `json:"server"`
	// The server version string (e.g,. "3.12.*")
	Version string `json:"version"`
	// Process ID of the server
	Pid int `json:"pid"`
	// License type (e.g., "community" or "enterprise")
	License string `json:"license"`
	// Mode in which the server is running
	Mode string `json:"mode"`
	// Operational mode (e.g., "server", "coordinator")
	OperationMode string `json:"operationMode"`
	// Whether the Foxx API is enabled
	FoxxApi bool `json:"foxxApi"`
	// Host of the server
	Host string `json:"host"`
	// System hostname of the server
	Hostname string `json:"hostname"`
	// Nested server information details
	ServerInfo ServerInformation `json:"serverInfo"`

	// Present only in cluster mode
	Coordinator *CoordinatorInfo `json:"coordinator,omitempty"`
	Agency      *AgencyInfo      `json:"agency,omitempty"`
}

// ServerInformation provides detailed information about the server’s state.
// Some fields are present only in cluster deployments.
type ServerInformation struct {
	// Current progress of the server
	Progress ServerProgress `json:"progress"`
	// Whether the server is in maintenance mode
	Maintenance bool `json:"maintenance"`
	// Role of the server (e.g., "SINGLE", "COORDINATOR")
	Role string `json:"role"`
	// Whether write operations are enabled
	WriteOpsEnabled bool `json:"writeOpsEnabled"`
	// Whether the server is in read-only mode
	ReadOnly bool `json:"readOnly"`

	// Persisted server identifier (cluster only)
	PersistedId *string `json:"persistedId,omitempty"`
	// Reboot ID
	RebootId *int `json:"rebootId,omitempty"`
	// Network address
	Address *string `json:"address,omitempty"`
	// Unique server identifier
	ServerId *string `json:"serverId,omitempty"`
	// Current server state
	State *string `json:"state,omitempty"`
}

// ServerProgress contains information about the startup or recovery phase.
type ServerProgress struct {
	// Current phase of the server (e.g., "in wait")
	Phase string `json:"phase"`
	// Current feature being processed (if any)
	Feature string `json:"feature"`
	// Recovery tick value
	RecoveryTick int `json:"recoveryTick"`
}

// CoordinatorInfo provides information specific to the coordinator role (cluster only).
type CoordinatorInfo struct {
	// ID of the Foxxmaster coordinator
	Foxxmaster *string `json:"foxxmaster,omitempty"`
	// Whether this server is the Foxxmaster
	IsFoxxmaster *bool `json:"isFoxxmaster,omitempty"`
}

// AgencyInfo contains information about the agency configuration (cluster only).
type AgencyInfo struct {
	// Agency communication details
	AgencyComm *AgencyCommInfo `json:"agencyComm,omitempty"`
}

// AgencyCommInfo contains communication endpoints for the agency.
type AgencyCommInfo struct {
	// List of agency endpoints
	Endpoints *[]string `json:"endpoints,omitempty"`
}
