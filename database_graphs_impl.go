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

// Graph opens a connection to an existing graph within the database.
// If no graph with given name exists, an NotFoundError is returned.
func (d *database) Graph(ctx context.Context, name string) (Graph, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/gharial", escapedName))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	g, err := newGraph(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return g, nil
}

// GraphExists returns true if a graph with given name exists within the database.
func (d *database) GraphExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/gharial", escapedName))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
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

type getGraphsResponse struct {
	Graphs []DocumentMeta `json:"graphs,omitempty"`
}

// Graphs returns a list of all graphs in the database.
func (d *database) Graphs(ctx context.Context) ([]Graph, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/gharial"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getGraphsResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Graph, 0, len(data.Graphs))
	for _, info := range data.Graphs {
		g, err := newGraph(info.Key, d)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, g)
	}
	return result, nil
}

type createGraphOptions struct {
	Name                    string                        `json:"name"`
	OrphanVertexCollections []string                      `json:"orphanCollections,omitempty"`
	EdgeDefinitions         []EdgeDefinition              `json:"edgeDefinitions,omitempty"`
	IsSmart                 bool                          `json:"isSmart,omitempty"`
	Options                 *createGraphAdditionalOptions `json:"options,omitempty"`
}

type createGraphAdditionalOptions struct {
	// SmartGraphAttribute is the attribute name that is used to smartly shard the vertices of a graph.
	// Every vertex in this Graph has to have this attribute.
	// Cannot be modified later.
	SmartGraphAttribute string `json:"smartGraphAttribute,omitempty"`
	// NumberOfShards is the number of shards that is used for every collection within this graph.
	// Cannot be modified later.
	NumberOfShards int `json:"numberOfShards,omitempty"`
}

// CreateGraph creates a new graph with given name and options, and opens a connection to it.
// If a graph with given name already exists within the database, a DuplicateError is returned.
func (d *database) CreateGraph(ctx context.Context, name string, options *CreateGraphOptions) (Graph, error) {
	input := createGraphOptions{
		Name: name,
	}
	if options != nil {
		input.OrphanVertexCollections = options.OrphanVertexCollections
		input.EdgeDefinitions = options.EdgeDefinitions
		input.IsSmart = options.IsSmart
		if options.SmartGraphAttribute != "" || options.NumberOfShards != 0 {
			input.Options = &createGraphAdditionalOptions{
				SmartGraphAttribute: options.SmartGraphAttribute,
				NumberOfShards:      options.NumberOfShards,
			}
		}
	}
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/gharial"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, WithStack(err)
	}
	g, err := newGraph(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return g, nil
}
