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

func newVertexCollection(vertex *graph, vertexColName string) *vertexCollection {
	return &vertexCollection{
		graph:         vertex,
		vertexColName: vertexColName,
	}
}

var _ VertexCollection = &vertexCollection{}

type vertexCollection struct {
	vertexColName string

	modifiers []connection.RequestModifier

	graph *graph
}

// creates the relative path to this vertex (`_db/<db-name>/_api/gharial/<graph-name>/vertex/<collection-name>`)
func (v *vertexCollection) url(parts ...string) string {
	p := append([]string{v.vertexColName, "vertex"}, parts...)
	return v.graph.url(p...)
}

func (v *vertexCollection) Name() string {
	return v.vertexColName
}

func (v *vertexCollection) GetVertex(ctx context.Context, key string, result interface{}, opts *GetVertexOptions) (DocumentMeta, error) {
	url := v.url(key)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		DocumentMeta          `json:",inline"`
	}

	data := newUnmarshalInto(result)

	resp, err := connection.CallGet(ctx, v.graph.db.connection(), url,
		newMultiUnmarshaller(&response, data), append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return DocumentMeta{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.DocumentMeta, nil
	default:
		return DocumentMeta{}, response.AsArangoErrorWithCode(code)
	}
}

func (g *GetVertexOptions) modifyRequest(r connection.Request) error {
	if g == nil {
		return nil
	}

	if g.Rev != "" {
		r.AddQuery("rev", g.Rev)
	}

	return nil
}

func (v *vertexCollection) CreateVertex(ctx context.Context, vertex interface{}, opts *CreateVertexOptions) (VertexCreateResponse, error) {
	url := v.url()

	var meta VertexCreateResponse

	if opts != nil {
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:",inline"`
		*shared.ResponseStruct `json:",inline"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,

		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPost(ctx, v.graph.db.connection(), url, &response, vertex, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return VertexCreateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return VertexCreateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *CreateVertexOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WaitForSync))
	}

	if c.NewObject != nil {
		r.AddQuery("returnNew", "true")
	}

	return nil
}

func (v *vertexCollection) UpdateVertex(ctx context.Context, key string, newValue interface{}, opts *VertexUpdateOptions) (VertexUpdateResponse, error) {
	url := v.url(key)

	var meta VertexUpdateResponse

	if opts != nil {
		meta.Old = opts.OldObject
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:",inline"`
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,

		Old: newUnmarshalInto(meta.Old),
		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPatch(ctx, v.graph.db.connection(), url, &response, newValue, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return VertexUpdateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return VertexUpdateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (v *VertexUpdateOptions) modifyRequest(r connection.Request) error {
	if v == nil {
		return nil
	}

	if v.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*v.WaitForSync))
	}

	if v.NewObject != nil {
		r.AddQuery("returnNew", "true")
	}

	if v.OldObject != nil {
		r.AddQuery("returnOld", "true")
	}

	if v.KeepNull != nil {
		r.AddQuery("keepNull", boolToString(*v.KeepNull))
	}

	return nil
}

func (v *vertexCollection) ReplaceVertex(ctx context.Context, key string, newValue interface{}, opts *VertexReplaceOptions) (VertexReplaceResponse, error) {
	url := v.url(key)

	var meta VertexReplaceResponse

	if opts != nil {
		meta.Old = opts.OldObject
		meta.New = opts.NewObject
	}

	response := struct {
		*DocumentMeta          `json:",inline"`
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,

		Old: newUnmarshalInto(meta.Old),
		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPatch(ctx, v.graph.db.connection(), url, &response, newValue, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return VertexReplaceResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return VertexReplaceResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (v *VertexReplaceOptions) modifyRequest(r connection.Request) error {
	if v == nil {
		return nil
	}

	if v.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*v.WaitForSync))
	}

	if v.NewObject != nil {
		r.AddQuery("returnNew", "true")
	}

	if v.OldObject != nil {
		r.AddQuery("returnOld", "true")
	}

	if v.KeepNull != nil {
		r.AddQuery("keepNull", boolToString(*v.KeepNull))
	}

	return nil
}

func (v *vertexCollection) DeleteVertex(ctx context.Context, key string, opts *DeleteVertexOptions) (VertexDeleteResponse, error) {
	url := v.url(key)

	var meta VertexDeleteResponse
	if opts != nil {
		meta.Old = opts.OldObject
	}

	resp, err := connection.CallDelete(ctx, v.graph.db.connection(), url, &meta, append(v.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return VertexDeleteResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return meta, nil
	default:
		return VertexDeleteResponse{}, meta.AsArangoErrorWithCode(code)
	}
}

func (c *DeleteVertexOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WaitForSync))
	}

	if c.OldObject != nil {
		r.AddQuery("returnOld", "true")
	}

	return nil
}
