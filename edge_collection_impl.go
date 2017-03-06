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
	"fmt"
	"path"
)

// newEdgeCollection creates a new EdgeCollection implementation.
func newEdgeCollection(name string, g *graph) (EdgeCollection, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if g == nil {
		return nil, WithStack(InvalidArgumentError{Message: "g is nil"})
	}
	return &edgeCollection{
		name: name,
		g:    g,
		conn: g.db.conn,
	}, nil
}

type edgeCollection struct {
	name string
	g    *graph
	conn Connection
}

type edgeDocument struct {
	From DocumentID `json:"_from,omitempty"`
	To   DocumentID `json:"_to,omitempty"`
}

// relPath creates the relative path to this edge collection (`_db/<db-name>/_api/gharial/<graph-name>/edge/<collection-name>`)
func (c *edgeCollection) relPath() string {
	escapedName := pathEscape(c.name)
	return path.Join(c.g.relPath(), "edge", escapedName)
}

// Name returns the name of the edge collection.
func (c *edgeCollection) Name() string {
	return c.name
}

// ReadEdge reads a single edge with given key from this edge collection.
// The document data is stored into result, the document meta data is returned.
// If no document exists with given key, a NotFoundError is returned.
func (c *edgeCollection) ReadEdge(ctx context.Context, key string, result interface{}) (EdgeMeta, error) {
	if err := validateKey(key); err != nil {
		return EdgeMeta{}, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath(), escapedKey))
	if err != nil {
		return EdgeMeta{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return EdgeMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return EdgeMeta{}, WithStack(err)
	}
	// Parse metadata
	var meta EdgeMeta
	if err := resp.ParseBody("edge", &meta); err != nil {
		return EdgeMeta{}, WithStack(err)
	}
	return meta, nil
}

// CreateEdge creates a new edge in this edge collection.
func (c *edgeCollection) CreateEdge(ctx context.Context, from, to DocumentID, document interface{}) (DocumentMeta, error) {
	if err := from.Validate(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("from invalid: %v", err)})
	}
	if err := to.Validate(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("to invalid: %v", err)})
	}
	req, err := c.conn.NewRequest("POST", c.relPath())
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(edgeDocument{To: to, From: from}, document); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("edge", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

// UpdateEdge updates a single edge with given key in the collection.
// To & from are allowed to be empty. If they are empty, they are not updated.
// The document meta data is returned.
// If no document exists with given key, a NotFoundError is returned.
func (c *edgeCollection) UpdateEdge(ctx context.Context, key string, from, to DocumentID, update interface{}) (DocumentMeta, error) {
	if err := from.ValidateOrEmpty(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("from invalid: %v", err)})
	}
	if err := to.ValidateOrEmpty(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("to invalid: %v", err)})
	}
	var edgeDoc *edgeDocument
	if !from.IsEmpty() || !to.IsEmpty() {
		edgeDoc = &edgeDocument{
			From: from,
			To:   to,
		}
	}
	req, err := c.conn.NewRequest("PATCH", c.relPath())
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(edgeDoc, update); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("edge", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *edgeCollection) ReplaceEdge(ctx context.Context, key string, from, to DocumentID, update interface{}) (DocumentMeta, error) {
	if err := from.Validate(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("from invalid: %v", err)})
	}
	if err := to.Validate(); err != nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("to invalid: %v", err)})
	}
	edgeDoc := edgeDocument{
		From: from,
		To:   to,
	}
	req, err := c.conn.NewRequest("PUT", c.relPath())
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(edgeDoc, update); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("edge", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

// Remove the edge collection from the graph.
func (c *edgeCollection) Remove(ctx context.Context) error {
	req, err := c.conn.NewRequest("DELETE", c.relPath())
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return WithStack(err)
	}
	return nil
}

// Replace creates an edge collection in the graph.
// collection: The name of the edge collection to be used.
// from: contains the names of one or more vertex collections that can contain source vertices.
// to: contains the names of one or more edge collections that can contain target vertices.
func (c *edgeCollection) Replace(ctx context.Context, from, to []string) error {
	req, err := c.conn.NewRequest("PUT", c.relPath())
	if err != nil {
		return WithStack(err)
	}
	input := EdgeDefinition{
		Collection: c.name,
		From:       from,
		To:         to,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return WithStack(err)
	}
	return nil
}
