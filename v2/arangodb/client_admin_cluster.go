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
