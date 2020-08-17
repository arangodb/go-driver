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

package arangodb

import (
	"context"
	"net/http"
	"sync"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/pkg/connection"
	"github.com/pkg/errors"
)

func newCursor(db *database, endpoint string, data cursorData) *cursor {
	return &cursor{
		db:       db,
		endpoint: endpoint,
		data:     data,
	}
}

var _ Cursor = &cursor{}

type cursor struct {
	db *database

	endpoint string

	data cursorData

	lock sync.Mutex
}

func (c cursor) Close() error {
	panic("implement me")
}

func (c *cursor) HasMore() bool {
	return c.data.Result.HasMore() || c.data.HasMore
}

func (c *cursor) ReadDocument(ctx context.Context, result interface{}) (driver.DocumentMeta, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.readDocument(ctx, result)
}

func (c *cursor) readDocument(ctx context.Context, result interface{}) (driver.DocumentMeta, error) {
	if !c.data.Result.HasMore() {
		if err := c.getNextBatch(ctx); err != nil {
			return driver.DocumentMeta{}, err
		}
	}

	var data byteDecoder
	if err := c.data.Result.Read(&data); err != nil {
		return driver.DocumentMeta{}, err
	}

	var meta driver.DocumentMeta

	if err := data.Unmarshal(&meta); err != nil {
		// Ignore error
	}

	if err := data.Unmarshal(result); err != nil {
		return driver.DocumentMeta{}, err
	}

	return meta, nil
}

func (c *cursor) getNextBatch(ctx context.Context) error {
	if !c.data.HasMore {
		return errors.WithStack(driver.NoMoreDocumentsError{})
	}

	url := c.db.url("_api", "cursor", c.data.ID)

	resp, err := connection.CallPut(ctx, c.db.connection(), url, &c.data, nil, c.db.modifiers...)
	if err != nil {
		return err
	}

	switch resp.Code() {
	case http.StatusOK:
		return nil
	default:
		return connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c *cursor) Count() int64 {
	return c.data.Count
}

func (c *cursor) Statistics() CursorStats {
	return c.data.Extra.Stats
}
