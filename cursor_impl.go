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
	"path"
	"sync"
	"sync/atomic"
)

// newCursor creates a new Cursor implementation.
func newCursor(data cursorData, endpoint string, db *database) (Cursor, error) {
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &cursor{
		cursorData: data,
		endpoint:   endpoint,
		db:         db,
		conn:       db.conn,
	}, nil
}

type cursor struct {
	cursorData
	endpoint    string
	resultIndex int
	db          *database
	conn        Connection
	closed      int32
	closeMutex  sync.Mutex
}

type cursorData struct {
	Count   int64        `json:"count,omitempty"`   // the total number of result documents available (only available if the query was executed with the count attribute set)
	ID      string       `json:"id"`                // id of temporary cursor created on the server (optional, see above)
	Result  []*RawObject `json:"result,omitempty"`  // an array of result documents (might be empty if query has no results)
	HasMore bool         `json:"hasMore,omitempty"` // A boolean indicator whether there are more results available for the cursor on the server
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

// ReadDocument reads the next document from the cursor.
// The document data is stored into result, the document meta data is returned.
// If the cursor has no more documents, a NoMoreDocuments error is returned.
func (c *cursor) ReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error) {
	// Force use of initial endpoint
	ctx = WithEndpoint(ctx, c.endpoint)

	if c.resultIndex >= len(c.Result) && c.cursorData.HasMore {
		// Fetch next batch
		req, err := c.conn.NewRequest("PUT", path.Join(c.relPath(), c.cursorData.ID))
		if err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		resp, err := c.conn.Do(ctx, req)
		if err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		if err := resp.CheckStatus(200); err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		var data cursorData
		if err := resp.ParseBody("", &data); err != nil {
			return DocumentMeta{}, WithStack(err)
		}
		c.cursorData = data
		c.resultIndex = 0
	}

	index := c.resultIndex
	if index >= len(c.Result) {
		// Out of data
		return DocumentMeta{}, WithStack(NoMoreDocumentsError{})
	}
	c.resultIndex++
	var meta DocumentMeta
	if err := c.conn.Unmarshal(*c.Result[index], &meta); err != nil {
		// If a cursor returns something other than a document, this will fail.
		// Just ignore it.
	}
	if err := c.conn.Unmarshal(*c.Result[index], result); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}
