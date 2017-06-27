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

// newCollection creates a new Collection implementation.
func newCollection(name string, db *database) (Collection, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &collection{
		name: name,
		db:   db,
		conn: db.conn,
	}, nil
}

type collection struct {
	name string
	db   *database
	conn Connection
}

// relPath creates the relative path to this collection (`_db/<db-name>/_api/<api-name>/<col-name>`)
func (c *collection) relPath(apiName string) string {
	escapedName := pathEscape(c.name)
	return path.Join(c.db.relPath(), "_api", apiName, escapedName)
}

// Name returns the name of the collection.
func (c *collection) Name() string {
	return c.name
}

// Status fetches the current status of the collection.
func (c *collection) Status(ctx context.Context) (CollectionStatus, error) {
	req, err := c.conn.NewRequest("GET", c.relPath("collection"))
	if err != nil {
		return CollectionStatus(0), WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return CollectionStatus(0), WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return CollectionStatus(0), WithStack(err)
	}
	var data CollectionInfo
	if err := resp.ParseBody("", &data); err != nil {
		return CollectionStatus(0), WithStack(err)
	}
	return data.Status, nil
}

// Count fetches the number of document in the collection.
func (c *collection) Count(ctx context.Context) (int64, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("collection"), "count"))
	if err != nil {
		return 0, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return 0, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return 0, WithStack(err)
	}
	var data struct {
		Count int64 `arangodb:"count,omitempty"`
	}
	if err := resp.ParseBody("", &data); err != nil {
		return 0, WithStack(err)
	}
	return data.Count, nil
}

// Statistics returns the number of documents and additional statistical information about the collection.
func (c *collection) Statistics(ctx context.Context) (CollectionStatistics, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("collection"), "figures"))
	if err != nil {
		return CollectionStatistics{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return CollectionStatistics{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return CollectionStatistics{}, WithStack(err)
	}
	var data CollectionStatistics
	if err := resp.ParseBody("", &data); err != nil {
		return CollectionStatistics{}, WithStack(err)
	}
	return data, nil
}

// Revision fetches the revision ID of the collection.
// The revision ID is a server-generated string that clients can use to check whether data
// in a collection has changed since the last revision check.
func (c *collection) Revision(ctx context.Context) (string, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("collection"), "revision"))
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	var data struct {
		Revision string `arangodb:"revision,omitempty"`
	}
	if err := resp.ParseBody("", &data); err != nil {
		return "", WithStack(err)
	}
	return data.Revision, nil
}

// Properties fetches extended information about the collection.
func (c *collection) Properties(ctx context.Context) (CollectionProperties, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("collection"), "properties"))
	if err != nil {
		return CollectionProperties{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return CollectionProperties{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return CollectionProperties{}, WithStack(err)
	}
	var data CollectionProperties
	if err := resp.ParseBody("", &data); err != nil {
		return CollectionProperties{}, WithStack(err)
	}
	return data, nil
}

// SetProperties changes properties of the collection.
func (c *collection) SetProperties(ctx context.Context, options SetCollectionPropertiesOptions) error {
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath("collection"), "properties"))
	if err != nil {
		return WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Load the collection into memory.
func (c *collection) Load(ctx context.Context) error {
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath("collection"), "load"))
	if err != nil {
		return WithStack(err)
	}
	opts := struct {
		Count bool `arangodb:"count"`
	}{
		Count: false,
	}
	if _, err := req.SetBody(opts); err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// UnLoad the collection from memory.
func (c *collection) Unload(ctx context.Context) error {
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath("collection"), "unload"))
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil

}

// Remove removes the entire collection.
// If the collection does not exist, a NotFoundError is returned.
func (c *collection) Remove(ctx context.Context) error {
	req, err := c.conn.NewRequest("DELETE", c.relPath("collection"))
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Truncate removes all documents from the collection, but leaves the indexes intact.
func (c *collection) Truncate(ctx context.Context) error {
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath("collection"), "truncate"))
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}
