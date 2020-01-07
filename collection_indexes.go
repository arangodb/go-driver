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

// CollectionIndexes provides access to the indexes in a single collection.
type CollectionIndexes interface {
	// Index opens a connection to an existing index within the collection.
	// If no index with given name exists, an NotFoundError is returned.
	Index(ctx context.Context, name string) (Index, error)

	// IndexExists returns true if an index with given name exists within the collection.
	IndexExists(ctx context.Context, name string) (bool, error)

	// Indexes returns a list of all indexes in the collection.
	Indexes(ctx context.Context) ([]Index, error)

	// EnsureFullTextIndex creates a fulltext index in the collection, if it does not already exist.
	// Fields is a slice of attribute names. Currently, the slice is limited to exactly one attribute.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureFullTextIndex(ctx context.Context, fields []string, options *EnsureFullTextIndexOptions) (Index, bool, error)

	// EnsureGeoIndex creates a hash index in the collection, if it does not already exist.
	// Fields is a slice with one or two attribute paths. If it is a slice with one attribute path location,
	// then a geo-spatial index on all documents is created using location as path to the coordinates.
	// The value of the attribute must be a slice with at least two double values. The slice must contain the latitude (first value)
	// and the longitude (second value). All documents, which do not have the attribute path or with value that are not suitable, are ignored.
	// If it is a slice with two attribute paths latitude and longitude, then a geo-spatial index on all documents is created
	// using latitude and longitude as paths the latitude and the longitude. The value of the attribute latitude and of the
	// attribute longitude must a double. All documents, which do not have the attribute paths or which values are not suitable, are ignored.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureGeoIndex(ctx context.Context, fields []string, options *EnsureGeoIndexOptions) (Index, bool, error)

	// EnsureHashIndex creates a hash index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureHashIndex(ctx context.Context, fields []string, options *EnsureHashIndexOptions) (Index, bool, error)

	// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsurePersistentIndex(ctx context.Context, fields []string, options *EnsurePersistentIndexOptions) (Index, bool, error)

	// EnsureSkipListIndex creates a skiplist index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureSkipListIndex(ctx context.Context, fields []string, options *EnsureSkipListIndexOptions) (Index, bool, error)

	// EnsureTTLIndex creates a TLL collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureTTLIndex(ctx context.Context, field string, expireAfter int, options *EnsureTTLIndexOptions) (Index, bool, error)
}

// EnsureFullTextIndexOptions contains specific options for creating a full text index.
type EnsureFullTextIndexOptions struct {
	// MinLength is the minimum character length of words to index. Will default to a server-defined
	// value if unspecified (0). It is thus recommended to set this value explicitly when creating the index.
	MinLength int
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}

// EnsureGeoIndexOptions contains specific options for creating a geo index.
type EnsureGeoIndexOptions struct {
	// If a geo-spatial index on a location is constructed and GeoJSON is true, then the order within the array
	// is longitude followed by latitude. This corresponds to the format described in http://geojson.org/geojson-spec.html#positions
	GeoJSON bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}

// EnsureHashIndexOptions contains specific options for creating a hash index.
type EnsureHashIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// If true, de-duplication of array-values, before being added to the index, will be turned off.
	// This flag requires ArangoDB 3.2.
	// Note: this setting is only relevant for indexes with array fields (e.g. "fieldName[*]")
	NoDeduplicate bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}

// EnsurePersistentIndexOptions contains specific options for creating a persistent index.
type EnsurePersistentIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}

// EnsureSkipListIndexOptions contains specific options for creating a skip-list index.
type EnsureSkipListIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// If true, de-duplication of array-values, before being added to the index, will be turned off.
	// This flag requires ArangoDB 3.2.
	// Note: this setting is only relevant for indexes with array fields (e.g. "fieldName[*]")
	NoDeduplicate bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}

// EnsureTTLIndexOptions provides specific options for creating a TTL index
type EnsureTTLIndexOptions struct {
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
}
