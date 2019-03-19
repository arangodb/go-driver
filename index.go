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

// IndexType represents a index type as string
type IndexType string

// Symbolic constants for index types
const (
	PrimaryIndex    = IndexType("primary")
	FullTextIndex   = IndexType("fulltext")
	HashIndex       = IndexType("hash")
	SkipListIndex   = IndexType("skiplist")
	PersistentIndex = IndexType("persistent")
	GeoIndex        = IndexType("geo")
	EdgeIndex       = IndexType("edge")
)

// Index provides access to a single index in a single collection.
type Index interface {
	// Name returns the name of the index.
	Name() string

	// Type returns the type of the index
	Type() IndexType

	// Remove removes the entire index.
	// If the index does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error
}
