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

import (
	"context"
	"encoding/json"
	"path"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// newCursor creates a new Cursor implementation.
func newCursor(data cursorData, endpoint string, db *database, allowDirtyReads bool) (Cursor, error) {
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &cursor{
		cursorData:      data,
		endpoint:        endpoint,
		db:              db,
		conn:            db.conn,
		allowDirtyReads: allowDirtyReads,
	}, nil
}

type cursor struct {
	cursorData
	endpoint         string
	resultIndex      int
	db               *database
	conn             Connection
	closed           int32
	closeMutex       sync.Mutex
	allowDirtyReads  bool
	lastReadWasDirty bool
}

type cursorStats struct {
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
}

type cursorData struct {
	Count   int64        `json:"count,omitempty"`   // the total number of result documents available (only available if the query was executed with the count attribute set)
	ID      string       `json:"id"`                // id of temporary cursor created on the server (optional, see above)
	Result  []*RawObject `json:"result,omitempty"`  // an array of result documents (might be empty if query has no results)
	HasMore bool         `json:"hasMore,omitempty"` // A boolean indicator whether there are more results available for the cursor on the server
	Extra   struct {
		Stats cursorStats `json:"stats,omitempty"`
	} `json:"extra"`
}

// relPath creates the relative path to this cursor (`_db/<db-name>/_api/cursor`)
func (c *cursor) relPath() string {
	return path.Join(c.db.relPath(), "_api", "cursor")
}

// Name returns the name of the collection.
func (c *cursor) HasMore() bool {
	return c.resultIndex < len(c.Result) || c.cursorData.HasMore
}

// Count returns the total number of result documents available.
// A valid return value is only available when the cursor has been created with a context that was
// prepare with `WithQueryCount`.
func (c *cursor) Count() int64 {
	return c.cursorData.Count
}

// Close deletes the cursor and frees the resources associated with it.
func (c *cursor) Close() error {
	if c == nil {
		// Avoid panics in the case that someone defer's a close before checking that the cursor is not nil.
		return nil
	}
	if c := atomic.LoadInt32(&c.closed); c != 0 {
		return nil
	}
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()
	if c.closed == 0 {
		if c.cursorData.ID != "" {
			// Force use of initial endpoint
			ctx := WithEndpoint(nil, c.endpoint)

			req, err := c.conn.NewRequest("DELETE", path.Join(c.relPath(), c.cursorData.ID))
			if err != nil {
				return WithStack(err)
			}
			resp, err := c.conn.Do(ctx, req)
			if err != nil {
				return WithStack(err)
			}
			if err := resp.CheckStatus(202); err != nil {
				return WithStack(err)
			}
		}
		atomic.StoreInt32(&c.closed, 1)
	}
	return nil
}

// ReadDocumentOnly reads the next document from the cursor.
// If the cursor has no more documents, a NoMoreDocuments error is returned.
func (c *cursor) ReadDocumentOnly(ctx context.Context, result interface{}) error {
	_, err := c.readDocumentMeta(ctx, result, true)
	return err
}

// ReadDocument reads the next document from the cursor.
// The document data is stored into result, and the document meta data is returned.
// If the cursor has no more documents, a NoMoreDocuments error is returned.
func (c *cursor) ReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error) {
	return c.readDocumentMeta(ctx, result, false)
}

// readDocumentMeta reads the next document from the cursor.
// The document data is stored into result, and the document meta data is optionally returned.
// If the cursor has no more documents, a NoMoreDocuments error is returned.
func (c *cursor) readDocumentMeta(ctx context.Context, result interface{}, excludeMeta bool) (DocumentMeta, error) {
	// Force use of initial endpoint
	ctx = WithEndpoint(ctx, c.endpoint)

	if c.resultIndex >= len(c.Result) && c.cursorData.HasMore {
		// This is required since we are interested if this was a dirty read
		// but we do not want to trash the users bool reference.
		var wasDirtyRead bool
		fetchctx := ctx
		if c.allowDirtyReads {
			fetchctx = WithAllowDirtyReads(ctx, &wasDirtyRead)
		}

		// Fetch next batch
		req, err := c.conn.NewRequest("PUT", path.Join(c.relPath(), c.cursorData.ID))
		if err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		cs := applyContextSettings(fetchctx, req)
		resp, err := c.conn.Do(fetchctx, req)
		if err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		if err := resp.CheckStatus(200); err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		loadContextResponseValues(cs, resp)
		var data cursorData
		if err := resp.ParseBody("", &data); err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		c.cursorData = data
		c.resultIndex = 0
		c.lastReadWasDirty = wasDirtyRead
	}
	// ReadDocument should act as if it would actually do a read
	// hence update the bool reference
	if c.allowDirtyReads {
		setDirtyReadFlagIfRequired(ctx, c.lastReadWasDirty)
	}

	index := c.resultIndex
	if index >= len(c.Result) {
		// Out of data
		return DocumentMeta{}, WithStack(NoMoreDocumentsError{})
	}
	c.resultIndex++
	var meta DocumentMeta
	resultPtr := c.Result[index]
	if resultPtr == nil {
		// Got NULL result
		rv := reflect.ValueOf(result)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return DocumentMeta{}, WithStack(&json.InvalidUnmarshalError{Type: reflect.TypeOf(result)})
		}
		e := rv.Elem()
		e.Set(reflect.Zero(e.Type()))
	} else {
		if !excludeMeta {
			if err := c.conn.Unmarshal(*resultPtr, &meta); err != nil {
				// If a cursor returns something other than a document, this will fail.
				// Just ignore it.
			}
		}
		if err := c.conn.Unmarshal(*resultPtr, result); err != nil {
			return DocumentMeta{}, WithStack(err)
		}
	}
	return meta, nil
}

// Return execution statistics for this cursor. This might not
// be valid if the cursor has been created with a context that was
// prepared with `WithStream`
func (c *cursor) Statistics() QueryStatistics {
	return c.cursorData.Extra.Stats
}

// the total number of data-modification operations successfully executed.
func (cs cursorStats) WritesExecuted() int64 {
	return cs.WritesExecutedInt
}

// The total number of data-modification operations that were unsuccessful
func (cs cursorStats) WritesIgnored() int64 {
	return cs.WritesIgnoredInt
}

// The total number of documents iterated over when scanning a collection without an index.
func (cs cursorStats) ScannedFull() int64 {
	return cs.ScannedFullInt
}

// The total number of documents iterated over when scanning a collection using an index.
func (cs cursorStats) ScannedIndex() int64 {
	return cs.ScannedIndexInt
}

// the total number of documents that were removed after executing a filter condition in a FilterNode
func (cs cursorStats) Filtered() int64 {
	return cs.FilteredInt
}

// Returns the numer of results before the last LIMIT in the query was applied.
// A valid return value is only available when the has been created with a context that was
// prepared with `WithFullCount`. Additionally this will also not return a valid value if
// the context was prepared with `WithStream`.
func (cs cursorStats) FullCount() int64 {
	return cs.FullCountInt
}

// query execution time (wall-clock time). value will be set from the outside
func (cs cursorStats) ExecutionTime() time.Duration {
	return time.Duration(cs.ExecutionTimeInt) * time.Second
}
