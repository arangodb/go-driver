//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"encoding/json"
	"path"
	"reflect"
)

// newCluster creates a new Cluster implementation.
func newCluster(conn Connection) (Cluster, error) {
	if conn == nil {
		return nil, WithStack(InvalidArgumentError{Message: "conn is nil"})
	}
	return &cluster{
		conn: conn,
	}, nil
}

type cluster struct {
	conn Connection
}

// LoggerState returns the state of the replication logger
func (c *cluster) Health(ctx context.Context) (ClusterHealth, error) {
	req, err := c.conn.NewRequest("GET", "_admin/cluster/health")
	if err != nil {
		return ClusterHealth{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ClusterHealth{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ClusterHealth{}, WithStack(err)
	}
	var result ClusterHealth
	if err := resp.ParseBody("", &result); err != nil {
		return ClusterHealth{}, WithStack(err)
	}
	return result, nil
}

// Get the inventory of the cluster containing all collections (with entire details) of a database.
func (c *cluster) DatabaseInventory(ctx context.Context, db Database) (DatabaseInventory, error) {
	req, err := c.conn.NewRequest("GET", path.Join("_db", db.Name(), "_api/replication/clusterInventory"))
	if err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	var result DatabaseInventory
	if err := resp.ParseBody("", &result); err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	return result, nil
}

type moveShardRequest struct {
	Database   string   `json:"database"`
	Collection string   `json:"collection"`
	Shard      ShardID  `json:"shard"`
	FromServer ServerID `json:"fromServer"`
	ToServer   ServerID `json:"toServer"`
}

// MoveShard moves a single shard of the given collection from server `fromServer` to
// server `toServer`.
func (c *cluster) MoveShard(ctx context.Context, col Collection, shard ShardID, fromServer, toServer ServerID) error {
	req, err := c.conn.NewRequest("POST", "_admin/cluster/moveShard")
	if err != nil {
		return WithStack(err)
	}
	input := moveShardRequest{
		Database:   col.Database().Name(),
		Collection: col.Name(),
		Shard:      shard,
		FromServer: fromServer,
		ToServer:   toServer,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(202); err != nil {
		return WithStack(err)
	}
	var result jobIDResponse
	if err := resp.ParseBody("", &result); err != nil {
		return WithStack(err)
	}
	if cs.JobIDResponse != nil {
		*cs.JobIDResponse = result.JobID
	}
	return nil
}

type cleanOutServerRequest struct {
	Server string `json:"server"`
}

type jobIDResponse struct {
	JobID string `json:"id"`
}

// CleanOutServer triggers activities to clean out a DBServers.
func (c *cluster) CleanOutServer(ctx context.Context, serverID string) error {
	req, err := c.conn.NewRequest("POST", "_admin/cluster/cleanOutServer")
	if err != nil {
		return WithStack(err)
	}
	input := cleanOutServerRequest{
		Server: serverID,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return WithStack(err)
	}
	var result jobIDResponse
	if err := resp.ParseBody("", &result); err != nil {
		return WithStack(err)
	}
	if cs.JobIDResponse != nil {
		*cs.JobIDResponse = result.JobID
	}
	return nil
}

// ResignServer triggers activities to let a DBServer resign for all shards.
func (c *cluster) ResignServer(ctx context.Context, serverID string) error {
	req, err := c.conn.NewRequest("POST", "_admin/cluster/resignLeadership")
	if err != nil {
		return WithStack(err)
	}
	input := cleanOutServerRequest{
		Server: serverID,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return WithStack(err)
	}
	var result jobIDResponse
	if err := resp.ParseBody("", &result); err != nil {
		return WithStack(err)
	}
	if cs.JobIDResponse != nil {
		*cs.JobIDResponse = result.JobID
	}
	return nil
}

// IsCleanedOut checks if the dbserver with given ID has been cleaned out.
func (c *cluster) IsCleanedOut(ctx context.Context, serverID string) (bool, error) {
	r, err := c.NumberOfServers(ctx)
	if err != nil {
		return false, WithStack(err)
	}
	for _, id := range r.CleanedServerIDs {
		if id == serverID {
			return true, nil
		}
	}
	return false, nil
}

// NumberOfServersResponse holds the data returned from a NumberOfServer request.
type NumberOfServersResponse struct {
	NoCoordinators   int      `json:"numberOfCoordinators,omitempty"`
	NoDBServers      int      `json:"numberOfDBServers,omitempty"`
	CleanedServerIDs []string `json:"cleanedServers,omitempty"`
}

// NumberOfServers returns the number of coordinator & dbservers in a clusters and the
// ID's of cleaned out servers.
func (c *cluster) NumberOfServers(ctx context.Context) (NumberOfServersResponse, error) {
	req, err := c.conn.NewRequest("GET", "_admin/cluster/numberOfServers")
	if err != nil {
		return NumberOfServersResponse{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return NumberOfServersResponse{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return NumberOfServersResponse{}, WithStack(err)
	}
	var result NumberOfServersResponse
	if err := resp.ParseBody("", &result); err != nil {
		return NumberOfServersResponse{}, WithStack(err)
	}
	return result, nil
}

// RemoveServer is a low-level option to remove a server from a cluster.
// This function is suitable for servers of type coordinator or dbserver.
// The use of `ClientServerAdmin.Shutdown` is highly recommended above this function.
func (c *cluster) RemoveServer(ctx context.Context, serverID ServerID) error {
	req, err := c.conn.NewRequest("POST", "_admin/cluster/removeServer")
	if err != nil {
		return WithStack(err)
	}
	if _, err := req.SetBody(serverID); err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return WithStack(err)
	}
	return nil
}

// replicationFactor represents the replication factor of a collection
// Has special value ReplicationFactorSatellite for satellite collections
type replicationFactor int

type inventoryCollectionParametersInternal struct {
	Deleted             bool             `json:"deleted,omitempty"`
	DoCompact           bool             `json:"doCompact,omitempty"`
	ID                  string           `json:"id,omitempty"`
	IndexBuckets        int              `json:"indexBuckets,omitempty"`
	Indexes             []InventoryIndex `json:"indexes,omitempty"`
	IsSmart             bool             `json:"isSmart,omitempty"`
	SmartGraphAttribute string           `json:"smartGraphAttribute,omitempty"`
	IsSystem            bool             `json:"isSystem,omitempty"`
	IsVolatile          bool             `json:"isVolatile,omitempty"`
	JournalSize         int64            `json:"journalSize,omitempty"`
	KeyOptions          struct {
		Type          string `json:"type,omitempty"`
		AllowUserKeys bool   `json:"allowUserKeys,omitempty"`
		LastValue     int64  `json:"lastValue,omitempty"`
	} `json:"keyOptions"`
	Name              string            `json:"name,omitempty"`
	NumberOfShards    int               `json:"numberOfShards,omitempty"`
	Path              string            `json:"path,omitempty"`
	PlanID            string            `json:"planId,omitempty"`
	ReplicationFactor replicationFactor `json:"replicationFactor,omitempty"`
	// Deprecated: use 'WriteConcern' instead
	MinReplicationFactor int `json:"minReplicationFactor,omitempty"`
	// Available from 3.6 arangod version.
	WriteConcern         int                    `json:"writeConcern,omitempty"`
	ShardKeys            []string               `json:"shardKeys,omitempty"`
	Shards               map[ShardID][]ServerID `json:"shards,omitempty"`
	Status               CollectionStatus       `json:"status,omitempty"`
	Type                 CollectionType         `json:"type,omitempty"`
	WaitForSync          bool                   `json:"waitForSync,omitempty"`
	DistributeShardsLike string                 `json:"distributeShardsLike,omitempty"`
	SmartJoinAttribute   string                 `json:"smartJoinAttribute,omitempty"`
	ShardingStrategy     ShardingStrategy       `json:"shardingStrategy,omitempty"`
}

func (p *InventoryCollectionParameters) asInternal() inventoryCollectionParametersInternal {
	return inventoryCollectionParametersInternal{
		Deleted:              p.Deleted,
		DoCompact:            p.DoCompact,
		ID:                   p.ID,
		IndexBuckets:         p.IndexBuckets,
		Indexes:              p.Indexes,
		IsSmart:              p.IsSmart,
		SmartGraphAttribute:  p.SmartGraphAttribute,
		IsSystem:             p.IsSystem,
		IsVolatile:           p.IsVolatile,
		JournalSize:          p.JournalSize,
		KeyOptions:           p.KeyOptions,
		Name:                 p.Name,
		NumberOfShards:       p.NumberOfShards,
		Path:                 p.Path,
		PlanID:               p.PlanID,
		ReplicationFactor:    replicationFactor(p.ReplicationFactor),
		MinReplicationFactor: p.MinReplicationFactor,
		WriteConcern:         p.WriteConcern,
		ShardKeys:            p.ShardKeys,
		Shards:               p.Shards,
		Status:               p.Status,
		Type:                 p.Type,
		WaitForSync:          p.WaitForSync,
		DistributeShardsLike: p.DistributeShardsLike,
		SmartJoinAttribute:   p.SmartJoinAttribute,
		ShardingStrategy:     p.ShardingStrategy,
	}
}

func (p *InventoryCollectionParameters) fromInternal(i inventoryCollectionParametersInternal) {
	*p = i.asExternal()
}

func (p *inventoryCollectionParametersInternal) asExternal() InventoryCollectionParameters {
	return InventoryCollectionParameters{
		Deleted:              p.Deleted,
		DoCompact:            p.DoCompact,
		ID:                   p.ID,
		IndexBuckets:         p.IndexBuckets,
		Indexes:              p.Indexes,
		IsSmart:              p.IsSmart,
		SmartGraphAttribute:  p.SmartGraphAttribute,
		IsSystem:             p.IsSystem,
		IsVolatile:           p.IsVolatile,
		JournalSize:          p.JournalSize,
		KeyOptions:           p.KeyOptions,
		Name:                 p.Name,
		NumberOfShards:       p.NumberOfShards,
		Path:                 p.Path,
		PlanID:               p.PlanID,
		ReplicationFactor:    int(p.ReplicationFactor),
		MinReplicationFactor: p.MinReplicationFactor,
		WriteConcern:         p.WriteConcern,
		ShardKeys:            p.ShardKeys,
		Shards:               p.Shards,
		Status:               p.Status,
		Type:                 p.Type,
		WaitForSync:          p.WaitForSync,
		DistributeShardsLike: p.DistributeShardsLike,
		SmartJoinAttribute:   p.SmartJoinAttribute,
		ShardingStrategy:     p.ShardingStrategy,
	}
}

// MarshalJSON converts InventoryCollectionParameters into json
func (p *InventoryCollectionParameters) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.asInternal())
}

// UnmarshalJSON loads InventoryCollectionParameters from json
func (p *InventoryCollectionParameters) UnmarshalJSON(d []byte) error {
	var internal inventoryCollectionParametersInternal
	if err := json.Unmarshal(d, &internal); err != nil {
		return err
	}

	p.fromInternal(internal)
	return nil
}

const (
	replicationFactorSatelliteString string = "satellite"
)

// MarshalJSON marshals InventoryCollectionParameters to arangodb json representation
func (r replicationFactor) MarshalJSON() ([]byte, error) {
	var replicationFactor interface{}

	if int(r) == ReplicationFactorSatellite {
		replicationFactor = replicationFactorSatelliteString
	} else {
		replicationFactor = int(r)
	}

	return json.Marshal(replicationFactor)
}

// UnmarshalJSON marshals InventoryCollectionParameters to arangodb json representation
func (r *replicationFactor) UnmarshalJSON(d []byte) error {
	var internal interface{}

	if err := json.Unmarshal(d, &internal); err != nil {
		return err
	}

	if i, ok := internal.(float64); ok {
		*r = replicationFactor(i)
		return nil
	} else if str, ok := internal.(string); ok {
		if ok && str == replicationFactorSatelliteString {
			*r = replicationFactor(ReplicationFactorSatellite)
			return nil
		}
	}

	return &json.UnmarshalTypeError{
		Value: string(d),
		Type:  reflect.TypeOf(r).Elem(),
	}
}
