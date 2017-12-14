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
	"path"
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
	Shard      int      `json:"shard"`
	FromServer ServerID `json:"fromServer"`
	ToServer   ServerID `json:"toServer"`
}

// MoveShard moves a single shard of the given collection from server `fromServer` to
// server `toServer`.
func (c *cluster) MoveShard(ctx context.Context, col Collection, shard int, fromServer, toServer ServerID) error {
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
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	if err := resp.ParseBody("", nil); err != nil {
		return WithStack(err)
	}
	return nil
}
