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

// newDatabase creates a new Database implementation.
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
	return path.Join(c.db.relPath(), "_api", apiName, c.name)
}

// Name returns the name of the collection.
func (c *collection) Name() string {
	return c.name
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

// ReadDocument reads a single document with given key from the collection.
// The document data is stored into result, the document meta data is returned.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("document"), key))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	//
	return nil
}

// CreateDocument creates a single document in the collection.
// The document data is loaded from the given document, the document meta data is returned.
// If the document data already contains a `_key` field, this will be used as key of the new document,
// otherwise a unique key is created.
// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
func (c *collection) CreateDocument(ctx context.Context, document interface{}) (DocumentMeta, error) {}

// UpdateDocument updates a single document with given key in the collection.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) UpdateDocument(ctx context.Context, key string, update map[string]interface{}) (DocumentMeta, error) {
}

// ReplaceDocument replaces a single document with given key in the collection with the document given in the document argument.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) ReplaceDocument(ctx context.Context, key string, update map[string]interface{}) (DocumentMeta, error) {
}

// RemoveDocument removes a single document with given key from the collection.
// The document meta data is returned.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) RemoveDocument(ctx context.Context, key string) (DocumentMeta, error) {}
