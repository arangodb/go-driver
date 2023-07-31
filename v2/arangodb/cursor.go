//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"io"
)

// Cursor is returned from a query, used to iterate over a list of documents.
// Note that a Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
type Cursor interface {
	io.Closer

	// CloseWithContext run Close with specified Context
	CloseWithContext(ctx context.Context) error

	// HasMore returns true if the next call to ReadDocument does not return a NoMoreDocuments error.
	HasMore() bool

	// ReadDocument reads the next document from the cursor.
	// The document data is stored into result, the document meta data is returned.
	// If the cursor has no more documents, a NoMoreDocuments error is returned.
	// Note: If the query (resulting in this cursor) does not return documents,
	//       then the returned DocumentMeta will be empty.
	ReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error)

	// Count returns the total number of result documents available.
	// A valid return value is only available when the cursor has been created with `Count` and not with `Stream`.
	Count() int64

	// Statistics returns the query execution statistics for this cursor.
	// This might not be valid if the cursor has been created with `Stream`
	Statistics() CursorStats

	// Plan returns the query execution plan for this cursor.
	Plan() CursorPlan
}

// CursorBatch is returned from a query, used to iterate over a list of documents.
// In contrast to Cursor, CursorBatch does not load all documents into memory, but returns them in batches and allows for retries in case of errors.
// Note that a Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
type CursorBatch interface {
	io.Closer

	// CloseWithContext run Close with specified Context
	CloseWithContext(ctx context.Context) error

	// HasMoreBatches returns true if the next call to ReadNextBatch does not return a NoMoreDocuments error.
	HasMoreBatches() bool

	// ReadNextBatch reads the next batch of documents from the cursor.
	// The result must be a pointer to a slice of documents.
	// E.g. `var result []MyStruct{}`.
	ReadNextBatch(ctx context.Context, result interface{}) error

	// RetryReadBatch retries the last batch read made by ReadNextBatch.
	// The result must be a pointer to a slice of documents.
	// E.g. `var result []MyStruct{}`.
	RetryReadBatch(ctx context.Context, result interface{}) error

	// Count returns the total number of result documents available.
	// A valid return value is only available when the cursor has been created with `Count` and not with `Stream`.
	Count() int64

	// Statistics returns the query execution statistics for this cursor.
	// This might not be valid if the cursor has been created with `Stream`
	Statistics() CursorStats

	// Plan returns the query execution plan for this cursor.
	Plan() CursorPlan
}

type CursorStats struct {
	// The total number of data-modification operations successfully executed.
	WritesExecutedInt uint64 `json:"writesExecuted,omitempty"`
	// The total number of data-modification operations that were unsuccessful
	WritesIgnoredInt uint64 `json:"writesIgnored,omitempty"`
	// The total number of documents iterated over when scanning a collection without an index.
	ScannedFullInt uint64 `json:"scannedFull,omitempty"`
	// The total number of documents iterated over when scanning a collection using an index.
	ScannedIndexInt uint64 `json:"scannedIndex,omitempty"`
	// The total number of documents that were removed after executing a filter condition in a FilterNode
	FilteredInt uint64 `json:"filtered,omitempty"`
	// The total number of documents that matched the search condition if the query's final LIMIT statement were not present.
	FullCountInt uint64 `json:"fullCount,omitempty"`
	// Query execution time (wall-clock time). value will be set from the outside
	ExecutionTimeInt float64 `json:"executionTime,omitempty"`

	HTTPRequests    uint64 `json:"httpRequests,omitempty"`
	PeakMemoryUsage uint64 `json:"peakMemoryUsage,omitempty"`

	// CursorsCreated the total number of cursor objects created during query execution. Cursor objects are created for index lookups.
	CursorsCreated uint64 `json:"cursorsCreated,omitempty"`
	// CursorsRearmed the total number of times an existing cursor object was repurposed.
	// Repurposing an existing cursor object is normally more efficient compared to destroying an existing cursor object
	// and creating a new one from scratch.
	CursorsRearmed uint64 `json:"cursorsRearmed,omitempty"`
	// CacheHits the total number of index entries read from in-memory caches for indexes of type edge or persistent.
	// This value will only be non-zero when reading from indexes that have an in-memory cache enabled,
	// and when the query allows using the in-memory cache (i.e. using equality lookups on all index attributes).
	CacheHits uint64 `json:"cacheHits,omitempty"`
	// CacheMisses the total number of cache read attempts for index entries that could not be served from in-memory caches for indexes of type edge or persistent.
	// This value will only be non-zero when reading from indexes that have an in-memory cache enabled,
	// the query allows using the in-memory cache (i.e. using equality lookups on all index attributes) and the looked up values are not present in the cache.
	CacheMisses uint64 `json:"cacheMisses,omitempty"`
}

type cursorData struct {
	Count       int64      `json:"count,omitempty"`       // the total number of result documents available (only available if the query was executed with the count attribute set)
	ID          string     `json:"id"`                    // id of temporary cursor created on the server (optional, see above)
	Result      jsonReader `json:"result,omitempty"`      // a stream of result documents (might be empty if query has no results)
	NextBatchID string     `json:"nextBatchId,omitempty"` // id of the next batch of the cursor on the server when `allowRetry` option is true
	HasMore     bool       `json:"hasMore,omitempty"`     // A boolean indicator whether there are more results available for the cursor on the server
	Extra       struct {
		Stats CursorStats `json:"stats,omitempty"`
		// Plan describes plan for a cursor.
		Plan CursorPlan `json:"plan,omitempty"`
	} `json:"extra"`
}

// CursorPlan describes execution plan for a query.
type CursorPlan struct {
	// Nodes describes a nested list of the execution plan nodes.
	Nodes []CursorPlanNodes `json:"nodes,omitempty"`
	// Rules describes a list with the names of the applied optimizer rules.
	Rules []string `json:"rules,omitempty"`
	// Collections describes list of the collections involved in the query.
	Collections []CursorPlanCollection `json:"collections,omitempty"`
	// Variables describes list of variables involved in the query.
	Variables []CursorPlanVariable `json:"variables,omitempty"`
	// EstimatedCost is an estimated cost of the query.
	EstimatedCost float64 `json:"estimatedCost,omitempty"`
	// EstimatedNrItems is an estimated number of results.
	EstimatedNrItems int `json:"estimatedNrItems,omitempty"`
	// IsModificationQuery describes whether the query contains write operations.
	IsModificationQuery bool `json:"isModificationQuery,omitempty"`
}

// CursorPlanNodes describes map of nodes which take part in the execution.
type CursorPlanNodes map[string]interface{}

// CursorPlanCollection describes a collection involved in the query.
type CursorPlanCollection struct {
	// Name is a name of collection.
	Name string `json:"name"`
	// Type describes how the collection is used: read, write or exclusive.
	Type string `json:"type"`
}

// CursorPlanVariable describes variable's settings.
type CursorPlanVariable struct {
	// ID is a variable's id.
	ID int `json:"id"`
	// Name is a variable's name.
	Name string `json:"name"`
	// IsDataFromCollection is set to true when data comes from a collection.
	IsDataFromCollection bool `json:"isDataFromCollection"`
	// IsFullDocumentFromCollection is set to true when all data comes from a collection.
	IsFullDocumentFromCollection bool `json:"isFullDocumentFromCollection"`
}
