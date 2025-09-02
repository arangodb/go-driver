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
	"encoding/json"
	"time"
)

// ClientReplication defines replication API methods.
type ClientReplication interface {
	// CreateNewBatch creates a new replication batch.
	CreateNewBatch(ctx context.Context, dbName string, DBserver *string, state *bool, opt CreateNewBatchOptions) (CreateNewBatchResponse, error)
	// GetInventory retrieves the inventory of a replication batch.
	GetInventory(ctx context.Context, dbName string, params InventoryQueryParams) (InventoryResponse, error)
	// DeleteBatch deletes a replication batch.
	DeleteBatch(ctx context.Context, dbName string, DBserver *string, batchId string) error
	// ExtendBatch extends the TTL of a replication batch.
	ExtendBatch(ctx context.Context, dbName string, DBserver *string, batchId string, opt CreateNewBatchOptions) error
	// Dump retrieves a chunk of data from a collection in a replication batch.
	Dump(ctx context.Context, dbName string, params ReplicationDumpParams) ([]byte, error)
	// LoggerState retrieves the state of the replication logger.
	LoggerState(ctx context.Context, dbName string, DBserver *string) (LoggerStateResponse, error)
	// LoggerFirstTick retrieves the first tick of the replication logger.
	LoggerFirstTick(ctx context.Context, dbName string) (LoggerFirstTickResponse, error)
	// LoggerTickRange retrieves the currently available ranges of tick values for all currently available WAL logfiles.
	LoggerTickRange(ctx context.Context, dbName string) ([]LoggerTickRangeResponseObj, error)
	// GetApplierConfig retrieves the configuration of the replication applier.
	GetApplierConfig(ctx context.Context, dbName string, global *bool) (ApplierConfigResponse, error)
	// UpdateApplierConfig updates the configuration of the replication applier.
	UpdateApplierConfig(ctx context.Context, dbName string, global *bool, opts ApplierOptions) (ApplierConfigResponse, error)
	// ApplierStart starts the replication applier.
	ApplierStart(ctx context.Context, dbName string, global *bool, from *string) (ApplierStateResp, error)
	// ApplierStop stops the replication applier.
	ApplierStop(ctx context.Context, dbName string, global *bool) (ApplierStateResp, error)
	// GetApplierState retrieves the state of the replication applier.
	GetApplierState(ctx context.Context, dbName string, global *bool) (ApplierStateResp, error)
	// GetReplicationServerId retrieves the server ID used for replication.
	GetReplicationServerId(ctx context.Context, dbName string) (string, error)
	// MakeFollower makes the current server a follower of the specified leader.
	MakeFollower(ctx context.Context, dbName string, opts ApplierOptions) (ApplierStateResp, error)
	// GetWALRange retrieves the WAL range information.
	GetWALRange(ctx context.Context, dbName string) (WALRangeResponse, error)
	// GetWALLastTick retrieves the last available tick information.
	GetWALLastTick(ctx context.Context, dbName string) (WALLastTickResponse, error)
	// GetWALTail retrieves the tail of the WAL.
	GetWALTail(ctx context.Context, dbName string, params *WALTailOptions) ([]byte, error)
	// RebuildShardRevisionTree rebuilds the Merkle tree for a shard.
	RebuildShardRevisionTree(ctx context.Context, dbName string, shardID ShardID) error
	// GetShardRevisionTree retrieves the Merkle tree for a shard.
	GetShardRevisionTree(ctx context.Context, dbName string, shardID ShardID, batchId string) (json.RawMessage, error)
	// ListDocumentRevisionsInRange retrieves documents by their revision IDs.
	ListDocumentRevisionsInRange(ctx context.Context, dbName string, queryParams RevisionQueryParams, opts [][2]string) ([][2]string, error)
	// FetchRevisionDocuments retrieves documents by their revision IDs.
	FetchRevisionDocuments(ctx context.Context, dbName string, queryParams RevisionQueryParams, opts []string) ([]map[string]interface{}, error)
}

// CreateNewBatchOptions represents the request body for creating a batch.
type CreateNewBatchOptions struct {
	Ttl int `json:"ttl"`
}

// CreateNewBatchResponse represents the response for batch creation.
type CreateNewBatchResponse struct {
	// The ID of the created batch
	ID string `json:"id"`
	// The last tick of the created batch
	LastTick string `json:"lastTick"`
	// Only present if the state URL parameter was set to true
	State map[string]interface{} `json:"state,omitempty"`
}

// InventoryQueryParams represents the query parameters for the replication inventory API.
type InventoryQueryParams struct {
	// IncludeSystem indicates whether to include system collections in the inventory.
	IncludeSystem *bool `json:"includeSystem,omitempty"`
	// Global indicates whether to return global inventory or not.
	// If true, the inventory will include all collections across all DBServers.
	Global *bool `json:"global,omitempty"`
	// BatchID is the ID of the replication batch to query.
	BatchID string `json:"batchId"`
	// Collection is the name of the collection to restrict inventory to.
	Collection *string `json:"collection,omitempty"`

	// Only for Coordinators
	// Restrict to a specific DBserver in cluster
	DBserver *string `json:"DBserver,omitempty"`
}

// InventoryResponse represents the full response from the replication inventory API.
type InventoryResponse struct {
	// Collections is the list of collections in the inventory.
	Collections []CollectionsInventoryResponse `json:"collections,omitempty"`
	// Database properties.
	Properties PropertiesInventoryResponse `json:"properties,omitempty"`
	// Views present in the database.
	Views []ViewInventoryResponse `json:"views,omitempty"`
	// Replication state information.
	State StateInventoryResponse `json:"state,omitempty"`
	// Last log tick at the time of inventory.
	Tick *string `json:"tick,omitempty"`
}

// CollectionsInventoryResponse represents a collection entry in the inventory.
type CollectionsInventoryResponse struct {
	// Indexes defined on the collection.
	// Note: Primary indexes and edge indexes are not included in this array.
	Indexes []IndexesInventoryResponse `json:"indexes,omitempty"`
	// Collection properties and metadata.
	Parameters ParametersInventoryResponse `json:"parameters,omitempty"`
}

// ParametersInventoryResponse represents metadata and settings of a collection.
type ParametersInventoryResponse struct {
	// Reusable basic properties like ID and Name
	BasicProperties
	// AllowUserKeys indicates whether user keys are allowed.
	AllowUserKeys *bool `json:"allowUserKeys,omitempty"`
	// CacheEnabled indicates whether in-memory cache is enabled.
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`
	// Cid is the collection ID.
	Cid *string `json:"cid,omitempty"`
	// ComputedValues holds the computed values for the collection.
	ComputedValues interface{} `json:"computedValues,omitempty"`
	// Deleted indicates whether the collection is deleted.
	Deleted *bool `json:"deleted,omitempty"`
	// GloballyUniqueId is the globally unique identifier for the collection.
	GloballyUniqueId *string `json:"globallyUniqueId,omitempty"`
	// InternalValidatorType is the internal validator type.
	InternalValidatorType *int `json:"internalValidatorType,omitempty"`
	// IsDisjoint indicates whether disjoint smart graphs are used.
	IsDisjoint *bool `json:"isDisjoint,omitempty"`
	// IsSmart indicates whether the collection is a smart graph collection.
	IsSmart *bool `json:"isSmart,omitempty"`
	// IsSmartChild indicates whether the collection this is a child shard of a smart graph.
	IsSmartChild *bool `json:"isSmartChild,omitempty"`
	// IsSystem indicates whether the collection is a system collection.
	IsSystem *bool `json:"isSystem,omitempty"`
	// KeyOptions defines the key generation options for the collection.
	KeyOptions *KeyOpts `json:"keyOptions,omitempty"`
	// MinReplicationFactor defines the minimum replication factor for the collection.
	MinReplicationFactor *int `json:"minReplicationFactor,omitempty"`
	// NumberOfShards defines the number of shards for the collection.
	NumberOfShards *int `json:"numberOfShards,omitempty"`
	// PlanId is the plan ID for the collection.
	PlanId *string `json:"planId,omitempty"`
	// ReplicationFactor defines the replication factor for the collection.
	ReplicationFactor interface{} `json:"replicationFactor,omitempty"`
	// Schema defines the schema for the collection.
	Schema interface{} `json:"schema,omitempty"`
	// ShardKeys defines the shard keys for the collection.
	ShardKeys []string `json:"shardKeys,omitempty"`
	// ShardingStrategy defines the sharding strategy for the collection.
	ShardingStrategy *string `json:"shardingStrategy,omitempty"`
	// Shards defines the shards for the collection.
	Shards map[string][]string `json:"shards,omitempty"`
	// Status defines the Collection status code.
	Status *int `json:"status,omitempty"`
	// SyncByRevision indicates whether the collection is synced by revision.
	SyncByRevision *bool `json:"syncByRevision,omitempty"`
	// Type defines the Collection type (document/edge).
	Type *int `json:"type,omitempty"`
	// UsesRevisionsAsDocumentIds indicates whether document revisions are used as document IDs.
	UsesRevisionsAsDocumentIds *bool `json:"usesRevisionsAsDocumentIds,omitempty"`
	// Version defines the version of the collection.
	Version *int `json:"version,omitempty"`
	// WaitForSync indicates whether the collection should wait for sync.
	WaitForSync *bool `json:"waitForSync,omitempty"`
	// WriteConcern defines the write concern level for the collection.
	WriteConcern *int `json:"writeConcern,omitempty"`
}

// IndexesInventoryResponse represents metadata for an index in the collection.
type IndexesInventoryResponse struct {
	// Reusable basic properties like ID and Name
	BasicProperties
	// Index type (hash, skiplist, etc.)
	Type *string `json:"type,omitempty"`
	// Indexed fields
	Fields []string `json:"fields,omitempty"`
	// Unique indicates whether the index enforces uniqueness.
	Unique *bool `json:"unique,omitempty"`
	// Sparse indicates whether the index skips null values.
	Sparse *bool `json:"sparse,omitempty"`
	// Deduplicate indicates whether the index enforces deduplication.
	Deduplicate *bool `json:"deduplicate,omitempty"`
	// Estimates indicates whether the index supports estimates.
	Estimates *bool `json:"estimates,omitempty"`
	// CacheEnabled indicates whether the index is cache enabled.
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`
}

// KeyOpts represents options for document key generation.
type KeyOpts struct {
	// Whether user-supplied keys are allowed
	AllowUserKeys *bool `json:"allowUserKeys,omitempty"`
	// Key type (autoincrement, traditional, etc.)
	Type *string `json:"type,omitempty"`
	// Last value for autoincrement keys
	LastValue *int `json:"lastValue,omitempty"`
}

// PropertiesInventoryResponse represents database-level properties.
type PropertiesInventoryResponse struct {
	// Reusable basic properties like ID and Name
	BasicProperties
	// Whether this is a system database
	IsSystem *bool `json:"isSystem,omitempty"`
	// Default sharding method
	Sharding *string `json:"sharding,omitempty"`
	// Default replication factor
	ReplicationFactor interface{} `json:"replicationFactor,omitempty"`
	// Default write concern
	WriteConcern *int `json:"writeConcern,omitempty"`
	// Replication protocol version
	ReplicationVersion *string `json:"replicationVersion,omitempty"`
}

// StateInventoryResponse represents replication state at the time of inventory.
type StateInventoryResponse struct {
	// Whether replication is running
	Running *bool `json:"running,omitempty"`
	// Last committed log tick
	LastLogTick *string `json:"lastLogTick,omitempty"`
	// Last uncommitted log tick
	LastUncommittedLogTick *string `json:"lastUncommittedLogTick,omitempty"`
	// Total number of events
	TotalEvents *int `json:"totalEvents,omitempty"`
	// Timestamp of the state
	Time *time.Time `json:"time,omitempty"`
}

// ViewInventoryResponse represents a view entry in the inventory.
type ViewInventoryResponse struct {
	// Reusable basic properties like ID and Name
	BasicProperties
	// View type (e.g. "arangosearch")
	Type *string `json:"type,omitempty"`
	// View properties
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// BasicProperties represents reusable ID and Name fields common to collections, views, etc.
type BasicProperties struct {
	// Unique identifier (collection ID, view ID, etc.)
	ID *string `json:"id,omitempty"`
	// Human-readable name
	Name *string `json:"name,omitempty"`
}

type ReplicationDumpParams struct {
	// Collection name
	Collection string `json:"collection"`
	// Size of each chunk in bytes
	ChunkSize *int32 `json:"chunkSize,omitempty"`
	// BatchID is the ID of the replication batch.
	BatchID string `json:"batchId"`
}

type State struct {
	// Whether replication is running
	Running *bool `json:"running"`
	// Last committed log tick
	LastLogTick *string `json:"lastLogTick"`
	// Last uncommitted log tick
	LastUncommittedLogTick *string `json:"lastUncommittedLogTick"`
	// Total number of events
	TotalEvents *int64 `json:"totalEvents"`
	// Timestamp of the state
	Time *time.Time `json:"time"`
}

type Server struct {
	// Version of the server
	Version  *string `json:"version"`
	ServerId *string `json:"serverId"`
	// Engine of the server
	Engine *string `json:"engine"`
}

type LoggerStateResponse struct {
	State   State                    `json:"state"`
	Server  Server                   `json:"server"`
	Clients []map[string]interface{} `json:"clients,inline"`
}

type LoggerFirstTickResponse struct {
	// The first tick of the logger
	FirstTick *string `json:"firstTick,omitempty"`
}

type LoggerTickRangeResponseObj struct {
	// Name of the logfile
	Datafile *string `json:"datafile,omitempty"`
	// Status of the datafile, in textual form (e.g. "sealed", "open")
	Status *string `json:"status,omitempty"`
	// Minimum tick value contained in logfile
	TickMin *string `json:"tickMin,omitempty"`
	// Maximum tick value contained in logfile
	TickMax *string `json:"tickMax,omitempty"`
}

type ApplierConfigResponse struct {
	// Logger server endpoint (e.g., tcp://127.0.0.1:8529)
	Endpoint *string `json:"endpoint,omitempty"`
	// Database name (e.g., "_system")
	Database *string `json:"database,omitempty"`
	// Username for authentication
	Username *string `json:"username,omitempty"`
	// Password for authentication
	Password *string `json:"password,omitempty"`
	// Maximum connection attempts before stopping
	MaxConnectRetries int `json:"maxConnectRetries"`
	// Timeout (seconds) for connecting to endpoint
	ConnectTimeout int `json:"connectTimeout"`
	// Timeout (seconds) for individual requests
	RequestTimeout int `json:"requestTimeout"`
	// Max size of log transfer packets
	ChunkSize int `json:"chunkSize"`
	// Whether applier auto-starts on server startup
	AutoStart bool `json:"autoStart"`
	// Whether adaptive polling is used
	AdaptivePolling bool `json:"adaptivePolling"`
	// Whether system collections are included
	IncludeSystem bool `json:"includeSystem"`
	// Whether full automatic resync is performed if needed
	AutoResync bool `json:"autoResync"`
	// Number of auto-resync retries before giving up
	AutoResyncRetries int `json:"autoResyncRetries"`
	// Max wait time (seconds) for initial sync
	InitialSyncMaxWaitTime int `json:"initialSyncMaxWaitTime"`
	// Idle time (seconds) before retrying failed connection
	ConnectionRetryWaitTime int `json:"connectionRetryWaitTime"`
	// Minimum idle wait time (seconds) when no new data
	IdleMinWaitTime int `json:"idleMinWaitTime"`
	// Maximum idle wait time (seconds) when no new data (may be fractional, hence float64)
	IdleMaxWaitTime float64 `json:"idleMaxWaitTime"`
	// If true, aborts if start tick not available on leader
	RequireFromPresent bool `json:"requireFromPresent"`
	// If true, logs each applier operation (debugging only)
	Verbose bool `json:"verbose"`
	// Type of collection restriction ("include" or "exclude")
	RestrictType string `json:"restrictType"`
	// Collections included/excluded depending on RestrictType
	RestrictCollections []string `json:"restrictCollections"`
	// Max number of errors to ignore
	IgnoreErrors *int `json:"ignoreErrors,omitempty"`
	// SSL protocol version
	SslProtocol *int `json:"sslProtocol,omitempty"`
	// Whether to skip create/drop collection operations
	SkipCreateDrop *bool `json:"skipCreateDrop,omitempty"`
	// Max packet size (bytes)
	MaxPacketSize *int64 `json:"maxPacketSize,omitempty"`
	// Whether to include Foxx queues
	IncludeFoxxQueues *bool `json:"includeFoxxQueues,omitempty"`
	// Whether incremental sync is used
	Incremental *bool `json:"incremental,omitempty"`
}

// ApplierOptions holds the configuration options for the replication applier.
// These settings can only be changed when the applier is not running.
type ApplierOptions struct {
	// AdaptivePolling controls whether the replication applier uses adaptive polling.
	AdaptivePolling *bool `json:"adaptivePolling,omitempty"`

	// AutoResync, if set to true, allows the applier to automatically
	// trigger a full resynchronization if it falls too far behind.
	AutoResync *bool `json:"autoResync,omitempty"`

	// AutoResyncRetries defines how many times the applier should retry
	// automatic resynchronization after failure.
	AutoResyncRetries *int `json:"autoResyncRetries,omitempty"`

	// AutoStart indicates if the applier should start automatically
	// once configured.
	AutoStart *bool `json:"autoStart,omitempty"`

	// ChunkSize is the maximum size (in bytes) of the data batches
	// fetched by the applier.
	ChunkSize *int `json:"chunkSize,omitempty"`

	// ConnectTimeout is the timeout (in seconds) for the initial
	// connection attempt to the master endpoint.
	ConnectTimeout *int `json:"connectTimeout,omitempty"`

	// ConnectionRetryWaitTime is the wait time (in seconds) before retrying
	// a failed connection attempt.
	ConnectionRetryWaitTime *int `json:"connectionRetryWaitTime,omitempty"`

	// Database is the name of the database on the master that the applier
	// should replicate from.
	Database *string `json:"database,omitempty"`

	// Endpoint specifies the master server endpoint (e.g., "tcp://127.0.0.1:8529")
	// from which replication data is pulled. This is required.
	Endpoint *string `json:"endpoint,omitempty"`

	// IdleMaxWaitTime is the maximum wait time (in seconds) between
	// polling requests when the applier is idle.
	IdleMaxWaitTime *int `json:"idleMaxWaitTime,omitempty"`

	// IdleMinWaitTime is the minimum wait time (in seconds) between
	// polling requests when the applier is idle.
	IdleMinWaitTime *int `json:"idleMinWaitTime,omitempty"`

	// IncludeSystem specifies whether system collections should be
	// replicated as well.
	IncludeSystem *bool `json:"includeSystem,omitempty"`

	// InitialSyncMaxWaitTime defines the maximum wait time (in seconds)
	// for the initial synchronization step.
	InitialSyncMaxWaitTime *int `json:"initialSyncMaxWaitTime,omitempty"`

	// MaxConnectRetries is the maximum number of retries for
	// initial connection attempts.
	MaxConnectRetries *int `json:"maxConnectRetries,omitempty"`

	// Password is the password used when connecting to the master.
	Password *string `json:"password,omitempty"`

	// RequestTimeout specifies the timeout (in seconds) for individual
	// HTTP requests made by the applier.
	RequestTimeout *int `json:"requestTimeout,omitempty"`

	// RequireFromPresent, if true, requires the replication to start from
	// the present and not accept missing history.
	RequireFromPresent *bool `json:"requireFromPresent,omitempty"`

	// RestrictCollections is an optional list of collections to include
	// or exclude in replication, depending on RestrictType.
	RestrictCollections *[]string `json:"restrictCollections,omitempty"`

	// RestrictType determines how RestrictCollections is interpreted:
	// "include" or "exclude".
	RestrictType *string `json:"restrictType,omitempty"`

	// Username is the username used when connecting to the master.
	Username *string `json:"username,omitempty"`

	// Verbose controls the verbosity of the applier's logging.
	Verbose *bool `json:"verbose,omitempty"`
}

// ApplierState represents the current state of the replication applier.
type ApplierState struct {
	// Started indicates when the applier was started.
	Started *string `json:"started"`

	// Running is true if the applier is currently running.
	Running *bool `json:"running"`

	// Phase describes the current applier phase (e.g., "running", "inactive").
	Phase *string `json:"phase"`

	// LastAppliedContinuousTick is the tick of the last operation applied by the applier.
	LastAppliedContinuousTick *string `json:"lastAppliedContinuousTick"`

	// LastProcessedContinuousTick is the tick of the last operation processed.
	LastProcessedContinuousTick *string `json:"lastProcessedContinuousTick"`

	// LastAvailableContinuousTick is the last tick available on the replication logger.
	LastAvailableContinuousTick *string `json:"lastAvailableContinuousTick"`

	// SafeResumeTick is the tick from which the applier can safely resume.
	SafeResumeTick *string `json:"safeResumeTick"`

	// TicksBehind indicates how many ticks the applier is behind the latest log.
	TicksBehind *int64 `json:"ticksBehind,omitempty"`

	// Progress provides detailed information about the last progress event.
	Progress *ApplierProgress `json:"progress,omitempty"`

	// TotalRequests is the total number of requests made by the applier.
	TotalRequests *int `json:"totalRequests,omitempty"`

	// TotalFailedConnects counts the number of failed connection attempts.
	TotalFailedConnects *int `json:"totalFailedConnects,omitempty"`

	// TotalEvents is the total number of replication events processed.
	TotalEvents *int `json:"totalEvents,omitempty"`

	// TotalDocuments is the number of document operations applied.
	TotalDocuments *int `json:"totalDocuments,omitempty"`

	// TotalRemovals is the number of document removal operations applied.
	TotalRemovals *int `json:"totalRemovals,omitempty"`

	// TotalResyncs counts how many times a resync was triggered.
	TotalResyncs *int `json:"totalResyncs,omitempty"`

	// TotalOperationsExcluded is the number of operations ignored (due to filters, etc.).
	TotalOperationsExcluded *int `json:"totalOperationsExcluded,omitempty"`

	// TotalApplyTime is the cumulative time (in ms) spent applying operations.
	TotalApplyTime *int `json:"totalApplyTime,omitempty"`

	// AverageApplyTime is the average time (in ms) spent applying operations.
	AverageApplyTime *int `json:"averageApplyTime,omitempty"`

	// TotalFetchTime is the cumulative time (in ms) spent fetching operations.
	TotalFetchTime *int `json:"totalFetchTime,omitempty"`

	// AverageFetchTime is the average time (in ms) spent fetching operations.
	AverageFetchTime *int `json:"averageFetchTime,omitempty"`

	// LastError contains information about the last error, if any.
	LastError *struct {
		// ErrorNum is the numeric error code of the last error.
		ErrorNum *int `json:"errorNum,omitempty"`
		// ErrorMessage is the descriptive message of the last error.
		ErrorMessage *string `json:"errorMessage,omitempty"`
		// Time is the timestamp of the last error.
		Time time.Time `json:"time,omitempty"`
	} `json:"lastError,omitempty"`

	// Time is the timestamp of this applier state snapshot.
	Time time.Time `json:"time,omitempty"`
}

// ApplierProgress contains details about the applier's last progress event.
type ApplierProgress struct {
	// Time is when the progress message was recorded.
	Time *string `json:"time,omitempty"`

	// Message provides a short description of the progress (e.g., "applied batch").
	Message *string `json:"message,omitempty"`

	// FailedConnects counts failed connection attempts at this progress point.
	FailedConnects *int `json:"failedConnects,omitempty"`
}

type ApplierStateResp struct {
	// State holds detailed information about the applier's current state.
	State ApplierState `json:"state"`

	// Server contains information about the server providing this state.
	Server ApplierServer `json:"server"`

	// Endpoint is the endpoint this applier is connected to.
	Endpoint *string `json:"endpoint"`
}

type ApplierServer struct {
	// Version is the ArangoDB version.
	Version *string `json:"version"`

	// ServerId is the unique ID of the server.
	ServerId *string `json:"serverId"`
}

type WALRangeResponse struct {
	// Time is the timestamp when the range was recorded.
	Time time.Time `json:"time"`
	// Minimum tick in the range
	TickMin string `json:"tickMin"`
	// Maximum tick in the range
	TickMax string `json:"tickMax"`
	// Server information
	Server ApplierServer `json:"server"`
}

type WALLastTickResponse struct {
	// Time is the timestamp when the range was recorded.
	Time time.Time `json:"time"`
	// Tick contains the last available tick
	Tick string `json:"tick"`
	// Server information
	Server ApplierServer `json:"server"`
}

type WALTailOptions struct {
	// Global indicates whether operations for all databases should be included.
	// If set to false, only the operations for the current database are included.
	// The value true is only valid on the _system database.
	Global *bool `json:"global,omitempty"`

	// From specifies the exclusive lower bound tick value for the replication.
	From *int64 `json:"from,omitempty"`

	// To specifies the inclusive upper bound tick value for the replication.
	To *int64 `json:"to,omitempty"`

	// LastScanned specifies the last scanned tick value (for RocksDB multi-response support).
	LastScanned *int `json:"lastScanned,omitempty"`

	// ChunkSize specifies the approximate maximum size of the returned result in bytes.
	ChunkSize *int `json:"chunkSize,omitempty"`

	// SyncerId specifies the ID of the client used to tail results.
	// Required if ServerId is not provided.
	SyncerId *int64 `json:"syncerId,omitempty"`

	// ServerId specifies the ID of the client machine.
	// Required if SyncerId is not provided.
	ServerId *int64 `json:"serverId,omitempty"`

	// ClientInfo provides a short description of the client (informational only).
	ClientInfo *string `json:"clientInfo,omitempty"`
}

type RevisionQueryParams struct {
	// Collection Name
	Collection string `json:"collection"`
	BatchId    string `json:"batchId"`
	// The revisionId at which to resume, if a previous request was truncated
	Resume *string `json:"resume,omitempty"`
}
