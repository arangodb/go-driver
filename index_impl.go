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
	"strings"
)

// newIndex creates a new Index implementation.
func newIndex(id string, col *collection) (Index, error) {
	if id == "" {
		return nil, WithStack(InvalidArgumentError{Message: "id is empty"})
	}
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, WithStack(InvalidArgumentError{Message: "id must be `collection/name`"})
	}
	if col == nil {
		return nil, WithStack(InvalidArgumentError{Message: "col is nil"})
	}
	return &index{
		id:   id,
		col:  col,
		db:   col.db,
		conn: col.conn,
	}, nil
}

type index struct {
	id   string
	db   *database
	col  *collection
	conn Connection
}

// relPath creates the relative path to this index (`_db/<db-name>/_api/index`)
func (i *index) relPath() string {
	return path.Join(i.db.relPath(), "_api", "index")
}

// Name returns the name of the index.
func (i *index) Name() string {
	parts := strings.Split(i.id, "/")
	return parts[1]
}

// Remove removes the entire index.
// If the index does not exist, a NotFoundError is returned.
func (i *index) Remove(ctx context.Context) error {
	req, err := i.conn.NewRequest("DELETE", path.Join(i.relPath(), i.id))
	if err != nil {
		return WithStack(err)
	}
	resp, err := i.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}
