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

// InvertedIndexOptions provides specific options for creating an inverted index
type InvertedIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name,omitempty"`

	// Fields contains the properties for individual fields of the element.
	// The key of the map are field names.
	// Required: true
	Fields []InvertedIndexField `json:"fields,omitempty"`

	// SearchField This option only applies if you use the inverted index in a search-alias Views.
	// You can set the option to true to get the same behavior as with arangosearch Views regarding the indexing of array values as the default.
	// If enabled, both, array and primitive values (strings, numbers, etc.) are accepted. Every element of an array is indexed according to the trackListPositions option.
	// If set to false, it depends on the attribute path. If it explicitly expand an array ([*]), then the elements are indexed separately.
	// Otherwise, the array is indexed as a whole, but only geopoint and aql Analyzers accept array inputs.
	// You cannot use an array expansion if searchField is enabled.
	SearchField *bool `json:"searchField,omitempty"`

	// Cache - Enable this option to always cache the field normalization values in memory for all fields by default.
	Cache *bool `json:"cache,omitempty"`

	// StoredValues The optional storedValues attribute can contain an array of paths to additional attributes to store in the index.
	// These additional attributes cannot be used for index lookups or for sorting, but they can be used for projections.
	// This allows an index to fully cover more queries and avoid extra document lookups.
	StoredValues []StoredValue `json:"storedValues,omitempty"`

	// PrimarySort You can define a primary sort order to enable an AQL optimization.
	// If a query iterates over all documents of a collection, wants to sort them by attribute values, and the (left-most) fields to sort by,
	// as well as their sorting direction, match with the primarySort definition, then the SORT operation is optimized away.
	PrimarySort *PrimarySort `json:"primarySort,omitempty"`

	// PrimaryKeyCache Enable this option to always cache the primary key column in memory.
	// This can improve the performance of queries that return many documents.
	PrimaryKeyCache *bool `json:"primaryKeyCache,omitempty"`

	// Analyzer  The name of an Analyzer to use by default. This Analyzer is applied to the values of the indexed
	// fields for which you don’t define Analyzers explicitly.
	Analyzer string `json:"analyzer,omitempty"`

	// Features list of analyzer features. You can set this option to overwrite what features are enabled for the default analyzer
	Features []ArangoSearchFeature `json:"features,omitempty"`

	// IncludeAllFields If set to true, all fields of this element will be indexed. Defaults to false.
	// Warning: Using includeAllFields for a lot of attributes in combination with complex Analyzers
	// may significantly slow down the indexing process.
	IncludeAllFields *bool `json:"includeAllFields,omitempty"`

	// TrackListPositions track the value position in arrays for array values.
	TrackListPositions bool `json:"trackListPositions,omitempty"`

	// Parallelism - The number of threads to use for indexing the fields. Default: 2
	Parallelism *int `json:"parallelism,omitempty"`

	// CleanupIntervalStep Wait at least this many commits between removing unused files in the ArangoSearch data directory
	// (default: 2, to disable use: 0).
	CleanupIntervalStep *int64 `json:"cleanupIntervalStep,omitempty"`

	// CommitIntervalMsec Wait at least this many milliseconds between committing View data store changes and making
	// documents visible to queries (default: 1000, to disable use: 0).
	CommitIntervalMsec *int64 `json:"commitIntervalMsec,omitempty"`

	// ConsolidationIntervalMsec Wait at least this many milliseconds between applying ‘consolidationPolicy’ to consolidate View data store
	// and possibly release space on the filesystem (default: 1000, to disable use: 0).
	ConsolidationIntervalMsec *int64 `json:"consolidationIntervalMsec,omitempty"`

	// ConsolidationPolicy The consolidation policy to apply for selecting which segments should be merged (default: {}).
	ConsolidationPolicy *ConsolidationPolicy `json:"consolidationPolicy,omitempty"`

	// WriteBufferIdle Maximum number of writers (segments) cached in the pool (default: 64, use 0 to disable)
	WriteBufferIdle *int64 `json:"writebufferIdle,omitempty"`

	// WriteBufferActive Maximum number of concurrent active writers (segments) that perform a transaction.
	// Other writers (segments) wait till current active writers (segments) finish (default: 0, use 0 to disable)
	WriteBufferActive *int64 `json:"writebufferActive,omitempty"`

	// WriteBufferSizeMax Maximum memory byte size per writer (segment) before a writer (segment) flush is triggered.
	// 0 value turns off this limit for any writer (buffer) and data will be flushed periodically based on the value defined for the flush thread (ArangoDB server startup option).
	// 0 value should be used carefully due to high potential memory consumption (default: 33554432, use 0 to disable)
	WriteBufferSizeMax *int64 `json:"writebufferSizeMax,omitempty"`

	// OptimizeTopK is an array of strings defining optimized sort expressions.
	// Introduced in v3.11.0, Enterprise Edition only.
	OptimizeTopK []string `json:"optimizeTopK,omitempty"`

	// InBackground You can set this option to true to create the index in the background,
	// which will not write-lock the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground *bool `json:"inBackground,omitempty"`
}

// InvertedIndexField contains configuration for indexing of the field
type InvertedIndexField struct {
	// Name (Required) An attribute path. The '.' character denotes sub-attributes.
	Name string `json:"name"`

	// Analyzer indicating the name of an analyzer instance
	// Default: the value defined by the top-level analyzer option, or if not set, the default identity Analyzer.
	Analyzer string `json:"analyzer,omitempty"`

	// Features is a list of Analyzer features to use for this field. They define what features are enabled for the analyzer
	Features []ArangoSearchFeature `json:"features,omitempty"`

	// IncludeAllFields This option only applies if you use the inverted index in a search-alias Views.
	// If set to true, then all sub-attributes of this field are indexed, excluding any sub-attributes that are configured separately by other elements in the fields array (and their sub-attributes). The analyzer and features properties apply to the sub-attributes.
	// If set to false, then sub-attributes are ignored. The default value is defined by the top-level includeAllFields option, or false if not set.
	IncludeAllFields *bool `json:"includeAllFields,omitempty"`

	// SearchField This option only applies if you use the inverted index in a search-alias Views.
	// You can set the option to true to get the same behavior as with arangosearch Views regarding the indexing of array values for this field. If enabled, both, array and primitive values (strings, numbers, etc.) are accepted. Every element of an array is indexed according to the trackListPositions option.
	// If set to false, it depends on the attribute path. If it explicitly expand an array ([*]), then the elements are indexed separately. Otherwise, the array is indexed as a whole, but only geopoint and aql Analyzers accept array inputs. You cannot use an array expansion if searchField is enabled.
	// Default: the value defined by the top-level searchField option, or false if not set.
	SearchField *bool `json:"searchField,omitempty"`

	// TrackListPositions This option only applies if you use the inverted index in a search-alias Views.
	// If set to true, then track the value position in arrays for array values. For example, when querying a document like { attr: [ "valueX", "valueY", "valueZ" ] }, you need to specify the array element, e.g. doc.attr[1] == "valueY".
	// If set to false, all values in an array are treated as equal alternatives. You don’t specify an array element in queries, e.g. doc.attr == "valueY", and all elements are searched for a match.
	// Default: the value defined by the top-level trackListPositions option, or false if not set.
	TrackListPositions bool `json:"trackListPositions,omitempty"`

	// Cache - Enable this option to always cache the field normalization values in memory for this specific field
	// Default: the value defined by the top-level 'cache' option.
	Cache *bool `json:"cache,omitempty"`

	// Nested Index the specified sub-objects that are stored in an array.
	// Other than with the fields property, the values get indexed in a way that lets you query for co-occurring values.
	// For example, you can search the sub-objects and all the conditions need to be met by a single sub-object instead of across all of them.
	// Enterprise-only feature
	Nested []InvertedIndexNestedField `json:"nested,omitempty"`
}

// InvertedIndexNestedField contains sub-object configuration for indexing of the field
type InvertedIndexNestedField struct {
	// Name An attribute path. The . character denotes sub-attributes.
	Name string `json:"name"`

	// Analyzer indicating the name of an analyzer instance
	// Default: the value defined by the top-level analyzer option, or if not set, the default identity Analyzer.
	Analyzer string `json:"analyzer,omitempty"`

	// Features is a list of Analyzer features to use for this field. They define what features are enabled for the analyzer
	Features []ArangoSearchFeature `json:"features,omitempty"`

	// SearchField This option only applies if you use the inverted index in a search-alias Views.
	// You can set the option to true to get the same behavior as with arangosearch Views regarding the indexing of array values for this field. If enabled, both, array and primitive values (strings, numbers, etc.) are accepted. Every element of an array is indexed according to the trackListPositions option.
	// If set to false, it depends on the attribute path. If it explicitly expand an array ([*]), then the elements are indexed separately. Otherwise, the array is indexed as a whole, but only geopoint and aql Analyzers accept array inputs. You cannot use an array expansion if searchField is enabled.
	// Default: the value defined by the top-level searchField option, or false if not set.
	SearchField *bool `json:"searchField,omitempty"`

	// Cache - Enable this option to always cache the field normalization values in memory for this specific field
	// Default: the value defined by the top-level 'cache' option.
	Cache *bool `json:"cache,omitempty"`

	// Nested - Index the specified sub-objects that are stored in an array.
	// Other than with the fields property, the values get indexed in a way that lets you query for co-occurring values.
	// For example, you can search the sub-objects and all the conditions need to be met by a single sub-object instead of across all of them.
	// Enterprise-only feature
	Nested []InvertedIndexNestedField `json:"nested,omitempty"`
}
