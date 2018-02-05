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
)

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
