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

// indexStringToType converts a string representation of an index to IndexType
func indexStringToType(indexTypeString string) (IndexType, error) {
	switch indexTypeString {
	case string(FullTextIndex):
		return FullTextIndex, nil
	case string(HashIndex):
		return HashIndex, nil
	case string(SkipListIndex):
		return SkipListIndex, nil
	case string(PrimaryIndex):
		return PrimaryIndex, nil
	case string(PersistentIndex):
		return PersistentIndex, nil
	case string(GeoIndex), "geo1", "geo2":
		return GeoIndex, nil
	case string(EdgeIndex):
		return EdgeIndex, nil
	case string(TTLIndex):
		return TTLIndex, nil
	case string(ZKDIndex):
		return ZKDIndex, nil
	default:
		return "", WithStack(InvalidArgumentError{Message: "unknown index type"})
	}
}

// newIndex creates a new Index implementation.
func newIndex(data indexData, col *collection) (Index, error) {
	if data.ID == "" {
		return nil, WithStack(InvalidArgumentError{Message: "id is empty"})
	}
	parts := strings.Split(data.ID, "/")
	if len(parts) != 2 {
		return nil, WithStack(InvalidArgumentError{Message: "id must be `collection/name`"})
	}
	if col == nil {
		return nil, WithStack(InvalidArgumentError{Message: "col is nil"})
	}
	indexType, err := indexStringToType(data.Type)
	if err != nil {
		return nil, WithStack(err)
	}
	return &index{
		indexData: data,
		indexType: indexType,
		col:       col,
		db:        col.db,
		conn:      col.conn,
	}, nil
}

type index struct {
	indexData
	indexType IndexType
	db        *database
	col       *collection
	conn      Connection
}

// relPath creates the relative path to this index (`_db/<db-name>/_api/index`)
func (i *index) relPath() string {
	return path.Join(i.db.relPath(), "_api", "index")
}

// Name returns the name of the index.
func (i *index) Name() string {
	parts := strings.Split(i.indexData.ID, "/")
	return parts[1]
}

// ID returns the ID of the index.
func (i *index) ID() string {
	return i.indexData.ID
}

// UserName returns the user provided name of the index or empty string if non is provided.
func (i *index) UserName() string {
	return i.indexData.Name
}

// Type returns the type of the index
func (i *index) Type() IndexType {
	return i.indexType
}

// Fields returns a list of attributes of this index.
func (i *index) Fields() []string {
	return i.indexData.Fields
}

// Unique returns if this index is unique.
func (i *index) Unique() bool {
	if i.indexData.Unique == nil {
		return false
	}
	return *i.indexData.Unique
}

// Deduplicate returns deduplicate setting of this index.
func (i *index) Deduplicate() bool {
	if i.indexData.Deduplicate == nil {
		return false
	}
	return *i.indexData.Deduplicate
}

// Sparse returns if this is a sparse index or not.
func (i *index) Sparse() bool {
	if i.indexData.Sparse == nil {
		return false
	}
	return *i.indexData.Sparse
}

// GeoJSON returns if geo json was set for this index or not.
func (i *index) GeoJSON() bool {
	if i.indexData.GeoJSON == nil {
		return false
	}
	return *i.indexData.GeoJSON
}

// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
func (i *index) InBackground() bool {
	if i.indexData.InBackground == nil {
		return false
	}
	return *i.indexData.InBackground
}

// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
func (i *index) Estimates() bool {
	if i.indexData.Estimates == nil {
		return false
	}
	return *i.indexData.Estimates
}

// MinLength returns min length for this index if set.
func (i *index) MinLength() int {
	return i.indexData.MinLength
}

// ExpireAfter returns an expire after for this index if set.
func (i *index) ExpireAfter() int {
	return i.indexData.ExpireAfter
}

// Remove removes the entire index.
// If the index does not exist, a NotFoundError is returned.
func (i *index) Remove(ctx context.Context) error {
	req, err := i.conn.NewRequest("DELETE", path.Join(i.relPath(), i.indexData.ID))
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
