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
func newGraph(graphData graphData, db *database) (Graph, error) {
	if graphData.Name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &graph{
		graphData: graphData,
		db:        db,
		conn:      db.conn,
	}, nil
}

type graph struct {
	graphData
	db   *database
	conn Connection
}

// relPath creates the relative path to this graph (`_db/<db-name>/_api/gharial/<graph-name>`)
func (g *graph) relPath() string {
	escapedName := pathEscape(g.graphData.Name)
	return path.Join(g.db.relPath(), "_api", "gharial", escapedName)
}

// Name returns the name of the graph.
func (g *graph) Name() string {
	return g.graphData.Name
}

// ID returns the id of the graph.
func (g *graph) ID() string {
	return g.graphData.ID
}

// Key returns the key of the graph.
func (g *graph) Key() DocumentID {
	return g.graphData.Key
}

// Key returns the key of the graph.
func (g *graph) Rev() string {
	return g.graphData.Rev
}

// EdgeDefinitions returns the edge definitions of the graph.
func (g *graph) EdgeDefinitions() []EdgeDefinition {
	return g.graphData.EdgeDefinitions
}

// IsSmart returns the isSmart setting of the graph.
func (g *graph) IsSmart() bool {
	return g.graphData.IsSmart
}

// MinReplicationFactor returns the minimum replication factor for the graph.
func (g *graph) MinReplicationFactor() int {
	return g.graphData.MinReplicationFactor
}

// NumberOfShards returns the number of shards for the graph.
func (g *graph) NumberOfShards() int {
	return g.graphData.NumberOfShards
}

// OrphanCollections returns the orphan collcetions of the graph.
func (g *graph) OrphanCollections() []string {
	return g.graphData.OrphanCollections
}

// ReplicationFactor returns the current replication factor.
func (g *graph) ReplicationFactor() int {
	return g.graphData.ReplicationFactor
}

// WriteConcern returns the write concern setting of the graph.
func (g *graph) WriteConcern() int {
	return g.graphData.WriteConcern
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
