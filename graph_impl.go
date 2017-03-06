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

// newGraph creates a new Graph implementation.
func newGraph(name string, db *database) (Graph, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &graph{
		name: name,
		db:   db,
		conn: db.conn,
	}, nil
}

type graph struct {
	name string
	db   *database
	conn Connection
}

// relPath creates the relative path to this graph (`_db/<db-name>/_api/gharial/<graph-name>`)
func (g *graph) relPath() string {
	escapedName := pathEscape(g.name)
	return path.Join(g.db.relPath(), "_api", "gharial", escapedName)
}

// Name returns the name of the graph.
func (g *graph) Name() string {
	return g.name
}

// Remove removes the entire graph.
// If the graph does not exist, a NotFoundError is returned.
func (g *graph) Remove(ctx context.Context) error {
	req, err := g.conn.NewRequest("DELETE", g.relPath())
	if err != nil {
		return WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return WithStack(err)
	}
	return nil
}
