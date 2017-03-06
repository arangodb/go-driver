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

import "context"

// GraphEdgeCollections provides access to all edge collections of a single graph in a database.
type GraphEdgeCollections interface {
	// EdgeCollection opens a connection to an existing edge-collection within the graph.
	// If no edge-collection with given name exists, an NotFoundError is returned.
	EdgeCollection(ctx context.Context, name string) (EdgeCollection, error)

	// EdgeCollectionExists returns true if an edge-collection with given name exists within the graph.
	EdgeCollectionExists(ctx context.Context, name string) (bool, error)

	// EdgeCollections returns all edge collections of this graph
	EdgeCollections(ctx context.Context) ([]EdgeCollection, error)

	// CreateEdgeCollection creates an edge collection in the graph.
	// collection: The name of the edge collection to be used.
	// from: contains the names of one or more vertex collections that can contain source vertices.
	// to: contains the names of one or more edge collections that can contain target vertices.
	CreateEdgeCollection(ctx context.Context, collection string, from, to []string) (EdgeCollection, error)
}
