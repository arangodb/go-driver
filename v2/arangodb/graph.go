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

// Graph provides access to all edge & vertex collections of a single graph in a database.
type Graph interface {
	// Name returns the name of the graph.
	Name() string

	// IsSmart Whether the graph is a SmartGraph (Enterprise Edition only).
	IsSmart() bool

	// IsSatellite Flag if the graph is a SatelliteGraph (Enterprise Edition only) or not.
	IsSatellite() bool

	// IsDisjoint Whether the graph is a Disjoint SmartGraph (Enterprise Edition only).
	IsDisjoint() bool

	// EdgeDefinitions returns the edge definitions of the graph.
	EdgeDefinitions() []EdgeDefinition

	// SmartGraphAttribute of the sharding attribute in the SmartGraph case (Enterprise Edition only).
	SmartGraphAttribute() string

	// NumberOfShards Number of shards created for every new collection in the graph.
	NumberOfShards() *int

	// OrphanCollections An array of additional vertex collections.
	// Documents in these collections do not have edges within this graph.
	OrphanCollections() []string

	// ReplicationFactor The replication factor used for every new collection in the graph.
	// For SatelliteGraphs, it is the string "satellite" (Enterprise Edition only).
	ReplicationFactor() int

	// WriteConcern The default write concern for new collections in the graph. It determines how many copies of each shard
	// are required to be in sync on the different DB-Servers. If there are less than these many copies in the cluster,
	// a shard refuses to write. Writes to shards with enough up-to-date copies succeed at the same time, however.
	// The value of writeConcern cannot be greater than replicationFactor. For SatelliteGraphs, the writeConcern is
	// automatically controlled to equal the number of DB-Servers and the attribute is not available. (cluster only)
	WriteConcern() *int

	// Remove the entire graph with options.
	Remove(ctx context.Context, opts *RemoveGraphOptions) error

	// GraphVertexCollections - Vertex collection functions
	GraphVertexCollections

	// GraphEdgesDefinition - Edge collection functions
	GraphEdgesDefinition
}

type RemoveGraphOptions struct {
	// Drop the collections of this graph as well. Collections are only dropped if they are not used in other graphs.
	DropCollections bool
}

type GraphDefinition struct {
	Name string `json:"name"`

	// IsSmart Whether the graph is a SmartGraph (Enterprise Edition only).
	IsSmart bool `json:"isSmart"`

	// IsSatellite Flag if the graph is a SatelliteGraph (Enterprise Edition only) or not.
	IsSatellite bool `json:"isSatellite"`

	// IsDisjoint Whether the graph is a Disjoint SmartGraph (Enterprise Edition only).
	IsDisjoint bool `json:"isDisjoint,omitempty"`

	// EdgeDefinitions An array of definitions for the relations of the graph
	EdgeDefinitions []EdgeDefinition `json:"edgeDefinitions,omitempty"`

	// NumberOfShards Number of shards created for every new collection in the graph.
	// For Satellite Graphs, it has to be set to 1
	NumberOfShards *int `json:"numberOfShards,omitempty"`

	// OrphanCollections An array of additional vertex collections.
	// Documents in these collections do not have edges within this graph.
	OrphanCollections []string `json:"orphanCollections,omitempty"`

	// WriteConcern The default write concern for new collections in the graph. It determines how many copies of each shard
	// are required to be in sync on the different DB-Servers. If there are less than these many copies in the cluster,
	// a shard refuses to write. Writes to shards with enough up-to-date copies succeed at the same time, however.
	// The value of writeConcern cannot be greater than replicationFactor. For SatelliteGraphs, the writeConcern is
	// automatically controlled to equal the number of DB-Servers and the attribute is not available. (cluster only)
	WriteConcern *int `json:"writeConcern,omitempty"`

	// ReplicationFactor The replication factor used for every new collection in the graph.
	// For SatelliteGraphs, it is the string "satellite" (Enterprise Edition only).
	ReplicationFactor graphReplicationFactor `json:"replicationFactor,omitempty"`

	// SmartGraphAttribute of the sharding attribute in the SmartGraph case (Enterprise Edition only).
	SmartGraphAttribute string `json:"smartGraphAttribute,omitempty"`
}

type EdgeDefinition struct {
	// Name of the edge collection, where the edges are stored in.
	Collection string `json:"collection"`

	// List of vertex collection names.
	// Edges in a collection can only be inserted if their _to is in any of the collections here.
	To []string `json:"to"`

	// List of vertex collection names.
	// Edges in a collection can only be inserted if their _from is in any of the collections here.
	From []string `json:"from"`
}
