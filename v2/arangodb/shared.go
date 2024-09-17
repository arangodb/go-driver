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
//

package arangodb

const (
	HeaderDirtyReads  = "x-arango-allow-dirty-read"
	HeaderTransaction = "x-arango-trx-id"
	HeaderIfMatch     = "If-Match"
	HeaderIfNoneMatch = "If-None-Match"

	QueryRev               = "rev"
	QueryIgnoreRevs        = "ignoreRevs"
	QueryWaitForSync       = "waitForSync"
	QueryReturnNew         = "returnNew"
	QueryReturnOld         = "returnOld"
	QueryKeepNull          = "keepNull"
	QueryDirection         = "direction"
	QuerySilent            = "silent"
	QueryRefillIndexCaches = "refillIndexCaches"
	QueryMergeObjects      = "mergeObjects"
	QueryOverwrite         = "overwrite"
	QueryOverwriteMode     = "overwriteMode"
	QueryVersionAttribute  = "versionAttribute"
	QueryIsRestore         = "isRestore"
)

// PrimarySortCompression Defines how to compress the primary sort data (introduced in v3.7.1)
type PrimarySortCompression string

const (
	// PrimarySortCompressionLz4 (default): use LZ4 fast compression.
	PrimarySortCompressionLz4 PrimarySortCompression = "lz4"

	// PrimarySortCompressionNone disable compression to trade space for speed.
	PrimarySortCompressionNone PrimarySortCompression = "none"
)

// SortDirection describes the sorting direction
type SortDirection string

const (
	// SortDirectionAsc sort ascending
	SortDirectionAsc SortDirection = "asc"

	// SortDirectionDesc sort descending
	SortDirectionDesc SortDirection = "desc"
)

// PrimarySort defines compression and list of fields to be sorted
type PrimarySort struct {
	// Fields (Required) - An array of the fields to sort the index by and the direction to sort each field in.
	Fields []PrimarySortEntry `json:"fields,omitempty"`

	// Compression Defines how to compress the primary sort data
	Compression PrimarySortCompression `json:"compression,omitempty"`

	// Cache - Enable this option to always cache the primary sort columns in memory.
	// This can improve the performance of queries that utilize the primary sort order.
	Cache *bool `json:"cache,omitempty"`
}

// PrimarySortEntry field to sort the index by and the direction
type PrimarySortEntry struct {
	// Field An attribute path. The . character denotes sub-attributes.
	Field string `json:"field,required"`

	// Ascending The sorting direction
	Ascending bool `json:"asc,required"`
}

// StoredValue defines the value stored in the index
type StoredValue struct {
	// Fields A list of attribute paths. The . character denotes sub-attributes.
	Fields []string `json:"fields,omitempty"`

	// Compression Defines how to compress the attribute values.
	Compression PrimarySortCompression `json:"compression,omitempty"`

	// Cache attribute allows you to always cache stored values in memory
	// Introduced in v3.9.5, Enterprise Edition only
	Cache *bool `json:"cache,omitempty"`
}

// ConsolidationPolicyType strings for consolidation types
type ConsolidationPolicyType string

const (
	// ConsolidationPolicyTypeTier consolidate based on segment byte size and live document count as dictated by the customization attributes.
	ConsolidationPolicyTypeTier ConsolidationPolicyType = "tier"

	// ConsolidationPolicyTypeBytesAccum consolidate if and only if ({threshold} range [0.0, 1.0])
	// {threshold} > (segment_bytes + sum_of_merge_candidate_segment_bytes) / all_segment_bytes,
	// i.e. the sum of all candidate segment's byte size is less than the total segment byte size multiplied by the {threshold}.
	ConsolidationPolicyTypeBytesAccum ConsolidationPolicyType = "bytes_accum"
)

// ConsolidationPolicy holds threshold values specifying when to consolidate view data.
// Semantics of the values depend on where they are used.
type ConsolidationPolicy struct {
	// Type returns the type of the ConsolidationPolicy. This interface can then be casted to the corresponding ConsolidationPolicy struct.
	Type ConsolidationPolicyType `json:"type,omitempty"`

	ConsolidationPolicyBytesAccum
	ConsolidationPolicyTier
}

// ConsolidationPolicyBytesAccum contains fields used for ConsolidationPolicyTypeBytesAccum
type ConsolidationPolicyBytesAccum struct {
	// Threshold, see ConsolidationTypeBytesAccum
	Threshold *float64 `json:"threshold,omitempty"`
}

// ConsolidationPolicyTier contains fields used for ConsolidationPolicyTypeTier
type ConsolidationPolicyTier struct {
	// MinScore Filter out consolidation candidates with a score less than this. Default: 0
	MinScore *int64 `json:"minScore,omitempty"`

	// SegmentsMin The minimum number of segments that are evaluated as candidates for consolidation. Default: 1
	SegmentsMin *int64 `json:"segmentsMin,omitempty"`

	// SegmentsMax The maximum number of segments that are evaluated as candidates for consolidation. Default: 10
	SegmentsMax *int64 `json:"segmentsMax,omitempty"`

	// SegmentsBytesMax  The maximum allowed size of all consolidated segments in bytes. Default: 5368709120
	SegmentsBytesMax *int64 `json:"segmentsBytesMax,omitempty"`

	// SegmentsBytesFloor Defines the value (in bytes) to treat all smaller segments as equal for consolidation selection. Default: 2097152
	SegmentsBytesFloor *int64 `json:"segmentsBytesFloor,omitempty"`
}
