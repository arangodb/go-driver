//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import "context"

const (
	// SatelliteGraph is a special replication factor for satellite graphs.
	// Use this replication factor to create a satellite graph.
	SatelliteGraph = -100
)

type DatabaseGraph interface {
	// Graph opens a connection to an existing graph within the database.
	// If no graph with given name exists, an NotFoundError is returned.
	Graph(ctx context.Context, name string, options *GetGraphOptions) (Graph, error)

	// GraphExists returns true if a graph with given name exists within the database.
	GraphExists(ctx context.Context, name string) (bool, error)

	// Graphs return a list of all graphs in the database.
	Graphs(ctx context.Context) (GraphsResponseReader, error)

	// CreateGraph creates a new graph with given name and options, and opens a connection to it.
	// If a graph with given name already exists within the database, a DuplicateError is returned.
	CreateGraph(ctx context.Context, name string, graph *GraphDefinition, options *CreateGraphOptions) (Graph, error)
}

type GetGraphOptions struct {
	// SkipExistCheck skips checking if graph exists
	SkipExistCheck bool `json:"skipExistCheck,omitempty"`
}

type CreateGraphOptions struct {
	// Satellites An array of collection names that is used to create SatelliteCollections for a (Disjoint) SmartGraph
	// using SatelliteCollections (Enterprise Edition only). Each array element must be a string and a valid
	// collection name. The collection type cannot be modified later.
	Satellites []string `json:"satellites,omitempty"`
}

type GraphsResponseReader interface {
	// Read returns next Graph. If no Graph left, shared.NoMoreDocumentsError returned
	Read() (Graph, error)
}
