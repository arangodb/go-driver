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

type GraphVertexCollections interface {
	// VertexCollection opens a connection to an existing vertex-collection within the graph.
	// If no vertex-collection with given name exists, an NotFoundError is returned.
	// Note: When calling Remove on the returned Collection, the collection is removed from the graph. Not from the database.
	VertexCollection(ctx context.Context, name string) (VertexCollection, error)

	// VertexCollectionExists returns true if a vertex-collection with given name exists within the graph.
	VertexCollectionExists(ctx context.Context, name string) (bool, error)

	// VertexCollections returns all vertex collections of this graph
	// Note: When calling Remove on any of the returned Collection's, the collection is removed from the graph. Not from the database.
	VertexCollections(ctx context.Context) ([]VertexCollection, error)

	// CreateVertexCollection creates a vertex collection in the graph
	CreateVertexCollection(ctx context.Context, name string, opts *CreateVertexCollectionOptions) (CreateVertexCollectionResponse, error)

	// DeleteVertexCollection Removes a vertex collection from the list of the graph’s orphan collections.
	// It can optionally delete the collection if it is not used in any other graph.
	// You cannot remove vertex collections that are used in one of the edge definitions of the graph.
	// You need to modify or remove the edge definition first to fully remove a vertex collection from the graph.
	DeleteVertexCollection(ctx context.Context, name string, opts *DeleteVertexCollectionOptions) (DeleteVertexCollectionResponse, error)
}

type CreateVertexCollectionOptions struct {
	// Satellites contain an array of collection names that will be used to create SatelliteCollections for
	// a Hybrid (Disjoint) SmartGraph (Enterprise Edition only)
	Satellites []string `json:"satellites,omitempty"`
}

type CreateVertexCollectionResponse struct {
	shared.ResponseStruct `json:",inline"`

	// GraphDefinition contains the updated graph definition
	GraphDefinition *GraphDefinition `json:"graph,omitempty"`

	VertexCollection
}

type DeleteVertexCollectionResponse struct {
	shared.ResponseStruct `json:",inline"`

	// GraphDefinition contains the updated graph definition
	GraphDefinition *GraphDefinition `json:"graph,omitempty"`
}

type DeleteVertexCollectionOptions struct {
	// Drop the collection as well. The collection is only dropped if it is not used in other graphs.
	DropCollection *bool
}
