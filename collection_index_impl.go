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

type indexData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type indexListResponse struct {
	Indexes []indexData `json:"indexes,omitempty"`
}

// Index opens a connection to an existing index within the collection.
// If no index with given name exists, an NotFoundError is returned.
func (c *collection) Index(ctx context.Context, name string) (Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data indexData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	idx, err := newIndex(data.ID, c)
	if err != nil {
		return nil, WithStack(err)
	}
	return idx, nil
}

// IndexExists returns true if an index with given name exists within the collection.
func (c *collection) IndexExists(ctx context.Context, name string) (bool, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

// Indexes returns a list of all indexes in the collection.
func (c *collection) Indexes(ctx context.Context) ([]Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.db.relPath(), "_api", "index"))
	if err != nil {
		return nil, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data indexListResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Index, 0, len(data.Indexes))
	for _, x := range data.Indexes {
		idx, err := newIndex(x.ID, c)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, idx)
	}
	return result, nil
}
