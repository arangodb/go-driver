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

type listEdgeCollectionResponse struct {
	Collections []string `json:"collections,omitempty"`
}

// EdgeCollection opens a connection to an existing edge-collection within the graph.
// If no edge-collection with given name exists, an NotFoundError is returned.
func (g *graph) EdgeCollection(ctx context.Context, name string) (EdgeCollection, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "edge"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data listEdgeCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	for _, n := range data.Collections {
		if n == name {
			ec, err := newEdgeCollection(name, g)
			if err != nil {
				return nil, WithStack(err)
			}
			return ec, nil
		}
	}
	return nil, WithStack(newArangoError(404, 0, "not found"))
}

// EdgeCollectionExists returns true if an edge-collection with given name exists within the graph.
func (g *graph) EdgeCollectionExists(ctx context.Context, name string) (bool, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "edge"))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return false, WithStack(err)
	}
	var data listEdgeCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return false, WithStack(err)
	}
	for _, n := range data.Collections {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}

// EdgeCollections returns all edge collections of this graph
func (g *graph) EdgeCollections(ctx context.Context) ([]EdgeCollection, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "edge"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data listEdgeCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]EdgeCollection, 0, len(data.Collections))
	for _, name := range data.Collections {
		ec, err := newEdgeCollection(name, g)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, ec)
	}
	return result, nil
}

// collection: The name of the edge collection to be used.
// from: contains the names of one or more vertex collections that can contain source vertices.
// to: contains the names of one or more edge collections that can contain target vertices.
func (g *graph) CreateEdgeCollection(ctx context.Context, collection string, from, to []string) (EdgeCollection, error) {
	req, err := g.conn.NewRequest("POST", path.Join(g.relPath(), "edge"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := EdgeDefinition{
		Collection: collection,
		From:       from,
		To:         to,
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, WithStack(err)
	}
	ec, err := newEdgeCollection(collection, g)
	if err != nil {
		return nil, WithStack(err)
	}
	return ec, nil
}
