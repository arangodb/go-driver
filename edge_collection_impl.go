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

// relPath creates the relative path to this edge collection (`_db/<db-name>/_api/gharial/<graph-name>/edge/<collection-name>`)
func (c *edgeCollection) relPath() string {
	escapedName := pathEscape(c.name)
	return path.Join(c.g.relPath(), "edge", escapedName)
}

// Name returns the name of the edge collection.
func (c *edgeCollection) Name() string {
	return c.name
}

// CreateEdge creates a new edge in this edge collection.
func (c *edgeCollection) CreateEdge(ctx context.Context, from, to DocumentID, document interface{}) (DocumentMeta, error) {
	return DocumentMeta{}, nil
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
