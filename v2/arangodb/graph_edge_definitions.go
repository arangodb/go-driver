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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

type GraphEdgesDefinition interface {
	// EdgeDefinition opens a connection to an existing Edge collection within the graph.
	// If no Edge collection with given name exists, an NotFoundError is returned.
	// Note: When calling Remove on the returned Collection, the collection is removed from the graph. Not from the database.
	EdgeDefinition(ctx context.Context, collection string) (Edge, error)

	// EdgeDefinitionExists returns true if an Edge collection with given name exists within the graph.
	EdgeDefinitionExists(ctx context.Context, collection string) (bool, error)

	// GetEdgeDefinitions returns all Edge collections of this graph
	// Note: When calling Remove on any of the returned Collection's, the collection is removed from the graph. Not from the database.
	GetEdgeDefinitions(ctx context.Context) ([]Edge, error)

	// CreateEdgeDefinition creates an Edge collection in the graph
	// This edge definition has to contain a 'collection' and an array of each 'from' and 'to' vertex collections.
	// An edge definition can only be added if this definition is either not used in any other graph, or it is used
	// with exactly the same definition.
	// For example, it is not possible to store a definition “e” from “v1” to “v2” in one graph,
	// and “e” from “v2” to “v1” in another graph, but both can have “e” from “v1” to “v2”.
	CreateEdgeDefinition(ctx context.Context, collection string, from, to []string, opts *CreateEdgeDefinitionOptions) (CreateEdgeDefinitionResponse, error)

	// ReplaceEdgeDefinition Change one specific edge definition.
	// This modifies all occurrences of this definition in all graphs known to your database.
	ReplaceEdgeDefinition(ctx context.Context, collection string, from, to []string, opts *ReplaceEdgeOptions) (ReplaceEdgeDefinitionResponse, error)

	// DeleteEdgeDefinition Remove one edge definition from the graph.
	// This only removes the edge collection from the graph definition.
	// The vertex collections of the edge definition become orphan collections,
	// but otherwise remain untouched and can still be used in your queries.
	DeleteEdgeDefinition(ctx context.Context, collection string, opts *DeleteEdgeDefinitionOptions) (DeleteEdgeDefinitionResponse, error)
}

type CreateEdgeDefinitionOptions struct {
	// An array of collection names that is used to create SatelliteCollections for a (Disjoint) SmartGraph
	// using SatelliteCollections (Enterprise Edition only).
	// Each array element must be a string and a valid collection name. The collection type cannot be modified later.
	Satellites []string `json:"satellites,omitempty"`
}

type CreateEdgeDefinitionResponse struct {
	shared.ResponseStruct `json:",inline"`

	// GraphDefinition contains the updated graph definition
	GraphDefinition *GraphDefinition `json:"graph,omitempty"`

	Edge
}

type ReplaceEdgeOptions struct {
	// An array of collection names that is used to create SatelliteCollections for a (Disjoint) SmartGraph
	// using SatelliteCollections (Enterprise Edition only).
	// Each array element must be a string and a valid collection name. The collection type cannot be modified later.
	Satellites []string `json:"satellites,omitempty"`

	// Define if the request should wait until synced to disk.
	WaitForSync *bool `json:"-"`

	// Drop the collection as well. The collection is only dropped if it is not used in other graphs.
	DropCollection *bool `json:"-"`
}

type ReplaceEdgeDefinitionResponse struct {
	shared.ResponseStruct `json:",inline"`

	// GraphDefinition contains the updated graph definition
	GraphDefinition *GraphDefinition `json:"graph,omitempty"`

	Edge
}

type DeleteEdgeDefinitionOptions struct {
	// Drop the collection as well. The collection is only dropped if it is not used in other graphs.
	DropCollection *bool

	// Define if the request should wait until synced to disk.
	WaitForSync *bool
}

type DeleteEdgeDefinitionResponse struct {
	shared.ResponseStruct `json:",inline"`

	// GraphDefinition contains the updated graph definition
	GraphDefinition *GraphDefinition `json:"graph,omitempty"`
}
