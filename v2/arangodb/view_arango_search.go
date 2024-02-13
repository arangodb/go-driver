//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
)

// ArangoSearchView provides access to the information of a view.
// Views are only available in ArangoDB 3.4 and higher.
type ArangoSearchView interface {
	// View Includes generic View functions
	View

	// Properties fetches extended information about the view.
	Properties(ctx context.Context) (ArangoSearchViewProperties, error)

	// SetProperties Changes all properties of a View by replacing them.
	SetProperties(ctx context.Context, options ArangoSearchViewProperties) error

	// UpdateProperties Partially changes the properties of a View by updating the specified attributes.
	UpdateProperties(ctx context.Context, options ArangoSearchViewProperties) error
}

// ArangoSearchViewProperties contains properties of view with type 'arangosearch'
type ArangoSearchViewProperties struct {
	ViewBase

	// CleanupIntervalStep Wait at least this many commits between removing unused files in the ArangoSearch data
	// directory (default: 2, to disable use: 0). For the case where the consolidation policies merge segments
	// often (i.e. a lot of commit+consolidate), a lower value causes a lot of disk space to be wasted.
	// For the case where the consolidation policies rarely merge segments (i.e. few inserts/deletes),
	// a higher value impacts performance without any added benefits.
	//
	// Background: With every “commit” or “consolidate” operation, a new state of the View’s internal data structures
	// is created on disk. Old states/snapshots are released once there are no longer any users remaining.
	// However, the files for the released states/snapshots are left on disk, and only removed by “cleanup” operation.
	CleanupIntervalStep *int64 `json:"cleanupIntervalStep,omitempty"`

	// ConsolidationInterval specifies the minimum number of milliseconds that must be waited
	// between committing index data changes and making them visible to queries.
	// Defaults to 60000.
	// Use 0 to disable.
	// For the case where there are a lot of inserts/updates, a lower value,
	// until commit, will cause the index not to account for them and memory usage
	// would continue to grow.
	// For the case where there are a few inserts/updates, a higher value will
	// impact performance and waste disk space for each commit call without
	// any added benefits.
	ConsolidationInterval *int64 `json:"consolidationIntervalMsec,omitempty"`

	// ConsolidationPolicy specifies thresholds for consolidation.
	ConsolidationPolicy *ArangoSearchConsolidationPolicy `json:"consolidationPolicy,omitempty"`

	// CommitInterval ArangoSearch waits at least this many milliseconds between committing view data store changes and making documents visible to queries
	CommitInterval *int64 `json:"commitIntervalMsec,omitempty"`

	// WriteBufferIdle specifies the maximum number of writers (segments) cached in the pool.
	// 0 value turns off caching, default value is 64.
	WriteBufferIdle *int64 `json:"writebufferIdle,omitempty"`

	// WriteBufferActive specifies the maximum number of concurrent active writers (segments) performs (a transaction).
	// Other writers (segments) are wait till current active writers (segments) finish.
	// 0 value turns off this limit and used by default.
	WriteBufferActive *int64 `json:"writebufferActive,omitempty"`

	// WriteBufferSizeMax specifies maximum memory byte size per writer (segment) before a writer (segment) flush is triggered.
	// 0 value turns off this limit fon any writer (buffer) and will be flushed only after a period defined for special thread during ArangoDB server startup.
	// 0 value should be used with carefully due to high potential memory consumption.
	WriteBufferSizeMax *int64 `json:"writebufferSizeMax,omitempty"`

	// Links contains the properties for how individual collections
	// are indexed in the view.
	// The key of the map are collection names.
	Links ArangoSearchLinks `json:"links,omitempty"`

	// OptimizeTopK is an array of strings defining optimized sort expressions.
	// Introduced in v3.11.0, Enterprise Edition only.
	OptimizeTopK []string `json:"optimizeTopK,omitempty"`

	// PrimarySort describes how individual fields are sorted
	PrimarySort []ArangoSearchPrimarySortEntry `json:"primarySort,omitempty"`

	// PrimarySortCompression Defines how to compress the primary sort data (introduced in v3.7.1).
	// ArangoDB v3.5 and v3.6 always compress the index using LZ4. This option is immutable.
	PrimarySortCompression PrimarySortCompression `json:"primarySortCompression,omitempty"`

	// PrimarySortCache If you enable this option, then the primary sort columns are always cached in memory.
	// Can't be changed after creating View.
	// Introduced in v3.9.5, Enterprise Edition only
	PrimarySortCache *bool `json:"primarySortCache,omitempty"`

	// PrimaryKeyCache If you enable this option, then the primary key columns are always cached in memory.
	// Introduced in v3.9.6, Enterprise Edition only
	// Can't be changed after creating View.
	PrimaryKeyCache *bool `json:"primaryKeyCache,omitempty"`

	// StoredValues An array of objects to describe which document attributes to store in the View index (introduced in v3.7.1).
	// It can then cover search queries, which means the data can be taken from the index directly and accessing the storage engine can be avoided.
	// This option is immutable.
	StoredValues []StoredValue `json:"storedValues,omitempty"`
}

// ArangoSearchConsolidationPolicyType strings for consolidation types
type ArangoSearchConsolidationPolicyType string

const (
	// ArangoSearchConsolidationPolicyTypeTier consolidate based on segment byte size and live document count as dictated by the customization attributes.
	ArangoSearchConsolidationPolicyTypeTier ArangoSearchConsolidationPolicyType = "tier"

	// ArangoSearchConsolidationPolicyTypeBytesAccum consolidate if and only if ({threshold} range [0.0, 1.0])
	// {threshold} > (segment_bytes + sum_of_merge_candidate_segment_bytes) / all_segment_bytes,
	// i.e. the sum of all candidate segment's byte size is less than the total segment byte size multiplied by the {threshold}.
	ArangoSearchConsolidationPolicyTypeBytesAccum ArangoSearchConsolidationPolicyType = "bytes_accum"
)

// ArangoSearchConsolidationPolicy holds threshold values specifying when to consolidate view data.
// Semantics of the values depend on where they are used.
type ArangoSearchConsolidationPolicy struct {
	// Type returns the type of the ConsolidationPolicy. This interface can then be casted to the corresponding ArangoSearchConsolidationPolicy* struct.
	Type ArangoSearchConsolidationPolicyType `json:"type,omitempty"`

	ArangoSearchConsolidationPolicyBytesAccum
	ArangoSearchConsolidationPolicyTier
}

// ArangoSearchConsolidationPolicyBytesAccum contains fields used for ArangoSearchConsolidationPolicyTypeBytesAccum
type ArangoSearchConsolidationPolicyBytesAccum struct {
	// Threshold, see ArangoSearchConsolidationTypeBytesAccum
	Threshold *float64 `json:"threshold,omitempty"`
}

// ArangoSearchConsolidationPolicyTier contains fields used for ArangoSearchConsolidationPolicyTypeTier
type ArangoSearchConsolidationPolicyTier struct {
	MinScore *int64 `json:"minScore,omitempty"`

	// MinSegments specifies the minimum number of segments that will be evaluated as candidates for consolidation.
	MinSegments *int64 `json:"segmentsMin,omitempty"`

	// MaxSegments specifies the maximum number of segments that will be evaluated as candidates for consolidation.
	MaxSegments *int64 `json:"segmentsMax,omitempty"`

	// SegmentsBytesMax specifies the maxinum allowed size of all consolidated segments in bytes.
	SegmentsBytesMax *int64 `json:"segmentsBytesMax,omitempty"`

	// SegmentsBytesFloor defines the value (in bytes) to treat all smaller segments as equal for consolidation selection.
	SegmentsBytesFloor *int64 `json:"segmentsBytesFloor,omitempty"`
}

// ArangoSearchPrimarySortEntry describes an entry for the primarySort list
type ArangoSearchPrimarySortEntry struct {
	Field     string `json:"field,omitempty"`
	Ascending *bool  `json:"asc,omitempty"`
}

// GetAscending returns the value of Ascending or false if not set
func (pse ArangoSearchPrimarySortEntry) GetAscending() bool {
	if pse.Ascending != nil {
		return *pse.Ascending
	}

	return false
}

// ArangoSearchLinks is a strongly typed map containing links between a
// collection and a view.
// The keys in the map are collection names.
type ArangoSearchLinks map[string]ArangoSearchElementProperties

// ArangoSearchFields is a strongly typed map containing properties per field.
// The keys in the map are field names.
type ArangoSearchFields map[string]ArangoSearchElementProperties

// ArangoSearchElementProperties contains properties that specify how an element
// is indexed in an ArangoSearch view.
// Note that this structure is recursive. Settings not specified (nil)
// at a given level will inherit their setting from a lower level.
type ArangoSearchElementProperties struct {
	AnalyzerDefinitions []AnalyzerDefinition `json:"analyzerDefinitions,omitempty"`

	// The list of analyzers to be used for indexing of string values. Defaults to ["identify"].
	Analyzers []string `json:"analyzers,omitempty"`

	// If set to true, all fields of this element will be indexed. Defaults to false.
	IncludeAllFields *bool `json:"includeAllFields,omitempty"`

	// If set to true, values in a listed are treated as separate values. Defaults to false.
	TrackListPositions *bool `json:"trackListPositions,omitempty"`

	// This values specifies how the view should track values.
	StoreValues ArangoSearchStoreValues `json:"storeValues,omitempty"`

	// Fields contains the properties for individual fields of the element.
	// The key of the map are field names.
	Fields ArangoSearchFields `json:"fields,omitempty"`

	// If set to true, then no exclusive lock is used on the source collection during View index creation,
	// so that it remains basically available. inBackground is an option that can be set when adding links.
	// It does not get persisted as it is not a View property, but only a one-off option
	InBackground *bool `json:"inBackground,omitempty"`

	// Nested contains the properties for nested fields (sub-objects) of the element
	// Enterprise Edition only
	Nested ArangoSearchFields `json:"nested,omitempty"`

	// Cache If you enable this option, then field normalization values are always cached in memory.
	// Introduced in v3.9.5, Enterprise Edition only
	Cache *bool `json:"cache,omitempty"`
}

// ArangoSearchStoreValues is the type of the StoreValues option of an ArangoSearch element.
type ArangoSearchStoreValues string

const (
	// ArangoSearchStoreValuesNone specifies that a view should not store values.
	ArangoSearchStoreValuesNone ArangoSearchStoreValues = "none"

	// ArangoSearchStoreValuesID specifies that a view should only store
	// information about value presence, to allow use of the EXISTS() function.
	ArangoSearchStoreValuesID ArangoSearchStoreValues = "id"
)
