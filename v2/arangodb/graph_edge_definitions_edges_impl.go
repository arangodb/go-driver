//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newEdgeCollection(edge *graph, edgeColName string) *edgeCollection {
	return &edgeCollection{
		graph:       edge,
		edgeColName: edgeColName,
	}
}

var _ Edge = &edgeCollection{}

type edgeCollection struct {
	edgeColName string

	modifiers []connection.RequestModifier

	graph *graph
}

// creates the relative path to this edge (`_db/<db-name>/_api/gharial/<graph-name>/edge/<collection-name>`)
func (v *edgeCollection) url(parts ...string) string {
	p := append([]string{"edge", v.edgeColName}, parts...)
	return v.graph.url(p...)
}

func (v *edgeCollection) Name() string {
	return v.edgeColName
}

func (v *edgeCollection) GetEdge(ctx context.Context, key string, result interface{}, opts *GetEdgeOptions) error {
	url := v.url(key)

	response := struct {
		*shared.ResponseStruct `json:",inline"`
		Edge                   *UnmarshalInto `json:"edge,omitempty"`
	}{
		Edge: newUnmarshalInto(result),
	}

	resp, err := connection.CallGet(ctx, v.graph.db.connection(), url, &response, append(v.graph.db.modifiers, opts.modifyRequest)...)
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

func (g *GetEdgeOptions) modifyRequest(r connection.Request) error {
	if g == nil {
		return nil
	}

	if g.Rev != "" {
		r.AddQuery(QueryRev, g.Rev)
	}

	if g.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, g.IfMatch)
	}

	if g.IfNoneMatch != "" {
		r.AddHeader(HeaderIfNoneMatch, g.IfNoneMatch)
	}

	if g.TransactionID != "" {
		r.AddHeader(HeaderTransaction, g.TransactionID)
	}

	return nil
}

func (v *edgeCollection) CreateEdge(ctx context.Context, edge interface{}, opts *CreateEdgeOptions) (EdgeCreateResponse, error) {
	url := v.url()

	var meta EdgeCreateResponse

	if opts != nil {
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:"edge,omitempty"`
		*shared.ResponseStruct `json:",inline"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,
		New:            newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPost(ctx, v.graph.db.connection(), url, &response, edge, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return EdgeCreateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return EdgeCreateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *CreateEdgeOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*c.WaitForSync))
	}

	if c.NewObject != nil {
		r.AddQuery(QueryReturnNew, "true")
	}

	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}

	return nil
}

func (v *edgeCollection) UpdateEdge(ctx context.Context, key string, newValue interface{}, opts *EdgeUpdateOptions) (EdgeUpdateResponse, error) {
	url := v.url(key)

	var meta EdgeUpdateResponse

	if opts != nil {
		meta.Old = opts.OldObject
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:"edge,inline"`
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,
		Old:            newUnmarshalInto(meta.Old),
		New:            newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPatch(ctx, v.graph.db.connection(), url, &response, newValue, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return EdgeUpdateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return EdgeUpdateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (v *EdgeUpdateOptions) modifyRequest(r connection.Request) error {
	if v == nil {
		return nil
	}

	if v.WaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*v.WaitForSync))
	}

	if v.NewObject != nil {
		r.AddQuery(QueryReturnNew, "true")
	}

	if v.OldObject != nil {
		r.AddQuery(QueryReturnOld, "true")
	}

	if v.KeepNull != nil {
		r.AddQuery(QueryKeepNull, boolToString(*v.KeepNull))
	}

	if v.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, v.IfMatch)
	}

	if v.TransactionID != "" {
		r.AddHeader(HeaderTransaction, v.TransactionID)
	}

	return nil
}

func (v *edgeCollection) ReplaceEdge(ctx context.Context, key string, newValue interface{}, opts *EdgeReplaceOptions) (EdgeReplaceResponse, error) {
	url := v.url(key)

	var meta EdgeReplaceResponse

	if opts != nil {
		meta.Old = opts.OldObject
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:"edge,omitempty"`
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,

		Old: newUnmarshalInto(meta.Old),
		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPut(ctx, v.graph.db.connection(), url, &response, newValue, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return EdgeReplaceResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return EdgeReplaceResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (v *EdgeReplaceOptions) modifyRequest(r connection.Request) error {
	if v == nil {
		return nil
	}

	if v.WaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*v.WaitForSync))
	}

	if v.NewObject != nil {
		r.AddQuery(QueryReturnNew, "true")
	}

	if v.OldObject != nil {
		r.AddQuery(QueryReturnOld, "true")
	}

	if v.KeepNull != nil {
		r.AddQuery(QueryKeepNull, boolToString(*v.KeepNull))
	}

	if v.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, v.IfMatch)
	}

	if v.TransactionID != "" {
		r.AddHeader(HeaderTransaction, v.TransactionID)
	}

	return nil
}

func (v *edgeCollection) DeleteEdge(ctx context.Context, key string, opts *DeleteEdgeOptions) (EdgeDeleteResponse, error) {
	url := v.url(key)

	var meta EdgeDeleteResponse
	if opts != nil {
		meta.Old = opts.OldObject
	}

	resp, err := connection.CallDelete(ctx, v.graph.db.connection(), url, &meta, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return EdgeDeleteResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return meta, nil
	default:
		return EdgeDeleteResponse{}, meta.AsArangoErrorWithCode(code)
	}
}

func (c *DeleteEdgeOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*c.WaitForSync))
	}

	if c.OldObject != nil {
		r.AddQuery(QueryReturnOld, "true")
	}

	if c.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, c.IfMatch)
	}

	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}

	return nil
}
