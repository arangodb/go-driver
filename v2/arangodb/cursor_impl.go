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
	"encoding/json"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCursor(db *database, endpoint string, data cursorData) *cursor {
	c := &cursor{
		db:       db,
		endpoint: endpoint,
		data:     data,
	}

	if data.NextBatchID != "" {
		c.retryData = &retryData{
			cursorID:       data.ID,
			currentBatchID: "1",
		}
	}

	return c
}

var _ Cursor = &cursor{}

type cursor struct {
	db        *database
	endpoint  string
	closed    bool
	data      cursorData
	lock      sync.Mutex
	retryData *retryData
}

type retryData struct {
	cursorID       string
	currentBatchID string
}

func (c *cursor) Close() error {
	return c.CloseWithContext(context.Background())
}

func (c *cursor) CloseWithContext(ctx context.Context) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil
	}

	if c.data.ID == "" {
		c.closed = true
		c.data = cursorData{}
		return nil
	}

	url := c.db.url("_api", "cursor", c.data.ID)

	resp, err := connection.CallDelete(ctx, c.db.connection(), url, &c.data, c.db.modifiers...)
	if err != nil {
		return err
	}
	c.closed = true

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		c.data = cursorData{}
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c *cursor) HasMore() bool {
	return c.data.Result.HasMore() || c.data.HasMore
}

func (c *cursor) HasMoreBatches() bool {
	return c.data.HasMore
}

func (c *cursor) ReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.readDocument(ctx, result)
}

func (c *cursor) ReadNextBatch(ctx context.Context, result interface{}) error {
	err := c.getNextBatch(ctx, "")
	if err != nil {
		return err
	}

	return json.Unmarshal(c.data.Result.in, result)
}

func (c *cursor) RetryReadBatch(ctx context.Context, result interface{}) error {
	err := c.getNextBatch(ctx, c.retryData.currentBatchID)
	if err != nil {
		return err
	}

	return json.Unmarshal(c.data.Result.in, result)
}

func (c *cursor) readDocument(ctx context.Context, result interface{}) (DocumentMeta, error) {
	if c.closed {
		return DocumentMeta{}, shared.NoMoreDocumentsError{}
	}

	if !c.data.Result.HasMore() {
		if err := c.getNextBatch(ctx, ""); err != nil {
			return DocumentMeta{}, err
		}
	}

	var data byteDecoder
	if err := c.data.Result.Read(&data); err != nil {
		return DocumentMeta{}, err
	}

	var meta DocumentMeta

	if err := data.Unmarshal(&meta); err != nil {
		// Ignore error
	}

	if err := data.Unmarshal(result); err != nil {
		return DocumentMeta{}, err
	}

	return meta, nil
}

func (c *cursor) getNextBatch(ctx context.Context, retryBatchID string) error {
	if !c.data.HasMore && retryBatchID == "" {
		return errors.WithStack(shared.NoMoreDocumentsError{})
	}

	url := c.db.url("_api", "cursor", c.data.ID)
	// If we have a NextBatchID, use it
	if c.data.NextBatchID != "" {
		url = c.db.url("_api", "cursor", c.data.ID, c.data.NextBatchID)
	}
	// We have to retry the batch instead of fetching the next one
	if retryBatchID != "" {
		url = c.db.url("_api", "cursor", c.retryData.cursorID, retryBatchID)
	}

	// Update currentBatchID before fetching the next batch (no retry case)
	if c.data.NextBatchID != "" && retryBatchID == "" {
		c.retryData.currentBatchID = c.data.NextBatchID
	}

	var data cursorData

	resp, err := connection.CallPost(ctx, c.db.connection(), url, &data, nil, c.db.modifiers...)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		c.data = data
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c *cursor) Count() int64 {
	return c.data.Count
}

func (c *cursor) Statistics() CursorStats {
	return c.data.Extra.Stats
}

// Plan returns the query execution plan for this cursor.
func (c *cursor) Plan() CursorPlan {
	return c.data.Extra.Plan
}
