//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
	"strconv"
)

type batch_params struct {
	ttl float64 `json:"ttl"`
}

// CreateBatch creates a "batch" to prevent WAL file removal and to take a snapshot
func (c *client) CreateBatch(ctx context.Context, serverID int64, db Database) (BatchMetadata, error) {
	req, err := c.conn.NewRequest("POST", path.Join("_db", db.Name(), "_api/replication/batch"))
	if err != nil {
		return BatchMetadata{}, WithStack(err)
	}
	req = req.SetQuery("serverId", strconv.FormatInt(serverID, 10))
	params := batch_params{ttl: 60.0} // just use a default ttl value
	req, err = req.SetBody(params) 
	if err != nil {
		return BatchMetadata{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return BatchMetadata{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return BatchMetadata{}, WithStack(err)
	}
	var result BatchMetadata
	if err := resp.ParseBody("", &result); err != nil {
		return BatchMetadata{}, WithStack(err)
	}
	return result, nil
}

// DeleteBatch deletes an existing dump batch
func (c *client) DeleteBatch(ctx context.Context, db Database, batchID string) error {
	req, err := c.conn.NewRequest("DELETE", path.Join("_db", db.Name(), "_api/replication/batch", batchID))
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(204); err != nil {
		return WithStack(err)
	}
	return nil
}

// Get the inventory of a server containing all collections (with entire details) of a database.
func (c *client) DatabaseInventory(ctx context.Context, db Database) (DatabaseInventory, error) {
	req, err := c.conn.NewRequest("GET", path.Join("_db", db.Name(), "_api/replication/inventory"))
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
