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
)

type indexData struct {
	ID          string   `json:"id,omitempty"`
	Type        string   `json:"type"`
	Fields      []string `json:"fields,omitempty"`
	Unique      *bool    `json:"unique,omitempty"`
	Deduplicate *bool    `json:"deduplicate,omitempty"`
	Sparse      *bool    `json:"sparse,omitempty"`
	GeoJSON     *bool    `json:"geoJson,omitempty"`
	MinLength   int      `json:"minLength,omitempty"`
}

type indexListResponse struct {
	Indexes []indexData `json:"indexes,omitempty"`
}

// Index opens a connection to an existing index within the collection.
// If no index with given name exists, an NotFoundError is returned.
func (c *collection) Index(ctx context.Context, name string) (Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data indexData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	idx, err := newIndex(data.ID, c)
	if err != nil {
		return nil, WithStack(err)
	}
	return idx, nil
}

// IndexExists returns true if an index with given name exists within the collection.
func (c *collection) IndexExists(ctx context.Context, name string) (bool, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

// Indexes returns a list of all indexes in the collection.
func (c *collection) Indexes(ctx context.Context) ([]Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.db.relPath(), "_api", "index"))
	if err != nil {
		return nil, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data indexListResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Index, 0, len(data.Indexes))
	for _, x := range data.Indexes {
		idx, err := newIndex(x.ID, c)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, idx)
	}
	return result, nil
}

// EnsureFullTextIndex creates a fulltext index in the collection, if it does not already exist.
//
// Fields is a slice of attribute names. Currently, the slice is limited to exactly one attribute.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureFullTextIndex(ctx context.Context, fields []string, options *EnsureFullTextIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   "fulltext",
		Fields: fields,
	}
	if options != nil {
		input.MinLength = options.MinLength
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureGeoIndex creates a hash index in the collection, if it does not already exist.
//
// Fields is a slice with one or two attribute paths. If it is a slice with one attribute path location,
// then a geo-spatial index on all documents is created using location as path to the coordinates.
// The value of the attribute must be a slice with at least two double values. The slice must contain the latitude (first value)
// and the longitude (second value). All documents, which do not have the attribute path or with value that are not suitable, are ignored.
// If it is a slice with two attribute paths latitude and longitude, then a geo-spatial index on all documents is created
// using latitude and longitude as paths the latitude and the longitude. The value of the attribute latitude and of the
// attribute longitude must a double. All documents, which do not have the attribute paths or which values are not suitable, are ignored.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureGeoIndex(ctx context.Context, fields []string, options *EnsureGeoIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   "geo",
		Fields: fields,
	}
	if options != nil {
		input.GeoJSON = &options.GeoJSON
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureHashIndex creates a hash index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureHashIndex(ctx context.Context, fields []string, options *EnsureHashIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   "hash",
		Fields: fields,
	}
	off := false
	if options != nil {
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		if options.NoDeduplicate {
			input.Deduplicate = &off
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsurePersistentIndex(ctx context.Context, fields []string, options *EnsurePersistentIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   "persistent",
		Fields: fields,
	}
	if options != nil {
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureSkipListIndex creates a skiplist index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureSkipListIndex(ctx context.Context, fields []string, options *EnsureSkipListIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   "skiplist",
		Fields: fields,
	}
	off := false
	if options != nil {
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		if options.NoDeduplicate {
			input.Deduplicate = &off
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// ensureIndex creates a persistent index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) ensureIndex(ctx context.Context, options indexData) (Index, bool, error) {
	req, err := c.conn.NewRequest("POST", path.Join(c.db.relPath(), "_api/index"))
	if err != nil {
		return nil, false, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	if _, err := req.SetBody(options); err != nil {
		return nil, false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, false, WithStack(err)
	}
	if err := resp.CheckStatus(200, 201); err != nil {
		return nil, false, WithStack(err)
	}
	created := resp.StatusCode() == 201
	var data indexData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, false, WithStack(err)
	}
	idx, err := newIndex(data.ID, c)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}
