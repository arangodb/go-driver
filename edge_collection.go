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

// EdgeCollection provides access to the edges of a single edge collection.
type EdgeCollection interface {
	// Name returns the name of the collection.
	Name() string

	// ReadEdge reads a single edge with given key from this edge collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadEdge(ctx context.Context, key string, result interface{}) (EdgeMeta, error)

	// CreateEdge creates a new edge in this edge collection.
	CreateEdge(ctx context.Context, from, to DocumentID, document interface{}) (DocumentMeta, error)

	// UpdateEdge updates a single edge with given key in the collection.
	// To & from are allowed to be empty. If they are empty, they are not updated.
	// The document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	UpdateEdge(ctx context.Context, key string, from, to DocumentID, update interface{}) (DocumentMeta, error)

	// ReplaceEdge replaces a single edge with given key in the collection.
	// The document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReplaceEdge(ctx context.Context, key string, from, to DocumentID, document interface{}) (DocumentMeta, error)

	// Remove the edge collection from the graph.
	Remove(ctx context.Context) error

	// Replace creates an edge collection in the graph.
	// from: contains the names of one or more vertex collections that can contain source vertices.
	// to: contains the names of one or more edge collections that can contain target vertices.
	Replace(ctx context.Context, from, to []string) error
}

// EdgeMeta is DocumentMeta data extended with From & To fields.
type EdgeMeta struct {
	DocumentMeta
	From DocumentID `json:"_from,omitempty"`
	To   DocumentID `json:"_to,omitempty"`
}
