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

	// GetSystemTime returns the current system time as a Unix timestamp with microsecond precision.
	GetSystemTime(ctx context.Context, dbName string) (float64, error)

	// GetServerStatus returns status information about the server.
	GetServerStatus(ctx context.Context, dbName string) (ServerStatusResponse, error)

	// GetDeploymentSupportInfo retrieves deployment information for support purposes.
	GetDeploymentSupportInfo(ctx context.Context) (SupportInfoResponse, error)

	// GetStartupConfiguration return the effective configuration of the queried arangod instance.
	GetStartupConfiguration(ctx context.Context) (map[string]interface{}, error)

	// GetStartupConfigurationDescription fetches the available startup configuration
	// options of the queried arangod instance.
	GetStartupConfigurationDescription(ctx context.Context) (map[string]interface{}, error)

	// ReloadRoutingTable reloads the routing information from the _routing system
	// collection, causing Foxx services to rebuild their routing table.
	ReloadRoutingTable(ctx context.Context, dbName string) error

	// ExecuteAdminScript executes JavaScript code on the server.
	// Note: Requires ArangoDB to be started with --javascript.allow-admin-execute enabled.
	ExecuteAdminScript(ctx context.Context, dbName string, script *string) (interface{}, error)

	// CompactDatabases can be used to reclaim disk space after substantial data deletions have taken place,
	// by compacting the entire database system data.
	// The endpoint requires superuser access.
	CompactDatabases(ctx context.Context, opts *CompactOpts) (map[string]interface{}, error)

	// GetTLSData returns information about the server's TLS configuration.
	// This call requires authentication.
	GetTLSData(ctx context.Context, dbName string) (TLSDataResponse, error)

	// ReloadTLSData triggers a reload of all TLS data (server key, client-auth CA)
	// and returns the updated TLS configuration summary.
	// Requires superuser rights.
	ReloadTLSData(ctx context.Context) (TLSDataResponse, error)

	// RotateEncryptionAtRestKey reloads the user-supplied encryption key from
	// the --rocksdb.encryption-keyfolder and re-encrypts the internal encryption key.
	// Requires superuser rights and is not available on Coordinators.
	RotateEncryptionAtRestKey(ctx context.Context) ([]EncryptionKey, error)
	// GetJWTSecrets retrieves information about the currently loaded JWT secrets
	// for a given database.
	// Requires a superuser JWT for authorization.
	GetJWTSecrets(ctx context.Context, dbName string) (JWTSecretsResult, error)

	// ReloadJWTSecrets forces the server to reload the JWT secrets from disk.
	// Requires a superuser JWT for authorization.
	ReloadJWTSecrets(ctx context.Context) (JWTSecretsResult, error)

	// GetDeploymentId retrieves the unique deployment ID for the ArangoDB deployment.
	GetDeploymentId(ctx context.Context) (DeploymentIdResponse, error)
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

type ServerStatusResponse struct {
	// The server type (e.g., "arango")
	Server *string `json:"server,omitempty"`
	// The server version string (e.g,. "3.12.*")
	Version *string `json:"version,omitempty"`
	// Process ID of the server
	Pid *int `json:"pid,omitempty"`
	// License type (e.g., "community" or "enterprise")
	License *string `json:"license,omitempty"`
	// Mode in which the server is running
	Mode *string `json:"mode,omitempty"`
	// Operational mode (e.g., "server", "coordinator")
	OperationMode *string `json:"operationMode,omitempty"`
	// Whether the Foxx API is enabled
	FoxxApi *bool `json:"foxxApi,omitempty"`
	// Host of the server
	Host *string `json:"host,omitempty"`
	// System hostname of the server
	Hostname *string `json:"hostname,omitempty"`
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
	Maintenance *bool `json:"maintenance,omitempty"`
	// Role of the server (e.g., "SINGLE", "COORDINATOR")
	Role *string `json:"role,omitempty"`
	// Whether write operations are enabled
	WriteOpsEnabled *bool `json:"writeOpsEnabled,omitempty"`
	// Whether the server is in read-only mode
	ReadOnly *bool `json:"readOnly,omitempty"`

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
	Phase *string `json:"phase,omitempty"`
	// Current feature being processed (if any)
	Feature *string `json:"feature,omitempty"`
	// Recovery tick value
	RecoveryTick *int `json:"recoveryTick,omitempty"`
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

// ServerInfo contains details about either a single server host
// (in single-server deployments) or individual servers (in cluster deployments).
type ServerInfo struct {
	// Role of the server (e.g., SINGLE, COORDINATOR, DBServer, etc.)
	Role *string `json:"role,omitempty"`

	// Whether the server is in maintenance mode
	Maintenance *bool `json:"maintenance,omitempty"`

	// Whether the server is in read-only mode
	ReadOnly *bool `json:"readOnly,omitempty"`

	// ArangoDB version running on the server
	Version *string `json:"version,omitempty"`

	// Build identifier of the ArangoDB binary
	Build *string `json:"build,omitempty"`

	// License type (e.g., community, enterprise)
	License *string `json:"license,omitempty"`

	// Operating system information string
	Os *string `json:"os,omitempty"`

	// Platform (e.g., linux, windows, macos)
	Platform *string `json:"platform,omitempty"`

	// Information about the physical memory of the host
	PhysicalMemory PhysicalMemoryInfo `json:"physicalMemory"`

	// Information about the number of CPU cores
	NumberOfCores PhysicalMemoryInfo `json:"numberOfCores"`

	// Process statistics (uptime, memory, threads, etc.)
	ProcessStats ProcessStatsInfo `json:"processStats"`

	// CPU utilization statistics
	CpuStats CpuStatsInfo `json:"cpuStats"`

	// Optional storage engine statistics (only present in some responses)
	EngineStats *EngineStatsInfo `json:"engineStats,omitempty"`
}

// PhysicalMemoryInfo represents a numeric system property and whether it was overridden.
type PhysicalMemoryInfo struct {
	// The value of the property (e.g., memory size, CPU cores)
	Value *int64 `json:"value,omitempty"`

	// Whether this value was overridden by configuration
	Overridden *bool `json:"overridden,omitempty"`
}

// ProcessStatsInfo contains runtime statistics of the ArangoDB process.
type ProcessStatsInfo struct {
	// Uptime of the process in seconds
	ProcessUptime *float64 `json:"processUptime,omitempty"`

	// Number of active threads
	NumberOfThreads *int `json:"numberOfThreads,omitempty"`

	// Virtual memory size in bytes
	VirtualSize *int64 `json:"virtualSize,omitempty"`

	// Resident set size (RAM in use) in bytes
	ResidentSetSize *int64 `json:"residentSetSize,omitempty"`

	// Number of open file descriptors
	FileDescriptors *int `json:"fileDescriptors,omitempty"`

	// Limit on the number of file descriptors
	FileDescriptorsLimit *int64 `json:"fileDescriptorsLimit,omitempty"`
}

// CpuStatsInfo contains CPU usage percentages.
type CpuStatsInfo struct {
	// Percentage of CPU time spent in user mode
	UserPercent *float64 `json:"userPercent,omitempty"`

	// Percentage of CPU time spent in system/kernel mode
	SystemPercent *float64 `json:"systemPercent,omitempty"`

	// Percentage of CPU time spent idle
	IdlePercent *float64 `json:"idlePercent,omitempty"`

	// Percentage of CPU time spent waiting for I/O
	IowaitPercent *float64 `json:"iowaitPercent,omitempty"`
}

// EngineStatsInfo contains metrics from the RocksDB storage engine and cache.
type EngineStatsInfo struct {
	CacheLimit                  *int64 `json:"cache.limit,omitempty"`
	CacheAllocated              *int64 `json:"cache.allocated,omitempty"`
	RocksdbEstimateNumKeys      *int   `json:"rocksdb.estimate-num-keys,omitempty"`
	RocksdbEstimateLiveDataSize *int   `json:"rocksdb.estimate-live-data-size,omitempty"`
	RocksdbLiveSstFilesSize     *int   `json:"rocksdb.live-sst-files-size,omitempty"`
	RocksdbBlockCacheCapacity   *int64 `json:"rocksdb.block-cache-capacity,omitempty"`
	RocksdbBlockCacheUsage      *int   `json:"rocksdb.block-cache-usage,omitempty"`
	RocksdbFreeDiskSpace        *int64 `json:"rocksdb.free-disk-space,omitempty"`
	RocksdbTotalDiskSpace       *int64 `json:"rocksdb.total-disk-space,omitempty"`
}

// DeploymentInfo contains information about the deployment type and cluster layout.
type DeploymentInfo struct {
	// Type of deployment ("single" or "cluster")
	Type *string `json:"type,omitempty"`

	// Map of servers in the cluster, keyed by server ID (only present in cluster mode)
	Servers *map[ServerID]ServerInfo `json:"servers,omitempty"`

	// Number of agents in the cluster (cluster only)
	Agents *int `json:"agents,omitempty"`

	// Number of coordinators in the cluster (cluster only)
	Coordinators *int `json:"coordinators,omitempty"`

	// Number of DB servers in the cluster (cluster only)
	DbServers *int `json:"dbServers,omitempty"`

	// Shard distribution details (cluster only)
	Shards *ShardsInfo `json:"shards,omitempty"`
}

// ShardsInfo contains information about shard distribution in a cluster deployment.
type ShardsInfo struct {
	Databases   *int `json:"databases,omitempty"`
	Collections *int `json:"collections,omitempty"`
	Shards      *int `json:"shards,omitempty"`
	Leaders     *int `json:"leaders,omitempty"`
	RealLeaders *int `json:"realLeaders,omitempty"`
	Followers   *int `json:"followers,omitempty"`
	Servers     *int `json:"servers,omitempty"`
}

// SupportInfoResponse is the top-level response for GET /_db/_system/_admin/support-info.
// It provides details about the current deployment and server environment.
type SupportInfoResponse struct {
	// Deployment information (single or cluster, with related details)
	Deployment DeploymentInfo `json:"deployment"`

	// Host/server details (only present in single-server mode)
	Host *ServerInfo `json:"host,omitempty"`

	// Timestamp when the data was collected
	Date *string `json:"date,omitempty"`
}

type CompactOpts struct {
	//whether or not compacted data should be moved to the minimum possible level.
	ChangeLevel *bool `json:"changeLevel,omitempty"`
	// Whether or not to compact the bottommost level of data.
	CompactBottomMostLevel *bool `json:"compactBottomMostLevel,omitempty"`
}

// ServerName represents the hostname used in SNI configuration.
type ServerName string

// TLSConfigObject describes the details of a TLS keyfile or CA file.
type TLSDataObject struct {
	// SHA-256 hash of the whole input file (certificate or CA file).
	Sha256 *string `json:"sha256,omitempty"`
	// Public certificates in the chain, in PEM format.
	Certificates []string `json:"certificates,omitempty"`
	// SHA-256 hash of the private key (only present for keyfile).
	PrivateKeySha256 *string `json:"privateKeySha256,omitempty"`
}

// TLSConfigResponse represents the response of the TLS configuration endpoint.
type TLSDataResponse struct {
	// Information about the server TLS keyfile (certificate + private key).
	Keyfile *TLSDataObject `json:"keyfile,omitempty"`
	// Information about the CA certificates used for client verification.
	ClientCA *TLSDataObject `json:"clientCA,omitempty"`
	// Optional mapping of server names (via SNI) to their respective TLS configurations.
	SNI map[ServerName]TLSDataObject `json:"sni,omitempty"`
}

// EncryptionKey represents metadata about an encryption key used for
// RocksDB encryption-at-rest in ArangoDB.
// The server exposes only the SHA-256 hash of the key for identification.
// The actual key material is never returned for security reasons.
type EncryptionKey struct {
	// SHA256 is the SHA-256 hash of the encryption key, encoded as a hex string.
	// This is used to uniquely identify which key is active/available.
	SHA256 *string `json:"sha256,omitempty"`
}

// JWTSecretsResult contains the active and passive JWT secrets
type JWTSecretsResult struct {
	Active  *JWTSecret  `json:"active,omitempty"`  // The currently active JWT secret
	Passive []JWTSecret `json:"passive,omitempty"` // List of passive JWT secrets (may be empty)
}

// JWTSecret represents a single JWT secret's SHA-256 hash
type JWTSecret struct {
	SHA256 *string `json:"sha256,omitempty"` // SHA-256 hash of the JWT secret
}

type DeploymentIdResponse struct {
	// Id represents the unique deployment identifier
	Id *string `json:"id"`
}
