//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCollection(db *database, name string, modifiers ...connection.RequestModifier) *collection {
	d := &collection{db: db, name: name, modifiers: append(db.modifiers, modifiers...)}

	d.collectionDocuments = newCollectionDocuments(d)
	d.collectionIndexes = newCollectionIndexes(d)

	return d
}

var _ Collection = &collection{}

type collection struct {
	name string

	db *database

	modifiers []connection.RequestModifier

	*collectionDocuments
	*collectionIndexes
}

func (c collection) Remove(ctx context.Context) error {
	return c.RemoveWithOptions(ctx, nil)
}

func (c collection) RemoveWithOptions(ctx context.Context, opts *RemoveCollectionOptions) error {
	url := c.url("collection")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, c.connection(), url, &response, c.withModifiers(opts.modifyRequest)...)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Truncate(ctx context.Context) error {
	urlEndpoint := c.url("collection", "truncate")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, struct{}{}, c.withModifiers()...)
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

func (c collection) Count(ctx context.Context) (int64, error) {
	urlEndpoint := c.url("collection", "count")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Count                 int64 `json:"count,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.connection(), urlEndpoint, &response, c.withModifiers()...)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Count, nil
	default:
		return 0, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Properties(ctx context.Context) (CollectionProperties, error) {
	urlEndpoint := c.url("collection", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionProperties  `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.connection(), urlEndpoint, &response, c.withModifiers()...)
	if err != nil {
		return CollectionProperties{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionProperties, nil
	default:
		return CollectionProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) SetPropertiesV2(ctx context.Context, options SetCollectionPropertiesOptionsV2) error {
	urlEndpoint := c.url("collection", "properties")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, options, c.withModifiers()...)
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

func (c collection) Name() string {
	return c.name
}

func (c collection) Database() Database {
	return c.db
}

func (c collection) withModifiers(modifiers ...connection.RequestModifier) []connection.RequestModifier {
	if len(modifiers) == 0 {
		return c.modifiers
	}

	z := len(c.modifiers)

	d := make([]connection.RequestModifier, len(modifiers)+z)

	copy(d, c.modifiers)

	for i, v := range modifiers {
		d[i+z] = v
	}

	return d
}

func (c collection) connection() connection.Connection {
	return c.db.connection()
}

// creates the relative path to this collection (`_db/<db-name>/_api/<document|collection|index>/<collection-name>`)
func (c collection) url(api string, parts ...string) string {
	return c.db.url(append([]string{"_api", api, url.PathEscape(c.name)}, parts...)...)
}

func (c collection) Shards(ctx context.Context, details bool) (CollectionShards, error) {
	var body struct {
		shared.ResponseStruct `json:",inline"`
		CollectionShards      `json:",inline"`
	}

	resp, err := connection.CallGet(
		ctx, c.connection(), c.url("collection", "shards"), &body,
		c.withModifiers(connection.WithQuery("details", boolToString(details)))...,
	)
	if err != nil {
		return CollectionShards{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return body.CollectionShards, nil
	default:
		return CollectionShards{}, body.AsArangoErrorWithCode(code)
	}
}

func (c collection) Statistics(ctx context.Context, details bool) (CollectionStatistics, error) {
	urlEndpoint := c.url("collection", "figures")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionStatistics  `json:",inline"`
	}

	resp, err := connection.CallGet(
		ctx, c.connection(), urlEndpoint, &response,
		c.withModifiers(connection.WithQuery("details", boolToString(details)))...,
	)
	if err != nil {
		return CollectionStatistics{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionStatistics, nil
	default:
		return CollectionStatistics{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Revision(ctx context.Context) (CollectionProperties, error) {
	urlEndpoint := c.url("collection", "revision")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionProperties  `json:",inline"`
	}

	resp, err := connection.CallGet(
		ctx, c.connection(), urlEndpoint, &response, c.withModifiers()...,
	)

	if err != nil {
		return CollectionProperties{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionProperties, nil
	default:
		return CollectionProperties{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Checksum(ctx context.Context, withRevisions *bool, withData *bool) (CollectionChecksum, error) {
	urlEndpoint := c.url("collection", "checksum")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionChecksum    `json:",inline"`
	}

	// Prepare query modifiers
	var modifiers []connection.RequestModifier
	if *withRevisions {
		modifiers = append(modifiers, connection.WithQuery("withRevisions", boolToString(*withRevisions)))
	}
	if *withData {
		modifiers = append(modifiers, connection.WithQuery("withData", boolToString(*withData)))
	}

	resp, err := connection.CallGet(
		ctx, c.connection(), urlEndpoint, &response,
		c.withModifiers(modifiers...)...,
	)
	if err != nil {
		return CollectionChecksum{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionChecksum, nil
	default:
		return CollectionChecksum{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) ResponsibleShard(ctx context.Context, options map[string]interface{}) (string, error) {
	urlEndpoint := c.url("collection", "responsibleShard")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ShardId               string `json:"shardId,omitempty"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, options, c.withModifiers()...)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ShardId, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c collection) LoadIndexesIntoMemory(ctx context.Context) (bool, error) {
	urlEndpoint := c.url("collection", "loadIndexesIntoMemory")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                bool `json:"result"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, nil, c.withModifiers()...)
	if err != nil {
		return false, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return false, response.AsArangoErrorWithCode(code)
	}
}

// Renaming collections is not supported in cluster deployments.
func (c collection) Rename(ctx context.Context, req RenameCollectionRequest) (CollectionInfo, error) {
	urlEndpoint := c.url("collection", "rename")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionInfo        `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, req, c.withModifiers()...)
	if err != nil {
		return CollectionInfo{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionInfo, nil
	default:
		return CollectionInfo{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) RecalculateCount(ctx context.Context) (bool, *int64, error) {
	urlEndpoint := c.url("collection", "recalculateCount")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Count                 *int64 `json:"count,omitempty"`
		Result                bool   `json:"result"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, nil, c.withModifiers()...)
	if err != nil {
		zero := int64(0)
		return false, &zero, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, response.Count, nil
	default:
		zero := int64(0)
		return false, &zero, response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Compact(ctx context.Context) (CollectionInfo, error) {
	urlEndpoint := c.url("collection", "compact")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		CollectionInfo        `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.connection(), urlEndpoint, &response, nil, c.withModifiers()...)
	if err != nil {
		return CollectionInfo{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CollectionInfo, nil
	default:
		return CollectionInfo{}, response.AsArangoErrorWithCode(code)
	}
}

type RemoveCollectionOptions struct {
	// IsSystem when set to true allows to remove system collections.
	// Use on your own risk!
	IsSystem *bool
}

func (o *RemoveCollectionOptions) modifyRequest(r connection.Request) error {
	if o == nil {
		return nil
	}
	if o.IsSystem != nil {
		r.AddQuery("isSystem", boolToString(*o.IsSystem))
	}
	return nil
}
