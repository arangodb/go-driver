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

	// DeleteLogLevels removes log levels for a specific server.
	DeleteLogLevels(ctx context.Context, serverId *string) (LogLevelResponse, error)

	// GetStructuredLogSettings returns the server's current structured log settings.
	GetStructuredLogSettings(ctx context.Context) (LogSettingsOptions, error)

	// UpdateStructuredLogSettings modifies and returns the server's current structured log settings.
	UpdateStructuredLogSettings(ctx context.Context, opts *LogSettingsOptions) (LogSettingsOptions, error)

	// GetRecentAPICalls gets a list of the most recent requests with a timestamp and the endpoint
	GetRecentAPICalls(ctx context.Context, dbName string) (ApiCallsResponse, error)

	// GetMetrics returns the instance's current metrics in Prometheus format
	GetMetrics(ctx context.Context, dbName string, serverId *string) ([]byte, error)
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

type LogLevelResponse struct {
	Agency string `json:"agency"`
	// Communication between Agency instances.
	AgencyComm string `json:"agencycomm"`
	// Agency's Raft store operations.
	AgencyStore string `json:"agencystore"`
	// Backup and restore processes.
	Backup string `json:"backup"`
	// Benchmarking and performance test logs.
	Bench string `json:"bench"`
	// General cluster-level logs.
	Cluster string `json:"cluster"`
	// Network communication between servers.
	Communication string `json:"communication"`
	// User authentication activities.
	Authentication string `json:"authentication"`
	// Configuration-related logs.
	Config string `json:"config"`
	// Crash handling.
	Crash string `json:"crash"`
	// Data export (dump) operations.
	Dump string `json:"dump"`
	// Storage engines (RocksDB)
	Engines string `json:"engines"`
	// General server logs not tied to a specific topic.
	General string `json:"general"`
	// Cluster heartbeat monitoring.
	Heartbeat string `json:"heartbeat"`
	// AQL query execution and planning.
	Aql string `json:"aql"`
	// Graph operations and traversals.
	Graphs string `json:"graphs"`
	// Maintenance operations in cluster.
	Maintenance string `json:"maintenance"`
	// User authorization and permissions.
	Authorization string `json:"authorization"`
	// Query execution and lifecycle.
	Queries string `json:"queries"`
	// Development/debugging logs.
	Development string `json:"development"`
	// Replication processes (followers, leaders).
	Replication string `json:"replication"`
	// V8 JavaScript engine logs.
	V8 string `json:"v8"`
	// Usage of deprecated features.
	Deprecation string `json:"deprecation"`
	// RocksDB storage engine-specific logs
	RocksDB string `json:"rocksdb"`
	// Audit logs for database operations.
	AuditDatabase string `json:"audit-database"`
	// Data validation errors/warnings.
	Validation string `json:"validation"`
	// RocksDB flush operations.
	Flush string `json:"flush"`
	// Audit logs for authorization events.
	AuditAuthorization string `json:"audit-authorization"`
	// System calls made by the server.
	Syscall string `json:"syscall"`
	// In-memory cache usage and performance.
	Cache string `json:"cache"`
	// Security-related logs.
	Security string `json:"security"`
	// Memory allocation and usage.
	Memory string `json:"memory"`
	// Restore operations from backup.
	Restore string `json:"restore"`
	// HTTP client communication logs.
	HTTPClient string `json:"httpclient"`
	// Audit logs for view operations.
	AuditView string `json:"audit-view"`
	// Audit logs for document operations.
	AuditDocument string `json:"audit-document"`
	// Audit logs for hot backup.
	AuditHotBackup string `json:"audit-hotbackup"`
	// Audit logs for collection operations.
	AuditCollection string `json:"audit-collection"`
	// Server statistics collection.
	Statistics string `json:"statistics"`
	// Incoming client requests.
	Requests string `json:"requests"`
	// Audit logs for service-level actions.
	AuditService string `json:"audit-service"`
	// TTL (Time-to-Live) expiration logs.
	TTL string `json:"ttl"`
	// Next-gen replication subsystem logs.
	Replication2 string `json:"replication2"`
	// SSL/TLS communication logs.
	SSL string `json:"ssl"`
	// Thread management logs.
	Threads string `json:"threads"`
	// License-related logs.
	License string `json:"license"`
	// IResearch (ArangoSearch) library logs.
	Libiresearch string `json:"libiresearch"`
	// Transactions.
	Trx string `json:"trx"`
	// Supervision process in the cluster.
	Supervision string `json:"supervision"`
	// Server startup sequence.
	Startup string `json:"startup"`
	// Audit logs for authentication events.
	AuditAuthentication string `json:"audit-authentication"`
	// Replication Write-Ahead Log.
	RepWal string `json:"rep-wal"`
	// View-related logs.
	Views string `json:"views"`
	// ArangoSearch engine logs.
	ArangoSearch string `json:"arangosearch"`
	// Replication state machine logs.
	RepState string `json:"rep-state"`
}

// LogSettingsOptions represents configurable flags for including
// specific fields in structured log output. It is used both in
// requests (to configure log behavior) and responses (to indicate
// which fields are currently enabled).
type LogSettingsOptions struct {
	// Database indicates whether the database name should be included
	// in structured log entries.
	Database *bool `json:"database,omitempty"`

	// Url indicates whether the request URL should be included
	// in structured log entries.
	Url *bool `json:"url,omitempty"`

	// Username indicates whether the authenticated username should be included
	// in structured log entries.
	Username *bool `json:"username,omitempty"`
}

type ApiCallsObject struct {
	// TimeStamp is the UTC timestamp when the API call was executed.
	TimeStamp string `json:"timeStamp"`

	// RequestType is the HTTP method used for the call (e.g., GET, POST).
	RequestType string `json:"requestType"`

	// Path is the HTTP request path that was accessed.
	Path string `json:"path"`

	// Database is the name of the database the API call was executed against.
	Database string `json:"database"`
}

type ApiCallsResponse struct {
	Calls []ApiCallsObject `json:"calls"`
}
