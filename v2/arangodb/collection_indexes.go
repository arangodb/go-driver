//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
// Author Jakub Wierzbowski
//

package arangodb

import (
	"context"
)

// CollectionIndexes provides access to the indexes in a single collection.
type CollectionIndexes interface {
	// Index opens a connection to an existing index within the collection.
	// If no index with given name exists, an NotFoundError is returned.
	Index(ctx context.Context, name string) (IndexResponse, error)

	// IndexExists returns true if an index with given name exists within the collection.
	IndexExists(ctx context.Context, name string) (bool, error)

	// Indexes returns a list of all indexes in the collection.
	Indexes(ctx context.Context) ([]IndexResponse, error)

	// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// NOTE: 'hash' and 'skiplist' being mere aliases for the persistent index type nowadays
	EnsurePersistentIndex(ctx context.Context, fields []string, options *CreatePersistentIndexOptions) (IndexResponse, bool, error)

	// EnsureGeoIndex creates a hash index in the collection, if it does not already exist.
	// Fields is a slice with one or two attribute paths. If it is a slice with one attribute path location,
	// then a geo-spatial index on all documents is created using location as path to the coordinates.
	// The value of the attribute must be a slice with at least two double values. The slice must contain the latitude (first value)
	// and the longitude (second value). All documents, which do not have the attribute path or with value that are not suitable, are ignored.
	// If it is a slice with two attribute paths latitude and longitude, then a geo-spatial index on all documents is created
	// using latitude and longitude as paths the latitude and the longitude. The value of the attribute latitude and of the
	// attribute longitude must a double. All documents, which do not have the attribute paths or which values are not suitable, are ignored.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureGeoIndex(ctx context.Context, fields []string, options *CreateGeoIndexOptions) (IndexResponse, bool, error)

	// EnsureTTLIndex creates a TLL collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureTTLIndex(ctx context.Context, fields []string, expireAfter int, options *CreateTTLIndexOptions) (IndexResponse, bool, error)

	// EnsureZKDIndex creates a ZKD multi-dimensional index for the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureZKDIndex(ctx context.Context, fields []string, options *CreateZKDIndexOptions) (IndexResponse, bool, error)

	// EnsureInvertedIndex creates an inverted index in the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// Available in ArangoDB 3.10 and later.
	// InvertedIndexOptions is an obligatory parameter and must contain at least `Fields` field
	EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (IndexResponse, bool, error)

	// DeleteIndex deletes an index from the collection.
	DeleteIndex(ctx context.Context, name string) error

	// DeleteIndexById deletes an index from the collection.
	DeleteIndexById(ctx context.Context, id string) error
}

// IndexType represents an index type as string
type IndexType string

const (
	// PrimaryIndexType  is automatically created for each collections. It indexes the documents’ primary keys,
	//  which are stored in the _key system attribute. The primary index is unique and can be used for queries on both the _key and _id attributes.
	// There is no way to explicitly create or delete primary indexes.
	PrimaryIndexType = IndexType("primary")

	// EdgeIndexType is automatically created for edge collections. It contains connections between vertex documents
	// and is invoked when the connecting edges of a vertex are queried. There is no way to explicitly create or delete edge indexes.
	// The edge index is non-unique.
	EdgeIndexType = IndexType("edge")

	// PersistentIndexType is a sorted index that can be used for finding individual documents or ranges of documents.
	PersistentIndexType = IndexType("persistent")

	// GeoIndexType index can accelerate queries that filter and sort by the distance between stored coordinates and coordinates provided in a query.
	GeoIndexType = IndexType("geo")

	// TTLIndexType index can be used for automatically removing expired documents from a collection.
	// Documents which are expired are eventually removed by a background thread.
	TTLIndexType = IndexType("ttl")

	// ZKDIndexType == multi-dimensional index. The zkd index type is an experimental index for indexing two- or higher dimensional data such as time ranges,
	// for efficient intersection of multiple range queries.
	ZKDIndexType = IndexType("zkd")

	// InvertedIndexType can be used to speed up a broad range of AQL queries, from simple to complex, including full-text search
	InvertedIndexType = IndexType("inverted")

	/*** DEPRECATED INDEXES ***/

	// FullTextIndex - Deprecated: since 3.10 version. Use ArangoSearch view instead.
	FullTextIndex = IndexType("fulltext")

	// HashIndex are an aliases for the persistent index type and should no longer be used to create new indexes.
	// The aliases will be removed in a future version.
	HashIndex = IndexType("hash")

	// SkipListIndex are an aliases for the persistent index type and should no longer be used to create new indexes.
	// The aliases will be removed in a future version.
	SkipListIndex = IndexType("skiplist")

	/*** DEPRECATED INDEXES ***/
)

// IndexResponse is the response from the Index list method
type IndexResponse struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// Type returns the type of the index
	Type IndexType `json:"type"`

	IndexSharedOptions `json:",inline"`

	// RegularIndex is the regular index object. It is empty for the InvertedIndex type.
	RegularIndex *IndexOptions `json:"indexes"`

	// InvertedIndex is the inverted index object. It is not empty only for InvertedIndex type.
	InvertedIndex *InvertedIndexOptions `json:"invertedIndexes"`
}

// IndexSharedOptions contains options that are shared between all index types
type IndexSharedOptions struct {
	// ID returns the ID of the index. Effectively this is `<collection-name>/<index.Name()>`.
	ID string `json:"id,omitempty"`

	// Unique is supported by persistent indexes. By default, all user-defined indexes are non-unique.
	// Only the attributes in fields are checked for uniqueness.
	// Any attributes in from storedValues are not checked for their uniqueness.
	Unique *bool `json:"unique,omitempty"`

	// Sparse You can control the sparsity for persistent indexes.
	// The inverted, fulltext, and geo index types are sparse by definition.
	Sparse *bool `json:"sparse,omitempty"`

	// IsNewlyCreated returns if this index was newly created or pre-existing.
	IsNewlyCreated *bool `json:"isNewlyCreated,omitempty"`
}

// IndexOptions contains the information about an regular index type
type IndexOptions struct {
	// Fields returns a list of attributes of this index.
	Fields []string `json:"fields,omitempty"`

	// Estimates determines if the to-be-created index should maintain selectivity estimates or not - PersistentIndex only
	Estimates *bool `json:"estimates,omitempty"`

	// SelectivityEstimate determines the selectivity estimate value of the index - PersistentIndex only
	SelectivityEstimate float64 `json:"selectivityEstimate,omitempty"`

	// MinLength returns min length for this index if set.
	MinLength *int `json:"minLength,omitempty"`

	// Deduplicate returns deduplicate setting of this index.
	Deduplicate *bool `json:"deduplicate,omitempty"`

	// ExpireAfter returns an expiry after for this index if set.
	ExpireAfter *int `json:"expireAfter,omitempty"`

	// CacheEnabled if true, then the index will be cached in memory. Caching is turned off by default.
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`

	// StoredValues returns a list of stored values for this index - PersistentIndex only
	StoredValues []string `json:"storedValues,omitempty"`

	// GeoJSON returns if geo json was set for this index or not.
	GeoJSON *bool `json:"geoJson,omitempty"`

	// LegacyPolygons returns if legacy polygons was set for this index or not before 3.10 - GeoIndex only
	LegacyPolygons *bool `json:"legacyPolygons,omitempty"`
}

// CreatePersistentIndexOptions contains specific options for creating a persistent index.
// Note: "hash" and "skiplist" are only aliases for "persistent" with the RocksDB storage engine which is only storage engine since 3.7
type CreatePersistentIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// CacheEnabled if true, then the index will be cached in memory. Caching is turned off by default.
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`

	// StoreValues if true, then the additional attributes will be included.
	// These additional attributes cannot be used for index lookups or sorts, but they can be used for projections.
	// There must be no overlap of attribute paths between `fields` and `storedValues`. The maximum number of values is 32.
	StoredValues []string `json:"storedValues,omitempty"`

	// Sparse You can control the sparsity for persistent indexes.
	// The inverted, fulltext, and geo index types are sparse by definition.
	Sparse *bool `json:"sparse,omitempty"`

	// Unique is supported by persistent indexes. By default, all user-defined indexes are non-unique.
	// Only the attributes in fields are checked for uniqueness.
	// Any attributes in from storedValues are not checked for their uniqueness.
	Unique *bool `json:"unique,omitempty"`

	// Deduplicate is supported by array indexes of type persistent. It controls whether inserting duplicate index
	// values from the same document into a unique array index will lead to a unique constraint error or not.
	// The default value is true, so only a single instance of each non-unique index value will be inserted into
	// the index per document.
	// Trying to insert a value into the index that already exists in the index will always fail,
	// regardless of the value of this attribute.
	Deduplicate *bool `json:"deduplicate,omitempty"`

	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	// Is supported by indexes of type persistent
	// This attribute controls whether index selectivity estimates are maintained for the index.
	// Not maintaining index selectivity estimates can have a slightly positive impact on write performance.
	// The downside of turning off index selectivity estimates will be that the query optimizer will not be able
	// to determine the usefulness of different competing indexes in AQL queries when there are multiple candidate
	// indexes to choose from. The estimates attribute is optional and defaults to true if not set.
	// It will have no effect on indexes other than persistent (with hash and skiplist being mere aliases for the persistent index type nowadays).
	Estimates *bool
}

// CreateGeoIndexOptions contains specific options for creating a geo index.
type CreateGeoIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// If a geo-spatial index on a location is constructed and GeoJSON is true, then the order within the array
	// is longitude followed by latitude. This corresponds to the format described in http://geojson.org/geojson-spec.html#positions
	GeoJSON *bool `json:"geoJson,omitempty"`

	// LegacyPolygons determines if the to-be-created index should use legacy polygons or not.
	// It is relevant for those that have geoJson set to true only.
	// Old geo indexes from versions from below 3.10 will always implicitly have the legacyPolygons option set to true.
	// Newly generated geo indexes from 3.10 on will have the legacyPolygons option by default set to false,
	// however, it can still be explicitly overwritten with true to create a legacy index but is not recommended.
	LegacyPolygons *bool `json:"legacyPolygons,omitempty"`
}

// CreateTTLIndexOptions provides specific options for creating a TTL index
type CreateTTLIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`
}

// CreateZKDIndexOptions provides specific options for creating a ZKD index
type CreateZKDIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// FieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
	FieldValueTypes string `json:"fieldValueTypes,required"`
}
