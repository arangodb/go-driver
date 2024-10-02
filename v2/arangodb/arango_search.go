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

// ArangoSearchFeature specifies a feature to an analyzer
type ArangoSearchFeature string

const (
	// ArangoSearchFeatureFrequency how often a term is seen, required for PHRASE()
	ArangoSearchFeatureFrequency ArangoSearchFeature = "frequency"

	// ArangoSearchFeatureNorm the field normalization factor
	ArangoSearchFeatureNorm ArangoSearchFeature = "norm"

	// ArangoSearchFeaturePosition sequentially increasing term position, required for PHRASE(). If present then the frequency feature is also required
	ArangoSearchFeaturePosition ArangoSearchFeature = "position"

	// ArangoSearchFeatureOffset can be specified if 'position' feature is set
	ArangoSearchFeatureOffset ArangoSearchFeature = "offset"
)

// ArangoSearchAnalyzerType specifies type of analyzer
type ArangoSearchAnalyzerType string

const (
	// ArangoSearchAnalyzerTypeIdentity treat value as atom (no transformation)
	ArangoSearchAnalyzerTypeIdentity ArangoSearchAnalyzerType = "identity"

	// ArangoSearchAnalyzerTypeDelimiter split into tokens at user-defined character
	ArangoSearchAnalyzerTypeDelimiter ArangoSearchAnalyzerType = "delimiter"

	// ArangoSearchAnalyzerTypeMultiDelimiter split into tokens at user-defined character
	//
	// Available in ArangoDB 3.12 and later.
	ArangoSearchAnalyzerTypeMultiDelimiter ArangoSearchAnalyzerType = "multi_delimiter"

	// ArangoSearchAnalyzerTypeStem apply stemming to the value as a whole
	ArangoSearchAnalyzerTypeStem ArangoSearchAnalyzerType = "stem"

	// ArangoSearchAnalyzerTypeNorm apply normalization to the value as a whole
	ArangoSearchAnalyzerTypeNorm ArangoSearchAnalyzerType = "norm"

	// ArangoSearchAnalyzerTypeNGram create n-grams from value with user-defined lengths
	ArangoSearchAnalyzerTypeNGram ArangoSearchAnalyzerType = "ngram"

	// ArangoSearchAnalyzerTypeText tokenize into words, optionally with stemming, normalization and stop-word filtering
	ArangoSearchAnalyzerTypeText ArangoSearchAnalyzerType = "text"

	// ArangoSearchAnalyzerTypeAQL an Analyzer capable of running a restricted AQL query to perform data manipulation / filtering.
	ArangoSearchAnalyzerTypeAQL ArangoSearchAnalyzerType = "aql"

	// ArangoSearchAnalyzerTypePipeline an Analyzer capable of chaining effects of multiple Analyzers into one. The pipeline is a list of Analyzers, where the output of an Analyzer is passed to the next for further processing. The final token value is determined by last Analyzer in the pipeline.
	ArangoSearchAnalyzerTypePipeline ArangoSearchAnalyzerType = "pipeline"

	// ArangoSearchAnalyzerTypeStopwords an Analyzer capable of removing specified tokens from the input.
	ArangoSearchAnalyzerTypeStopwords ArangoSearchAnalyzerType = "stopwords"

	// ArangoSearchAnalyzerTypeGeoJSON an Analyzer capable of breaking up a GeoJSON object into a set of indexable tokens for further usage with ArangoSearch Geo functions.
	ArangoSearchAnalyzerTypeGeoJSON ArangoSearchAnalyzerType = "geojson"

	// ArangoSearchAnalyzerTypeGeoS2 an Analyzer capable of index GeoJSON data with inverted indexes or Views similar
	// to the existing `geojson` Analyzer, but it internally uses a format for storing the geo-spatial data.
	// that is more efficient.
	ArangoSearchAnalyzerTypeGeoS2 ArangoSearchAnalyzerType = "geo_s2"

	// ArangoSearchAnalyzerTypeGeoPoint an Analyzer capable of breaking up JSON object describing a coordinate into a set of indexable tokens for further usage with ArangoSearch Geo functions.
	ArangoSearchAnalyzerTypeGeoPoint ArangoSearchAnalyzerType = "geopoint"

	// ArangoSearchAnalyzerTypeSegmentation an Analyzer capable of breaking up the input text into tokens in a language-agnostic manner
	ArangoSearchAnalyzerTypeSegmentation ArangoSearchAnalyzerType = "segmentation"

	// ArangoSearchAnalyzerTypeCollation an Analyzer capable of converting the input into a set of language-specific tokens
	ArangoSearchAnalyzerTypeCollation ArangoSearchAnalyzerType = "collation"

	// ArangoSearchAnalyzerTypeClassification An Analyzer capable of classifying tokens in the input text. (EE only)
	ArangoSearchAnalyzerTypeClassification ArangoSearchAnalyzerType = "classification"

	// ArangoSearchAnalyzerTypeNearestNeighbors An Analyzer capable of finding nearest neighbors of tokens in the input. (EE only)
	ArangoSearchAnalyzerTypeNearestNeighbors ArangoSearchAnalyzerType = "nearest_neighbors"

	// ArangoSearchAnalyzerTypeMinhash an analyzer which is capable of evaluating so called MinHash signatures as a stream of tokens. (EE only)
	ArangoSearchAnalyzerTypeMinhash ArangoSearchAnalyzerType = "minhash"

	// ArangoSearchAnalyzerTypeWildcard An Analyzer that creates n-grams to enable fast partial matching for wildcard
	// queries if you have large string values, especially if you want to search for suffixes or substrings in the
	// middle of strings (infixes) as opposed to prefixes.
	//
	// Available in ArangoDB 3.12 and later.
	ArangoSearchAnalyzerTypeWildcard ArangoSearchAnalyzerType = "wildcard"
)

// ArangoSearchAnalyzerProperties specifies options for the analyzer.
// Required and respected depend on the analyzer type.
// See docs: https://docs.arangodb.com/stable/index-and-search/analyzers/#analyzer-properties
type ArangoSearchAnalyzerProperties struct {
	IsSystem bool `json:"isSystem,omitempty"`

	// Locale used by ArangoSearchAnalyzerTypeStem, ArangoSearchAnalyzerTypeNorm, Text
	Locale string `json:"locale,omitempty"`

	// Delimiter used by ArangoSearchAnalyzerTypeDelimiter
	Delimiter string `json:"delimiter,omitempty"`

	// Delimiters used by ArangoSearchAnalyzerTypeMultiDelimiter
	Delimiters []string `json:"delimiters,omitempty"`

	// Accent used by ArangoSearchAnalyzerTypeNorm, ArangoSearchAnalyzerTypeText
	Accent *bool `json:"accent,omitempty"`

	// Case used by ArangoSearchAnalyzerTypeNorm, ArangoSearchAnalyzerTypeText, ArangoSearchAnalyzerTypeSegmentation
	Case ArangoSearchCaseType `json:"case,omitempty"`

	// EdgeNGram used by ArangoSearchAnalyzerTypeText
	EdgeNGram *ArangoSearchEdgeNGram `json:"edgeNgram,omitempty"`

	// Min used by ArangoSearchAnalyzerTypeNGram
	Min *int64 `json:"min,omitempty"`

	// Max used by ArangoSearchAnalyzerTypeNGram

	Max *int64 `json:"max,omitempty"`
	// PreserveOriginal used by ArangoSearchAnalyzerTypeNGram
	PreserveOriginal *bool `json:"preserveOriginal,omitempty"`

	// StartMarker used by ArangoSearchAnalyzerTypeNGram
	StartMarker *string `json:"startMarker,omitempty"`

	// EndMarker used by ArangoSearchAnalyzerTypeNGram
	EndMarker *string `json:"endMarker,omitempty"`

	// StreamType used by ArangoSearchAnalyzerTypeNGram
	StreamType *ArangoSearchNGramStreamType `json:"streamType,omitempty"`

	// Stemming used by ArangoSearchAnalyzerTypeText
	Stemming *bool `json:"stemming,omitempty"`

	// Stopwords used by ArangoSearchAnalyzerTypeText and ArangoSearchAnalyzerTypeStopwords.
	// This field is not mandatory since version 3.7 of arangod so it can not be omitted in 3.6.
	Stopwords []string `json:"stopwords"`

	// StopwordsPath used by ArangoSearchAnalyzerTypeText
	StopwordsPath []string `json:"stopwordsPath,omitempty"`

	// QueryString used by ArangoSearchAnalyzerTypeAQL.
	QueryString string `json:"queryString,omitempty"`

	// CollapsePositions used by ArangoSearchAnalyzerTypeAQL.
	CollapsePositions *bool `json:"collapsePositions,omitempty"`

	// KeepNull used by ArangoSearchAnalyzerTypeAQL.
	KeepNull *bool `json:"keepNull,omitempty"`

	// BatchSize used by ArangoSearchAnalyzerTypeAQL.
	BatchSize *int `json:"batchSize,omitempty"`

	// MemoryLimit used by ArangoSearchAnalyzerTypeAQL.
	MemoryLimit *int `json:"memoryLimit,omitempty"`

	// ReturnType used by ArangoSearchAnalyzerTypeAQL.
	ReturnType *ArangoSearchAnalyzerAQLReturnType `json:"returnType,omitempty"`

	// Pipeline used by ArangoSearchAnalyzerTypePipeline.
	Pipeline []ArangoSearchAnalyzerPipeline `json:"pipeline,omitempty"`

	// Type used by ArangoSearchAnalyzerTypeGeoJSON.
	Type *ArangoSearchAnalyzerGeoJSONType `json:"type,omitempty"`

	// Options used by ArangoSearchAnalyzerTypeGeoJSON and ArangoSearchAnalyzerTypeGeoPoint
	Options *ArangoSearchAnalyzerGeoOptions `json:"options,omitempty"`

	// Latitude used by ArangoSearchAnalyzerTypeGeoPoint.
	Latitude []string `json:"latitude,omitempty"`

	// Longitude used by ArangoSearchAnalyzerTypeGeoPoint.
	Longitude []string `json:"longitude,omitempty"`

	// Break used by ArangoSearchAnalyzerTypeSegmentation
	Break ArangoSearchBreakType `json:"break,omitempty"`

	// Hex used by ArangoSearchAnalyzerTypeStopwords.
	// If false then each string in stopwords is used verbatim.
	// If true, then each string in stopwords needs to be hex-encoded.
	Hex *bool `json:"hex,omitempty"`

	// ModelLocation used by ArangoSearchAnalyzerTypeClassification, ArangoSearchAnalyzerTypeNearestNeighbors
	// The on-disk path to the trained fastText supervised model.
	// Note: if you are running this in an ArangoDB cluster, this model must exist on every machine in the cluster.
	ModelLocation string `json:"model_location,omitempty"`

	// TopK  used by ArangoSearchAnalyzerTypeClassification, ArangoSearchAnalyzerTypeNearestNeighbors
	// The number of class labels that will be produced per input (default: 1)
	TopK *uint64 `json:"top_k,omitempty"`

	// Threshold  used by ArangoSearchAnalyzerTypeClassification
	// The probability threshold for which a label will be assigned to an input.
	// A fastText model produces a probability per class label, and this is what will be filtered (default: 0.99).
	Threshold *float64 `json:"threshold,omitempty"`

	// Analyzer used by ArangoSearchAnalyzerTypeMinhash
	// Definition of inner analyzer to use for incoming data. In case if omitted field or empty object falls back to 'identity' analyzer.
	Analyzer *AnalyzerDefinition `json:"analyzer,omitempty"`

	// NumHashes used by ArangoSearchAnalyzerTypeMinhash
	// Size of min hash signature. Must be greater or equal to 1.
	NumHashes *uint64 `json:"numHashes,omitempty"`

	// Format is the internal binary representation to use for storing the geo-spatial data in an index.
	Format *ArangoSearchFormat `json:"format,omitempty"`

	// NGramSize used by ArangoSearchAnalyzerTypeWildcard
	// It is an unsigned integer for the n-gram length, needs to be at least 2.
	// It can be greater than the substrings between wildcards that you want to search for, e.g. 4 with an expected
	// search pattern of %up%if%ref% (substrings of length 2 and 3 between %), but this leads to a slower search
	// (for ref% with post-validation using the ICU regular expression engine).
	// A value of 3 is a good default, 2 is better for short strings
	NGramSize uint `json:"ngramSize"`
}

type ArangoSearchCaseType string

const (
	// ArangoSearchCaseUpper to convert to all lower-case characters
	ArangoSearchCaseUpper ArangoSearchCaseType = "upper"

	// ArangoSearchCaseLower to convert to all upper-case characters
	ArangoSearchCaseLower ArangoSearchCaseType = "lower"

	// ArangoSearchCaseNone to not change character case (default)
	ArangoSearchCaseNone ArangoSearchCaseType = "none"
)

// ArangoSearchEdgeNGram specifies options for the edgeNGram text analyzer.
// More information can be found here: https://docs.arangodb.com/stable/index-and-search/analyzers/#text
type ArangoSearchEdgeNGram struct {
	// Min used by Text
	Min *int64 `json:"min,omitempty"`

	// Max used by Text
	Max *int64 `json:"max,omitempty"`

	// PreserveOriginal used by Text
	PreserveOriginal *bool `json:"preserveOriginal,omitempty"`
}

type ArangoSearchFormat string

const (
	// ArangoSearchFormatLatLngDouble stores each latitude and longitude value as an 8-byte floating-point value (16 bytes per coordinate pair).
	// It is default value.
	ArangoSearchFormatLatLngDouble ArangoSearchFormat = "latLngDouble"

	// ArangoSearchFormatLatLngInt stores each latitude and longitude value as an 4-byte integer value (8 bytes per coordinate pair).
	// This is the most compact format but the precision is limited to approximately 1 to 10 centimeters.

	ArangoSearchFormatLatLngInt ArangoSearchFormat = "latLngInt"
	// ArangoSearchFormatS2Point store each longitude-latitude pair in the native format of Google S2 which is used for geo-spatial
	// calculations (24 bytes per coordinate pair).
	ArangoSearchFormatS2Point ArangoSearchFormat = "s2Point"
)

type ArangoSearchNGramStreamType string

const (
	// ArangoSearchNGramStreamBinary used by NGram. Default value
	ArangoSearchNGramStreamBinary ArangoSearchNGramStreamType = "binary"

	// ArangoSearchNGramStreamUTF8 used by NGram
	ArangoSearchNGramStreamUTF8 ArangoSearchNGramStreamType = "utf8"
)

type ArangoSearchAnalyzerAQLReturnType string

const (
	ArangoSearchAnalyzerAQLReturnTypeString ArangoSearchAnalyzerAQLReturnType = "string"
	ArangoSearchAnalyzerAQLReturnTypeNumber ArangoSearchAnalyzerAQLReturnType = "number"
	ArangoSearchAnalyzerAQLReturnTypeBool   ArangoSearchAnalyzerAQLReturnType = "bool"
)

// New returns pointer to selected return type
func (a ArangoSearchAnalyzerAQLReturnType) New() *ArangoSearchAnalyzerAQLReturnType {
	return &a
}

// ArangoSearchAnalyzerPipeline provides object definition for Pipeline array parameter
type ArangoSearchAnalyzerPipeline struct {
	// Type of the Pipeline Analyzer
	Type ArangoSearchAnalyzerType `json:"type"`

	// Properties of the Pipeline Analyzer
	Properties ArangoSearchAnalyzerProperties `json:"properties,omitempty"`
}

// ArangoSearchAnalyzerGeoJSONType GeoJSON Type parameter.
type ArangoSearchAnalyzerGeoJSONType string

// New returns pointer to selected return type
func (a ArangoSearchAnalyzerGeoJSONType) New() *ArangoSearchAnalyzerGeoJSONType {
	return &a
}

const (
	// ArangoSearchAnalyzerGeoJSONTypeShape define index all GeoJSON geometry types (Point, Polygon etc.). (default)
	ArangoSearchAnalyzerGeoJSONTypeShape ArangoSearchAnalyzerGeoJSONType = "shape"

	// ArangoSearchAnalyzerGeoJSONTypeCentroid define compute and only index the centroid of the input geometry.
	ArangoSearchAnalyzerGeoJSONTypeCentroid ArangoSearchAnalyzerGeoJSONType = "centroid"

	// ArangoSearchAnalyzerGeoJSONTypePoint define only index GeoJSON objects of type Point, ignore all other geometry types.
	ArangoSearchAnalyzerGeoJSONTypePoint ArangoSearchAnalyzerGeoJSONType = "point"
)

// ArangoSearchAnalyzerGeoOptions for fine-tuning geo queries. These options should generally remain unchanged.
type ArangoSearchAnalyzerGeoOptions struct {
	// MaxCells define maximum number of S2 cells.
	MaxCells *int `json:"maxCells,omitempty"`

	// MinLevel define the least precise S2 level.
	MinLevel *int `json:"minLevel,omitempty"`

	// MaxLevel define the most precise S2 level
	MaxLevel *int `json:"maxLevel,omitempty"`
}

type ArangoSearchBreakType string

const (
	// ArangoSearchBreakTypeAll to return all tokens
	ArangoSearchBreakTypeAll ArangoSearchBreakType = "all"

	// ArangoSearchBreakTypeAlpha to return tokens composed of alphanumeric characters only (default)
	ArangoSearchBreakTypeAlpha ArangoSearchBreakType = "alpha"

	// ArangoSearchBreakTypeGraphic to return tokens composed of non-whitespace characters only
	ArangoSearchBreakTypeGraphic ArangoSearchBreakType = "graphic"
)
