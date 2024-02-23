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
//

package arangodb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCollectionIndexes(collection *collection) *collectionIndexes {
	d := &collectionIndexes{collection: collection}
	return d
}

var (
	_ CollectionIndexes = &collectionIndexes{}
)

type collectionIndexes struct {
	collection *collection
}

func (c *collectionIndexes) Index(ctx context.Context, name string) (IndexResponse, error) {
	urlEndpoint := c.collection.url("index", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
		IndexResponse         `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.collection.connection(), urlEndpoint, &response, c.collection.withModifiers()...)
	if err != nil {
		return IndexResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.IndexResponse, nil
	default:
		return IndexResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *collectionIndexes) IndexExists(ctx context.Context, name string) (bool, error) {
	urlEndpoint := c.collection.url("index", url.PathEscape(name))

	resp, err := connection.CallGet(ctx, c.collection.connection(), urlEndpoint, nil, c.collection.withModifiers()...)
	if err != nil {
		if shared.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c *collectionIndexes) Indexes(ctx context.Context) ([]IndexResponse, error) {
	urlEndpoint := c.collection.db.url("_api", "index")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Indexes               []IndexResponse `json:"indexes,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.collection.connection(), urlEndpoint, &response,
		c.collection.withModifiers(connection.WithQuery("collection", c.collection.name))...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Indexes, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c *collectionIndexes) EnsurePersistentIndex(ctx context.Context, fields []string, options *CreatePersistentIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type   IndexType `json:"type"`
		Fields []string  `json:"fields"`
		*CreatePersistentIndexOptions
	}{
		Type:                         PersistentIndexType,
		Fields:                       fields,
		CreatePersistentIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureGeoIndex(ctx context.Context, fields []string, options *CreateGeoIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type   IndexType `json:"type"`
		Fields []string  `json:"fields"`
		*CreateGeoIndexOptions
	}{
		Type:                  GeoIndexType,
		Fields:                fields,
		CreateGeoIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureTTLIndex(ctx context.Context, fields []string, expireAfter int, options *CreateTTLIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type        IndexType `json:"type"`
		Fields      []string  `json:"fields"`
		ExpireAfter int       `json:"expireAfter"`
		*CreateTTLIndexOptions
	}{
		Type:                  TTLIndexType,
		Fields:                fields,
		ExpireAfter:           expireAfter,
		CreateTTLIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureZKDIndex(ctx context.Context, fields []string, options *CreateZKDIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type   IndexType `json:"type"`
		Fields []string  `json:"fields"`
		*CreateZKDIndexOptions
	}{
		Type:                  ZKDIndexType,
		Fields:                fields,
		CreateZKDIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureMDIIndex(ctx context.Context, fields []string, options *CreateMDIIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type   IndexType `json:"type"`
		Fields []string  `json:"fields"`
		*CreateMDIIndexOptions
	}{
		Type:                  MDIIndexType,
		Fields:                fields,
		CreateMDIIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureMDIPrefixedIndex(ctx context.Context, fields []string, options *CreateMDIPrefixedIndexOptions) (IndexResponse, bool, error) {
	reqData := struct {
		Type   IndexType `json:"type"`
		Fields []string  `json:"fields"`
		*CreateMDIPrefixedIndexOptions
	}{
		Type:                          MDIPrefixedIndexType,
		Fields:                        fields,
		CreateMDIPrefixedIndexOptions: options,
	}

	result := responseIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newIndexResponse(&result), exist, err
}

func (c *collectionIndexes) EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (IndexResponse, bool, error) {
	if options == nil || options.Fields == nil || len(options.Fields) == 0 {
		return IndexResponse{}, false, errors.New("InvertedIndexOptions with non-empty Fields are required")
	}

	reqData := struct {
		Type IndexType `json:"type"`
		*InvertedIndexOptions
	}{
		Type:                 InvertedIndexType,
		InvertedIndexOptions: options,
	}

	result := responseInvertedIndex{}
	exist, err := c.ensureIndex(ctx, &reqData, &result)
	return newInvertedIndexResponse(&result), exist, err
}

func (c *collectionIndexes) ensureIndex(ctx context.Context, reqData interface{}, result interface{}) (bool, error) {
	urlEndpoint := c.collection.db.url("_api", "index")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}
	data := newUnmarshalInto(&result)

	resp, err := connection.CallPost(ctx, c.collection.connection(), urlEndpoint, newMultiUnmarshaller(&response, data), &reqData,
		c.collection.withModifiers(connection.WithQuery("collection", c.collection.name))...)
	if err != nil {
		return false, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return false, nil
	case http.StatusCreated:
		return true, nil
	default:
		return false, response.AsArangoErrorWithCode(code)
	}
}

func (c *collectionIndexes) DeleteIndex(ctx context.Context, name string) error {
	idx, err := c.Index(ctx, name)
	if err != nil {
		return errors.WithStack(err)
	}

	return c.DeleteIndexByID(ctx, idx.ID)
}

func (c *collectionIndexes) DeleteIndexByID(ctx context.Context, id string) error {
	urlEndpoint := c.collection.db.url("_api", "index", id)

	response := shared.ResponseStruct{}
	resp, err := connection.CallDelete(ctx, c.collection.connection(), urlEndpoint, &response, c.collection.withModifiers()...)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

type responseIndex struct {
	Name               string    `json:"name,omitempty"`
	Type               IndexType `json:"type"`
	IndexSharedOptions `json:",inline"`
	IndexOptions       `json:",inline"`
}

type responseInvertedIndex struct {
	Name                 string    `json:"name,omitempty"`
	Type                 IndexType `json:"type"`
	IndexSharedOptions   `json:",inline"`
	InvertedIndexOptions `json:",inline"`
}

func newIndexResponse(res *responseIndex) IndexResponse {
	return IndexResponse{
		Name:               res.Name,
		Type:               res.Type,
		IndexSharedOptions: res.IndexSharedOptions,
		RegularIndex:       &res.IndexOptions,
	}
}

func newInvertedIndexResponse(res *responseInvertedIndex) IndexResponse {
	return IndexResponse{
		Name:               res.Name,
		Type:               res.Type,
		IndexSharedOptions: res.IndexSharedOptions,
		InvertedIndex:      &res.InvertedIndexOptions,
	}
}

func (i *IndexResponse) UnmarshalJSON(data []byte) error {
	var respSimple struct {
		Type IndexType `json:"type"`
		Name string    `json:"name"`
	}
	if err := json.Unmarshal(data, &respSimple); err != nil {
		return err
	}

	i.Name = respSimple.Name
	i.Type = respSimple.Type

	if respSimple.Type == InvertedIndexType {
		result := responseInvertedIndex{}
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}

		i.IndexSharedOptions = result.IndexSharedOptions
		i.InvertedIndex = &result.InvertedIndexOptions
	} else {
		result := responseIndex{}
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}

		i.IndexSharedOptions = result.IndexSharedOptions
		i.RegularIndex = &result.IndexOptions
	}

	return nil
}
