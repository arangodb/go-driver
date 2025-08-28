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
	UpdateApplierConfig(ctx context.Context, dbName string, global *bool, opts UpdateApplierConfigOptions) (ApplierConfigResponse, error)
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
	IncludeSystem *bool `json:"includeSystem"`
	// Global indicates whether to return global inventory or not.
	// If true, the inventory will include all collections across all DBServers.
	Global *bool `json:"global"`
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

// UpdateApplierConfigOptions holds the configuration options for the replication applier.
// These settings can only be changed when the applier is not running.
type UpdateApplierConfigOptions struct {
	// AdaptivePolling controls whether the replication applier uses adaptive polling.
	AdaptivePolling *bool `json:"adaptivePolling"`

	// AutoResync, if set to true, allows the applier to automatically
	// trigger a full resynchronization if it falls too far behind.
	AutoResync *bool `json:"autoResync"`

	// AutoResyncRetries defines how many times the applier should retry
	// automatic resynchronization after failure.
	AutoResyncRetries *int `json:"autoResyncRetries"`

	// AutoStart indicates if the applier should start automatically
	// once configured.
	AutoStart *bool `json:"autoStart"`

	// ChunkSize is the maximum size (in bytes) of the data batches
	// fetched by the applier.
	ChunkSize *int `json:"chunkSize"`

	// ConnectTimeout is the timeout (in seconds) for the initial
	// connection attempt to the master endpoint.
	ConnectTimeout *int `json:"connectTimeout"`

	// ConnectionRetryWaitTime is the wait time (in seconds) before retrying
	// a failed connection attempt.
	ConnectionRetryWaitTime *int `json:"connectionRetryWaitTime"`

	// Database is the name of the database on the master that the applier
	// should replicate from.
	Database *string `json:"database"`

	// Endpoint specifies the master server endpoint (e.g., "tcp://127.0.0.1:8529")
	// from which replication data is pulled. This is required.
	Endpoint *string `json:"endpoint"`

	// IdleMaxWaitTime is the maximum wait time (in seconds) between
	// polling requests when the applier is idle.
	IdleMaxWaitTime *int `json:"idleMaxWaitTime"`

	// IdleMinWaitTime is the minimum wait time (in seconds) between
	// polling requests when the applier is idle.
	IdleMinWaitTime *int `json:"idleMinWaitTime"`

	// IncludeSystem specifies whether system collections should be
	// replicated as well.
	IncludeSystem *bool `json:"includeSystem"`

	// InitialSyncMaxWaitTime defines the maximum wait time (in seconds)
	// for the initial synchronization step.
	InitialSyncMaxWaitTime *int `json:"initialSyncMaxWaitTime"`

	// MaxConnectRetries is the maximum number of retries for
	// initial connection attempts.
	MaxConnectRetries *int `json:"maxConnectRetries"`

	// Password is the password used when connecting to the master.
	Password *string `json:"password"`

	// RequestTimeout specifies the timeout (in seconds) for individual
	// HTTP requests made by the applier.
	RequestTimeout *int `json:"requestTimeout"`

	// RequireFromPresent, if true, requires the replication to start from
	// the present and not accept missing history.
	RequireFromPresent *bool `json:"requireFromPresent"`

	// RestrictCollections is an optional list of collections to include
	// or exclude in replication, depending on RestrictType.
	RestrictCollections *[]string `json:"restrictCollections"`

	// RestrictType determines how RestrictCollections is interpreted:
	// "include" or "exclude".
	RestrictType *string `json:"restrictType"`

	// Username is the username used when connecting to the master.
	Username *string `json:"username"`

	// Verbose controls the verbosity of the applier's logging.
	Verbose *bool `json:"verbose"`
}
