//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

import "context"

type ClientAdminCluster interface {
	// Health returns the cluster configuration & health. Not available in single server deployments (403 Forbidden).
	Health(ctx context.Context) (ClusterHealth, error)

	// DatabaseInventory the inventory of the cluster collections (with entire details) from a specific database.
	DatabaseInventory(ctx context.Context, dbName string) (DatabaseInventory, error)

	// MoveShard moves a single shard of the given collection between `fromServer` and `toServer`.
	MoveShard(ctx context.Context, col Collection, shard ShardID, fromServer, toServer ServerID) (string, error)

	// CleanOutServer triggers activities to clean out a DBServer.
	CleanOutServer(ctx context.Context, serverID ServerID) (string, error)

	// ResignServer triggers activities to let a DBServer resign for all shards.
	ResignServer(ctx context.Context, serverID ServerID) (string, error)

	// NumberOfServers returns the number of coordinators & dbServers in a clusters and the ID's of cleanedOut servers.
	NumberOfServers(ctx context.Context) (NumberOfServersResponse, error)

	// IsCleanedOut checks if the dbServer with given ID has been cleaned out.
	IsCleanedOut(ctx context.Context, serverID ServerID) (bool, error)

	// RemoveServer is a low-level option to remove a server from a cluster.
	// This function is suitable for servers of type coordinator or dbServer.
	// The use of `ClientServerAdmin.Shutdown` is highly recommended above this function.
	RemoveServer(ctx context.Context, serverID ServerID) error

	// ClusterStatistics retrieves statistical information from a specific DBServer
	// in an ArangoDB cluster. The statistics include system, client, HTTP, and server
	// metrics such as CPU usage, memory, connections, requests, and transaction details.
	ClusterStatistics(ctx context.Context, dbServer string) (ClusterStatisticsResponse, error)

	// ClusterEndpoints returns the endpoints of a cluster.
	ClusterEndpoints(ctx context.Context) (ClusterEndpointsResponse, error)

	// GetDBServerMaintenance retrieves the maintenance status of a given DB-Server.
	// It checks whether the specified DB-Server is in maintenance mode and,
	// if so, until what date and time (in ISO 8601 format) the maintenance will last.
	GetDBServerMaintenance(ctx context.Context, dbServer string) (ClusterMaintenanceResponse, error)

	// SetDBServerMaintenance sets the maintenance mode for a specific DB-Server.
	// This endpoint affects only the given DB-Server. When in maintenance mode,
	// the server is excluded from supervision actions such as shard distribution
	// or failover. This is typically used during planned restarts or upgrades.
	SetDBServerMaintenance(ctx context.Context, dbServer string, opts *ClusterMaintenanceOpts) error

	// SetClusterMaintenance sets the cluster-wide supervision maintenance mode.
	// This endpoint affects the supervision (Agency) component of the cluster.
	// While enabled, automatic failovers, shard movements, and repair jobs
	// are suspended. The mode can be:
	//
	//   - "on":   Enable maintenance mode for the default 60 minutes.
	//   - "off":  Disable maintenance mode immediately.
	//   - "<number>":  Enable maintenance mode for <number> seconds.
	//
	// Be aware that no automatic failovers of any kind will take place while
	// the maintenance mode is enabled. The supervision will reactivate itself
	// automatically after the duration expires.
	SetClusterMaintenance(ctx context.Context, mode string) error

	// GetClusterRebalance retrieves the current cluster imbalance status.
	// It computes the imbalance across leaders and shards, and includes the number of
	// ongoing and pending move shard operations.
	GetClusterRebalance(ctx context.Context) (RebalanceResponse, error)

	// ComputeClusterRebalance computes a set of move shard operations to improve cluster balance.
	ComputeClusterRebalance(ctx context.Context, opts *RebalanceRequestBody) (RebalancePlan, error)

	// ExecuteClusterRebalance executes a set of shard move operations on the cluster.
	ExecuteClusterRebalance(ctx context.Context, opts *ExecuteRebalanceRequestBody) error

	// ComputeAndExecuteClusterRebalance computes moves internally then executes them.
	ComputeAndExecuteClusterRebalance(ctx context.Context, opts *RebalanceRequestBody) (RebalancePlan, error)
}

type NumberOfServersResponse struct {
	NoCoordinators   int        `json:"numberOfCoordinators,omitempty"`
	NoDBServers      int        `json:"numberOfDBServers,omitempty"`
	CleanedServerIDs []ServerID `json:"cleanedServers,omitempty"`
}

type DatabaseInventory struct {
	Info        DatabaseInfo          `json:"properties,omitempty"`
	Collections []InventoryCollection `json:"collections,omitempty"`
	Views       []InventoryView       `json:"views,omitempty"`
	State       ServerStatus          `json:"state,omitempty"`
	Tick        string                `json:"tick,omitempty"`
}

type InventoryCollection struct {
	Parameters  InventoryCollectionParameters `json:"parameters"`
	Indexes     []InventoryIndex              `json:"indexes,omitempty"`
	PlanVersion int64                         `json:"planVersion,omitempty"`
	IsReady     bool                          `json:"isReady,omitempty"`
	AllInSync   bool                          `json:"allInSync,omitempty"`
}

type InventoryCollectionParameters struct {
	Deleted bool                   `json:"deleted,omitempty"`
	Shards  map[ShardID][]ServerID `json:"shards,omitempty"`
	PlanID  string                 `json:"planId,omitempty"`

	CollectionProperties
}

type InventoryIndex struct {
	ID              string   `json:"id,omitempty"`
	Type            string   `json:"type,omitempty"`
	Fields          []string `json:"fields,omitempty"`
	Unique          bool     `json:"unique"`
	Sparse          bool     `json:"sparse"`
	Deduplicate     bool     `json:"deduplicate"`
	MinLength       int      `json:"minLength,omitempty"`
	GeoJSON         bool     `json:"geoJson,omitempty"`
	Name            string   `json:"name,omitempty"`
	ExpireAfter     int      `json:"expireAfter,omitempty"`
	Estimates       bool     `json:"estimates,omitempty"`
	FieldValueTypes string   `json:"fieldValueTypes,omitempty"`
	CacheEnabled    *bool    `json:"cacheEnabled,omitempty"`
}

type InventoryView struct {
	Name     string   `json:"name,omitempty"`
	Deleted  bool     `json:"deleted,omitempty"`
	ID       string   `json:"id,omitempty"`
	IsSystem bool     `json:"isSystem,omitempty"`
	PlanID   string   `json:"planId,omitempty"`
	Type     ViewType `json:"type,omitempty"`

	ArangoSearchViewProperties
}

// CollectionByName returns the InventoryCollection with given name. Return false if not found.
func (i DatabaseInventory) CollectionByName(name string) (InventoryCollection, bool) {
	for _, c := range i.Collections {
		if c.Parameters.Name == name {
			return c, true
		}
	}
	return InventoryCollection{}, false
}

// ViewByName returns the InventoryView with given name. Return false if not found.
func (i DatabaseInventory) ViewByName(name string) (InventoryView, bool) {
	for _, v := range i.Views {
		if v.Name == name {
			return v, true
		}
	}
	return InventoryView{}, false
}

// ClusterStatisticsResponse contains statistical data about the server as a whole.
type ClusterStatisticsResponse struct {
	Time       float64     `json:"time"`
	Enabled    bool        `json:"enabled"`
	System     SystemStats `json:"system"`
	Client     ClientStats `json:"client"`
	ClientUser ClientStats `json:"clientUser"`
	HTTP       HTTPStats   `json:"http"`
	Server     ServerStats `json:"server"`
}

// SystemStats contains statistical data about the system, this is part of
type SystemStats struct {
	MinorPageFaults     int64   `json:"minorPageFaults"`
	MajorPageFaults     int64   `json:"majorPageFaults"`
	UserTime            float32 `json:"userTime"`
	SystemTime          float32 `json:"systemTime"`
	NumberOfThreads     int     `json:"numberOfThreads"`
	ResidentSize        int64   `json:"residentSize"`
	ResidentSizePercent float64 `json:"residentSizePercent"`
	VirtualSize         int64   `json:"virtualSize"`
}

type ClientStats struct {
	HttpConnections int       `json:"httpConnections"`
	ConnectionTime  TimeStats `json:"connectionTime"`
	TotalTime       TimeStats `json:"totalTime"`
	RequestTime     TimeStats `json:"requestTime"`
	QueueTime       TimeStats `json:"queueTime"`
	IoTime          TimeStats `json:"ioTime"`
	BytesSent       TimeStats `json:"bytesSent"`
	BytesReceived   TimeStats `json:"bytesReceived"`
}

// TimeStats is used for various time-related statistics.
type TimeStats struct {
	Sum    float64 `json:"sum"`
	Count  int     `json:"count"`
	Counts []int   `json:"counts"`
}

// HTTPStats contains statistics about the HTTP traffic.
type HTTPStats struct {
	RequestsTotal     int `json:"requestsTotal"`
	RequestsSuperuser int `json:"requestsSuperuser"`
	RequestsUser      int `json:"requestsUser"`
	RequestsAsync     int `json:"requestsAsync"`
	RequestsGet       int `json:"requestsGet"`
	RequestsHead      int `json:"requestsHead"`
	RequestsPost      int `json:"requestsPost"`
	RequestsPut       int `json:"requestsPut"`
	RequestsPatch     int `json:"requestsPatch"`
	RequestsDelete    int `json:"requestsDelete"`
	RequestsOptions   int `json:"requestsOptions"`
	RequestsOther     int `json:"requestsOther"`
}

// ServerStats contains statistics about the server.
type ServerStats struct {
	Uptime         float64          `json:"uptime"`
	PhysicalMemory int64            `json:"physicalMemory"`
	Transactions   TransactionStats `json:"transactions"`
	V8Context      V8ContextStats   `json:"v8Context"`
	Threads        ThreadStats      `json:"threads"`
}

// TransactionStats contains statistics about transactions.
type TransactionStats struct {
	Started             int `json:"started"`
	Aborted             int `json:"aborted"`
	Committed           int `json:"committed"`
	IntermediateCommits int `json:"intermediateCommits"`
	ReadOnly            int `json:"readOnly,omitempty"`
	DirtyReadOnly       int `json:"dirtyReadOnly,omitempty"`
}

// V8ContextStats contains statistics about V8 contexts.
type V8ContextStats struct {
	Available int           `json:"available"`
	Busy      int           `json:"busy"`
	Dirty     int           `json:"dirty"`
	Free      int           `json:"free"`
	Max       int           `json:"max"`
	Min       int           `json:"min"`
	Memory    []MemoryStats `json:"memory"`
}

// MemoryStats contains statistics about memory usage.
type MemoryStats struct {
	ContextId    int     `json:"contextId"`
	TMax         float64 `json:"tMax"`
	CountOfTimes int     `json:"countOfTimes"`
	HeapMax      int64   `json:"heapMax"`
	HeapMin      int64   `json:"heapMin"`
	Invocations  int     `json:"invocations"`
}

// ThreadStats contains statistics about threads.
type ThreadStats struct {
	SchedulerThreads int `json:"scheduler-threads"`
	Blocked          int `json:"blocked"`
	Queued           int `json:"queued"`
	InProgress       int `json:"in-progress"`
	DirectExec       int `json:"direct-exec"`
}

// It contains a list of cluster endpoints that a client can use
// to connect to the ArangoDB cluster.
type ClusterEndpointsResponse struct {
	// Endpoints is the list of cluster endpoints (usually coordinators)
	// that the client can use to connect to the cluster.
	Endpoints []ClusterEndpoint `json:"endpoints,omitempty"`
}

// ClusterEndpoint represents a single cluster endpoint.
// Each endpoint provides a URL to connect to a coordinator.
type ClusterEndpoint struct {
	// Endpoint is the connection string (protocol + host + port)
	// of a coordinator in the cluster, e.g. "tcp://127.0.0.1:8529".
	Endpoint string `json:"endpoint,omitempty"`
}

// ClusterMaintenanceResponse represents the maintenance status of a DB-Server.
type ClusterMaintenanceResponse struct {
	// The mode of the DB-Server. The value is "maintenance".
	Mode string `json:"mode,omitempty"`

	// Until what date and time the maintenance mode currently lasts,
	// in the ISO 8601 date/time format.
	Until string `json:"until,omitempty"`
}

// ClusterMaintenanceOpts represents the options for setting maintenance mode
// on a DB-Server in an ArangoDB cluster.
type ClusterMaintenanceOpts struct {
	// Mode specifies the maintenance mode to apply to the DB-Server.
	// Possible values:
	//   - "maintenance": enable maintenance mode
	//   - "normal": disable maintenance mode
	// This field is required.
	Mode string `json:"mode"`

	// Timeout specifies how long the maintenance mode should last, in seconds.
	// This field is optional; if nil, the server will use the default timeout (usually 3600 seconds).
	Timeout *int `json:"timeout"`
}

// It contains leader statistics, shard statistics, and the count of ongoing/pending move shard operations.
type RebalanceResponse struct {
	// Statistics related to leader distribution
	Leader LeaderStats `json:"leader,omitempty"`
	// Statistics related to shard distribution (JSON key is "shards", not "shard")
	Shards ShardStats `json:"shards,omitempty"`
	// Number of ongoing move shard operations
	PendingMoveShards *int64 `json:"pendingMoveShards,omitempty"`
	// Number of pending (scheduled) move shard operations
	TodoMoveShards *int64 `json:"todoMoveShards,omitempty"`
}

// LeaderStats holds information about leader balancing across DB-Servers.
type LeaderStats struct {
	// Actual leader weight used per server
	WeightUsed []int `json:"weightUsed,omitempty"`
	// Target leader weight per server
	TargetWeight []float64 `json:"targetWeight,omitempty"`
	// Number of leader shards per server
	NumberShards []int `json:"numberShards,omitempty"`
	// Number of duplicated leaders per server
	LeaderDupl []int `json:"leaderDupl,omitempty"`
	// Total leader weight
	TotalWeight *int `json:"totalWeight,omitempty"`
	// Computed imbalance percentage
	Imbalance *float64 `json:"imbalance,omitempty"`
	// Total number of leader shards
	TotalShards *int64 `json:"totalShards,omitempty"`
}

// ShardStats holds information about shard balancing across DB-Servers.
type ShardStats struct {
	// Actual size used per server
	SizeUsed []int64 `json:"sizeUsed,omitempty"`
	// Target size per server
	TargetSize []float64 `json:"targetSize,omitempty"`
	// Number of shards per server
	NumberShards []int `json:"numberShards,omitempty"`
	// Total size used across servers
	TotalUsed *int64 `json:"totalUsed,omitempty"`
	// Total number of shards
	TotalShards *int64 `json:"totalShards,omitempty"`
	// Number of shards belonging to system collections
	TotalShardsFromSystemCollections *int64 `json:"totalShardsFromSystemCollections,omitempty"`
	// Computed imbalance factor for shards
	Imbalance *float64 `json:"imbalance,omitempty"`
}

// RebalanceRequestBody provides a default configuration for rebalancing requests.
// RebalanceRequestBody provides the options for computing a rebalance plan.
// It corresponds to the request body for POST /_admin/cluster/rebalance.
type RebalanceRequestBody struct {
	// DatabasesExcluded is a list of database names to exclude from analysis.
	DatabasesExcluded []string `json:"databasesExcluded,omitempty"`
	// ExcludeSystemCollections indicates whether to exclude system collections.
	ExcludeSystemCollections *bool `json:"excludeSystemCollections,omitempty"`
	// LeaderChanges indicates whether leader changes are allowed.
	LeaderChanges *bool `json:"leaderChanges,omitempty"`
	// MaximumNumberOfMoves is the maximum number of shard move operations to generate.
	MaximumNumberOfMoves *int `json:"maximumNumberOfMoves,omitempty"`
	// MoveFollowers indicates whether follower shard moves are allowed.
	MoveFollowers *bool `json:"moveFollowers,omitempty"`
	// MoveLeaders indicates whether leader shard moves are allowed.
	MoveLeaders *bool `json:"moveLeaders,omitempty"`
	// PiFactor is the weighting factor used in imbalance computation.
	PiFactor *int `json:"piFactor,omitempty"`
	// Version must be set to 1.
	Version *int `json:"version"`
}

// RebalancePlan contains the imbalance statistics before
// and after rebalancing, along with the list of suggested move operations.
type RebalancePlan struct {
	// ImbalanceBefore shows the imbalance metrics before applying the plan.
	ImbalanceBefore ImbalanceStats `json:"imbalanceBefore"`
	// ImbalanceAfter shows the imbalance metrics after applying the plan.
	ImbalanceAfter ImbalanceStats `json:"imbalanceAfter"`
	// Moves contains the list of suggested shard move operations.
	Moves []MoveOperation `json:"moves"`
}

// ImbalanceStats holds leader and shard distribution statistics
// used to measure cluster imbalance.
type ImbalanceStats struct {
	// Leader contains statistics related to leader distribution.
	Leader LeaderStats `json:"leader,omitempty"`
	// Shards contains statistics related to shard distribution.
	Shards ShardStats `json:"shards,omitempty"`
}

// MoveOperation describes a suggested shard move as part of the rebalance plan.
type MoveOperation struct {
	// Collection is the collection identifier for the shard.
	Collection *string `json:"collection,omitempty"`
	// From is the source server ID.
	From *string `json:"from,omitempty"`
	// IsLeader indicates if the move involves a leader shard.
	IsLeader *bool `json:"isLeader,omitempty"`
	// Shard is the shard identifier being moved.
	Shard *string `json:"shard,omitempty"`
	// To is the destination server ID.
	To *string `json:"to,omitempty"`

	// Database is the database name containing the collection.
	Database *string `json:"database,omitempty"`
}

// It contains the set of shard move operations to perform and the version of the rebalance plan.
type ExecuteRebalanceRequestBody struct {
	// Moves is a list of shard move operations that should be executed.
	// Each move specifies which shard to move, from which server to which server,
	// whether it is a leader shard, the collection, and the database.
	Moves []MoveOperation `json:"moves"`

	// Version specifies the version of the rebalance plan that this request applies to.
	// This should match the version returned by ComputeClusterRebalance.
	Version *int `json:"version"`
}
