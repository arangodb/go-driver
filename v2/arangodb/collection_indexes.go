//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
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

	// IndexesWithOptions returns a list of all indexes in the collection.
	// Pass WithHidden to include indexes that are not yet fully built and, from
	// ArangoDB 3.12.10, per-shard vector index details (shards / resolvedNLists).
	IndexesWithOptions(ctx context.Context, opts *IndexListOptions) ([]IndexResponse, error)

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
	// expireAfter is the time interval (in seconds) from the point in time stored in the fields attribute after which the documents count as expired.
	// Can be set to 0 to let documents expire as soon as the server time passes the point in time stored in the document attribute, or to a higher number to delay the expiration.
	// fields The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureTTLIndex(ctx context.Context, fields []string, expireAfter int, options *CreateTTLIndexOptions) (IndexResponse, bool, error)

	// EnsureMDIIndex creates a multidimensional index for the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// Available in ArangoDB 3.12 and later.
	EnsureMDIIndex(ctx context.Context, fields []string, options *CreateMDIIndexOptions) (IndexResponse, bool, error)

	// EnsureMDIPrefixedIndex creates is an additional index variant of mdi index that lets you specify additional
	// attributes for the index to narrow down the search space using equality checks.
	// Available in ArangoDB 3.12 and later.
	EnsureMDIPrefixedIndex(ctx context.Context, fields []string, options *CreateMDIPrefixedIndexOptions) (IndexResponse, bool, error)

	// EnsureInvertedIndex creates an inverted index in the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// Available in ArangoDB 3.10 and later.
	// InvertedIndexOptions is an obligatory parameter and must contain at least `Fields` field
	EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (IndexResponse, bool, error)

	// DeleteIndex deletes an index from the collection.
	DeleteIndex(ctx context.Context, name string) error

	// DeleteIndexByID deletes an index from the collection.
	DeleteIndexByID(ctx context.Context, id string) error

	// EnsureVectorIndex creates a vector index in the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// Available in ArangoDB 3.12.4 and later.
	// VectorParams is required and must contain at least Dimension and Metric.
	// From ArangoDB 3.12.10, NLists is optional (the server applies a default scaling specification).
	// Up to 3.12.9, NLists must be a positive number.
	// CreateVectorIndexOptions.InBackground is sent when set; see CreateVectorIndexOptions.InBackground.
	EnsureVectorIndex(ctx context.Context, fields []string, params *VectorParams, options *CreateVectorIndexOptions) (IndexResponse, bool, error)
}

// IndexType represents an index type as string
type IndexType string

const (
	// PrimaryIndexType is automatically created for each collection. It indexes the documents’ primary keys,
	// which are stored in the _key system attribute. The primary index is unique and can be used for queries on both the _key and _id attributes.
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

	// MDIIndexType is multidimensional index for indexing two- or higher dimensional data such as time ranges,
	// for efficient intersection of multiple range queries.
	// Available in ArangoDB 3.12 and later.
	MDIIndexType = IndexType("mdi")

	// MDIPrefixedIndexType is an additional `mdi` index variant that lets you specify additional attributes
	// for the index to narrow down the search space using equality checks.
	// Available in ArangoDB 3.12 and later.
	MDIPrefixedIndexType = IndexType("mdi-prefixed")

	// InvertedIndexType can be used to speed up a broad range of AQL queries, from simple to complex, including full-text search
	InvertedIndexType = IndexType("inverted")

	// Deprecated: Fulltext index type will be removed in ArangoDB v4.0. Use InvertedIndexType for full-text search. Kept for backward compatibility when reading index lists from older servers.
	FulltextIndexType = IndexType("fulltext")

	// VectorIndexType is used for efficient similarity searches on high-dimensional embeddings, enabling fast and scalable AI use cases.
	VectorIndexType = IndexType("vector")
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

	// VectorIndex is the vector index params. It is not empty only for VectorIndex type.
	VectorIndex *VectorParams `json:"params,omitempty"`

	// TrainingState is the vector index training state (e.g. unusable, training, ingesting, ready).
	// Available in ArangoDB 3.12.9 and later.
	TrainingState *VectorIndexTrainingState `json:"trainingState,omitempty"`

	// VectorErrorMessage is present for unusable vector indexes.
	// It is populated for vector indexes from the index object's `errorMessage`.
	VectorErrorMessage *string `json:"errorMessage,omitempty"`

	// VectorFields is the indexed attribute path for vector indexes (always one field).
	VectorFields []string `json:"-"`

	// VectorStoredValues is the list of additionally stored attribute paths for vector indexes.
	VectorStoredValues []string `json:"-"`

	// VectorShards holds per-shard vector index details (trainingState, error, resolvedNLists).
	// Available in ArangoDB 3.12.10 and later when listing indexes with WithHidden.
	VectorShards map[string]VectorIndexShardInfo `json:"shards,omitempty"`
}

// IndexListOptions controls GET /_api/index listing.
type IndexListOptions struct {
	// WithHidden includes indexes that are not yet fully built and, from ArangoDB 3.12.10,
	// per-shard vector index details (shards / resolvedNLists).
	WithHidden *bool
}

// IndexSharedOptions contains options that are shared between all index types
type IndexSharedOptions struct {
	// ID returns the ID of the index. Effectively this is `<collection-name>/<index.Name()>`.
	ID string `json:"id,omitempty"`

	// Unique is supported by persistent indexes. By default, all user-defined indexes are non-unique.
	// Only the attributes in fields are checked for uniqueness.
	// Any attributes in from storedValues are not checked for their uniqueness.
	Unique *bool `json:"unique,omitempty"`

	// Sparse controls whether the index excludes documents with the indexed attribute missing or null.
	// Used by persistent indexes. Inverted and geo index types are sparse by definition.
	// Fulltext indexes (deprecated, removed in v4.0) also used this option.
	// Vector indexes use CreateVectorIndexOptions.Sparse when creating.
	Sparse *bool `json:"sparse,omitempty"`

	// IsNewlyCreated returns if this index was newly created or pre-existing.
	IsNewlyCreated *bool `json:"isNewlyCreated,omitempty"`
}

// IndexOptions contains the information about a regular index type
type IndexOptions struct {
	// Fields returns a list of attributes of this index.
	Fields []string `json:"fields,omitempty"`

	// Estimates determines if the to-be-created index should maintain selectivity estimates or not - PersistentIndex only
	Estimates *bool `json:"estimates,omitempty"`

	// SelectivityEstimate determines the selectivity estimate value of the index - PersistentIndex only
	SelectivityEstimate float64 `json:"selectivityEstimate,omitempty"`

	// Deprecated: MinLength is only relevant for fulltext indexes, which are planned to be removed in ArangoDB v4.0.
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

	// Sparse controls whether the index excludes documents with the indexed attribute missing or null.
	// Used by persistent indexes. (Vector indexes have their own Sparse in CreateVectorIndexOptions.)
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

	// Estimates determines if the to-be-created index should maintain selectivity estimates or not.
	// Is supported by indexes of type persistent
	// This attribute controls whether index selectivity estimates are maintained for the index.
	// Not maintaining index selectivity estimates can have a slightly positive impact on write performance.
	// The downside of turning off index selectivity estimates will be that the query optimizer will not be able
	// to determine the usefulness of different competing indexes in AQL queries when there are multiple candidate
	// indexes to choose from. The estimates attribute is optional and defaults to true if not set.
	// It will have no effect on indexes other than persistent (with hash and skiplist being mere aliases for the persistent index type nowadays).
	Estimates *bool `json:"estimates,omitempty"`

	// InBackground You can set this option to true to create the index in the background,
	// which will not write-lock the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground *bool `json:"inBackground,omitempty"`
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

	// InBackground You can set this option to true to create the index in the background,
	// which will not write-lock the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground *bool `json:"inBackground,omitempty"`
}

// CreateTTLIndexOptions provides specific options for creating a TTL index
type CreateTTLIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// InBackground You can set this option to true to create the index in the background,
	// which will not write-lock the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground *bool `json:"inBackground,omitempty"`
}

type MDIFieldType string

const MDIDoubleFieldType MDIFieldType = "double"

// CreateMDIIndexOptions provides specific options for creating a MKD index
type CreateMDIIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// FieldValueTypes is required and the only allowed value is "double".
	//Future extensions of the index will allow other types.
	FieldValueTypes MDIFieldType `json:"fieldValueTypes,required"`

	// Unique if true, then create a unique index.
	Unique *bool `json:"unique,omitempty"`

	// Sparse If `true`, then create a sparse index to exclude documents from the index that do not have the defined
	// attributes or are explicitly set to `null` values. If a non-value is set, it still needs to be numeric.
	Sparse *bool `json:"sparse,omitempty"`

	// InBackground You can set this option to true to create the index in the background,
	// which will not write-lock the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground *bool `json:"inBackground,omitempty"`

	// StoredValues The optional `storedValues` attribute can contain an array of paths to additional attributes to
	// store in the index. These additional attributes cannot be used for index lookups or for sorting, but they can
	// be used for projections. This allows an index to fully cover more queries and avoid extra document lookups.
	// The maximum number of attributes in `storedValues` is 32.
	//
	// Attributes in `storedValues` cannot overlap with attributes specified in `prefixFields` but you can have
	// the attributes in both `storedValues` and `fields`.
	StoredValues []string `json:"storedValues,omitempty"`
}

type CreateMDIPrefixedIndexOptions struct {
	CreateMDIIndexOptions `json:",inline"`

	// PrefixFields is required and contains nn array of attribute names used as search prefix.
	// Array expansions are not allowed.
	PrefixFields []string `json:"prefixFields,required"`
}

type CreateVectorIndexOptions struct {
	// InBackground, when true, keeps the collection available for writes during index creation
	// by not taking an exclusive write lock. Training is deferred until enough data is available.
	// When false or unset, the call waits until training finishes (timeouts may still occur).
	//
	// Up to v3.12.9, a failed training (for example not enough data) returns an error while
	// still leaving an index in "unusable" state.
	// From v3.12.10 onward, the same case returns success with TrainingState "unusable" and
	// the reason in VectorErrorMessage. Inspect those fields on a successful create.
	InBackground *bool `json:"inBackground,omitempty"`
	// Optional index name.
	Name *string `json:"name,omitempty"`
	// Number of threads to use for index creation.
	Parallelism *int `json:"parallelism,omitempty"`
	// Exclude docs missing the field.
	Sparse *bool `json:"sparse,omitempty"`
	// Introduced in v3.12.7. Up to 32 additional attributes can be stored in the index.
	// Up to v3.12.9 these are used for filtering. From v3.12.10 they are also used to
	// cover projections so attributes can be returned from the index without materializing documents.
	StoredValues []string `json:"storedValues,omitempty"`
}

type VectorIndexTrainingState string

const (
	VectorIndexTrainingStateUnusable  VectorIndexTrainingState = "unusable"
	VectorIndexTrainingStateTraining  VectorIndexTrainingState = "training"
	VectorIndexTrainingStateIngesting VectorIndexTrainingState = "ingesting"
	VectorIndexTrainingStateReady     VectorIndexTrainingState = "ready"
)

type VectorParams struct {
	// Neighbors considered in search.
	DefaultNProbe *int `json:"defaultNProbe,omitempty"`
	// Vector length.
	Dimension *int `json:"dimension,omitempty"`
	// Faiss factory string. From v3.12.10 you can use a {} placeholder for the
	// centroid count, e.g. "IVF{}_HNSW32,SQ8". It is substituted with the number
	// of centroids that nLists resolves to (per shard in cluster deployments).
	Factory *string `json:"factory,omitempty"`
	// Similarity measure.
	Metric *VectorMetric `json:"metric,omitempty"`
	// Number of Voronoi cells / centroids. Up to v3.12.9 this must be a positive
	// integer. From v3.12.10 it is optional and may be a fixed number or a scaling
	// specification. If omitted, the server uses a default autoSqrt scaling specification.
	// Use NewVectorNLists for a fixed count or NewVectorNListsScaling for scaling mode.
	NLists *VectorNLists `json:"nLists,omitempty"`
	// How many vectors per centroid to include in the training sample.
	// Introduced in v3.12.10. Must be 1 or greater. Server default is 100.
	// Up to v3.12.9 this is not configurable and a fixed value of 256 is used.
	NumberOfDocsPerCentroid *int `json:"numberOfDocsPerCentroid,omitempty"`
	// Faiss training iterations
	TrainingIterations *int `json:"trainingIterations,omitempty"`
}

// VectorNListsStrategy is how scaling-mode nLists computes the centroid count.
type VectorNListsStrategy string

const (
	// VectorNListsStrategyAutoSqrt computes max(minNLists, multiplier * sqrt(N)).
	VectorNListsStrategyAutoSqrt VectorNListsStrategy = "autoSqrt"
)

// VectorNLists is params.nLists: either a fixed centroid count or, from ArangoDB
// 3.12.10, a scaling specification. JSON is a number or an object.
type VectorNLists struct {
	// Fixed is a fixed number of centroids. Mutually exclusive with Scaling.
	Fixed *int
	// Scaling computes the number of centroids from the document count at training time (3.12.10+).
	Scaling *VectorNListsScaling
}

// VectorNListsScaling is the 3.12.10+ object form of nLists.
type VectorNListsScaling struct {
	// Strategy is how to compute the number of centroids if no tier applies.
	// The only available value is "autoSqrt".
	Strategy *VectorNListsStrategy `json:"strategy,omitempty"`
	// Multiplier is the factor for the autoSqrt strategy. Must be 1 or greater.
	Multiplier *int `json:"multiplier,omitempty"`
	// MinNLists is the lower bound for the computed number of centroids and the
	// document-count threshold that triggers training. Must be 1 or greater.
	MinNLists *int `json:"minNLists,omitempty"`
	// Tiers optionally pin a fixed centroid count for large document counts.
	// The tier with the highest threshold that is <= N wins.
	Tiers []VectorNListsTier `json:"tiers,omitempty"`
}

// VectorNListsTier pins a fixed centroid count once the document count reaches Threshold.
type VectorNListsTier struct {
	Threshold  *int `json:"threshold,omitempty"`
	FixedValue *int `json:"fixedValue,omitempty"`
}

// VectorIndexShardInfo is per-shard vector index status from GET /_api/index?withHidden=true (3.12.10+).
type VectorIndexShardInfo struct {
	TrainingState  *VectorIndexTrainingState `json:"trainingState,omitempty"`
	Error          *string                   `json:"error,omitempty"`
	ResolvedNLists *int                      `json:"resolvedNLists,omitempty"`
}

// NewVectorNLists returns a fixed-count nLists value.
func NewVectorNLists(n int) *VectorNLists {
	return &VectorNLists{Fixed: &n}
}

// NewVectorNListsScaling returns a scaling-mode nLists value (ArangoDB 3.12.10+).
func NewVectorNListsScaling(s VectorNListsScaling) *VectorNLists {
	cp := s
	return &VectorNLists{Scaling: &cp}
}

// VectorMetric defines the type of similarity metric for vector comparison.
type VectorMetric string

const (
	// Cosine similarity between vectors.
	VectorMetricCosine VectorMetric = "cosine"
	// Introduced in v3.12.6
	// Inner product similarity.
	VectorMetricInnerProduct VectorMetric = "innerProduct"
	// Euclidean (L2) distance between vectors.
	VectorMetricL2 VectorMetric = "l2"
)
