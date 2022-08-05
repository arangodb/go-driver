//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
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
}

// CursorStats TODO: all these int64 should be changed into uint64
type CursorStats struct {
	// The total number of data-modification operations successfully executed.
	WritesExecutedInt int64 `json:"writesExecuted,omitempty"`
	// The total number of data-modification operations that were unsuccessful
	WritesIgnoredInt int64 `json:"writesIgnored,omitempty"`
	// The total number of documents iterated over when scanning a collection without an index.
	ScannedFullInt int64 `json:"scannedFull,omitempty"`
	// The total number of documents iterated over when scanning a collection using an index.
	ScannedIndexInt int64 `json:"scannedIndex,omitempty"`
	// The total number of documents that were removed after executing a filter condition in a FilterNode
	FilteredInt int64 `json:"filtered,omitempty"`
	// The total number of documents that matched the search condition if the query's final LIMIT statement were not present.
	FullCountInt int64 `json:"fullCount,omitempty"`
	// Query execution time (wall-clock time). value will be set from the outside
	ExecutionTimeInt float64 `json:"executionTime,omitempty"`

	HttpRequests    uint64 `json:"httpRequests,omitempty"`
	PeakMemoryUsage uint64 `json:"peakMemoryUsage,omitempty"`

	CursorsCreated uint64 `json:"cursorsCreated,omitempty"`
	CursorsRearmed uint64 `json:"cursorsRearmed,omitempty"`
	CacheHits      uint64 `json:"cacheHits,omitempty"`
	CacheMisses    uint64 `json:"cacheMisses,omitempty"`
}

type cursorData struct {
	Count   int64      `json:"count,omitempty"`   // the total number of result documents available (only available if the query was executed with the count attribute set)
	ID      string     `json:"id"`                // id of temporary cursor created on the server (optional, see above)
	Result  jsonReader `json:"result,omitempty"`  // a stream of result documents (might be empty if query has no results)
	HasMore bool       `json:"hasMore,omitempty"` // A boolean indicator whether there are more results available for the cursor on the server
	Extra   struct {
		Stats CursorStats `json:"stats,omitempty"`
	} `json:"extra"`
}
