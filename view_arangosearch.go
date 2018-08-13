//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

import (
	"context"
)

// ArangoSearchView provides access to the information of a view.
// Views are only available in ArangoDB 3.4 and higher.
type ArangoSearchView interface {
	// Include generic View functions
	View

	// Properties fetches extended information about the view.
	Properties(ctx context.Context) (ArangoSearchViewProperties, error)

	// SetProperties changes properties of the view.
	SetProperties(ctx context.Context, options ArangoSearchViewProperties) error
}

// ArangoSearchViewProperties contains properties an an ArangoSearch view.
type ArangoSearchViewProperties struct {
	// Locale specifies the default locale used for queries on analyzed string values.
	// Defaults to "C". TODO What is that?
	Locale Locale `json:"locale,omitempty"`
	// Commit behavior related properties
	Commit *ArangoSearchCommitProperties `json:"commit,omitempty"`
	// Links contains the properties for how individual collections
	// are indexed in thie view.
	// The key of the map are collection names.
	Links ArangoSearchLinks `json:"links,omitempty"`
}

// ArangoSearchCommitProperties contains properties related to the commit
// behavior of an ArangoSearch view.
type ArangoSearchCommitProperties struct {
	// Consolidate specifies boundaries for various consolitation policies.
	Consolidate *ArangoSearchConsolidationProperties `json:"consolidate,omitempty"`
	// CommitInterval specifies the minimum number of milliseconds that must be waited
	// between committing index data changes and making them visible to queries.
	// Defaults to 60000.
	// Use 0 to disable.
	// For the case where there are a lot of inserts/updates, a lower value,
	// until commit, will cause the index not to account for them and memory usage
	// would continue to grow.
	// For the case where there are a few inserts/updates, a higher value will
	// impact performance and waste disk space for each commit call without
	// any added benefits.
	CommitInterval int64 `json:"commitIntervalMsec,omitempty"`
	// CleanupIntervalStep specifies the minimum number of commits to wait between
	// removing unused files in the data directory.
	// Defaults to 10.
	// Use 0 to disable waiting.
	// For the case where the consolidation policies merge segments often
	// (i.e. a lot of commit+consolidate), a lower value will cause a lot of
	// disk space to be wasted.
	// For the case where the consolidation policies rarely merge segments
	// (i.e. few inserts/deletes), a higher value will impact performance
	// without any added benefits.
	CleanupIntervalStep int64 `json:"cleanupIntervalStep,omitempty"`
}

// Locale is a strongly typed specifier of a locale.
// TODO specify semantics.
type Locale string

// ArangoSearchConsolidationProperties holds values specifying when to
// consolidate view data.
type ArangoSearchConsolidationProperties struct {
	// Count specifies consilidation boundaries based on the number of documents.
	Count *ArangoSearchConsolidationThreshold `json:"count,omitempty"`
	// Bytes specifies consilidation boundaries based on the size of the data.
	Bytes *ArangoSearchConsolidationThreshold `json:"bytes,omitempty"`
	// BytesAccumulated specifies consilidation boundaries based on the size of the data. ???? TODO
	BytesAccumulated *ArangoSearchConsolidationThreshold `json:"bytes_accum,omitempty"`
	// Fill specifies consilidation boundaries based on ????? TODO
	Fill *ArangoSearchConsolidationThreshold `json:"fill,omitempty"`
}

// ArangoSearchConsolidationThreshold holds threshold values specifying when to
// consolidate view data.
// Semantics of the values depend on where they are used.
type ArangoSearchConsolidationThreshold struct {
	// Threshold is a percentage (0..1)
	Threshold float64 `json:"threshold,omitempty"`
	// SegmentThreshold is an absolute value.
	SegmentThreshold int64 `json:"segmentThreshold,omitempty"`
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
