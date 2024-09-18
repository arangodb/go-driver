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

import (
	"context"

	"github.com/arangodb/go-driver/v2/connection"
)

const (
	// SatelliteGraph is a special replication factor for satellite graphs.
	// Use this replication factor to create a satellite graph.
	SatelliteGraph = -100
)

type EdgeDirection string

const (
	// EdgeDirectionIn selects inbound edges
	EdgeDirectionIn EdgeDirection = "in"
	// EdgeDirectionOut selects outbound edges
	EdgeDirectionOut EdgeDirection = "out"
)

type DatabaseGraph interface {
	// GetEdges returns inbound and outbound edge documents of a given vertex.
	// Requires Edge collection name and vertex ID
	GetEdges(ctx context.Context, name, vertex string, options *GetEdgesOptions) ([]EdgeDetails, error)

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

type GetEdgesOptions struct {
	// The direction of the edges. Allowed values are "in" and "out". If not set, edges in both directions are returned.
	Direction EdgeDirection `json:"direction,omitempty"`

	// Set this to true to allow the Coordinator to ask any shard replica for the data, not only the shard leader.
	// This may result in “dirty reads”.
	AllowDirtyReads *bool `json:"-"`
}

type EdgeDetails struct {
	DocumentMeta
	From  string `json:"_from"`
	To    string `json:"_to"`
	Label string `json:"$label"`
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

func (q *GetEdgesOptions) modifyRequest(r connection.Request) error {
	if q == nil {
		return nil
	}

	if q.AllowDirtyReads != nil {
		r.AddHeader(HeaderDirtyReads, boolToString(*q.AllowDirtyReads))
	}

	if q.Direction != "" {
		r.AddQuery(QueryDirection, string(q.Direction))
	}

	return nil
}
