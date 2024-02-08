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
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newGraphVertexCollections(graph *graph) *graphVertexCollections {
	return &graphVertexCollections{
		graph: graph,
	}
}

var _ GraphVertexCollections = &graphVertexCollections{}

type graphVertexCollections struct {
	graph *graph
}

// creates the relative path to this vertex (`_db/<db-name>/_api/gharial/<graph-name>/vertex`)
func (g *graphVertexCollections) url(parts ...string) string {
	p := append([]string{"vertex"}, parts...)
	return g.graph.url(p...)
}

func (g *graphVertexCollections) VertexCollection(ctx context.Context, name string) (VertexCollection, error) {
	collections, err := g.getCollections(ctx)
	if err != nil {
		return nil, err
	}
	for _, n := range collections {
		if n == name {
			return newVertexCollection(g.graph, name), nil
		}
	}

	err = shared.ArangoError{
		HasError:     true,
		Code:         http.StatusNotFound,
		ErrorNum:     0,
		ErrorMessage: fmt.Sprintf("Vertex colletion with name: '%s' not found.", name),
	}
	return nil, errors.WithStack(err)
}

func (g *graphVertexCollections) VertexCollectionExists(ctx context.Context, name string) (bool, error) {
	collections, err := g.getCollections(ctx)
	if err != nil {
		return false, err
	}
	for _, n := range collections {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}

func (g *graphVertexCollections) VertexCollections(ctx context.Context) ([]VertexCollection, error) {
	collections, err := g.getCollections(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]VertexCollection, len(collections))
	for id, name := range collections {
		result[id] = newVertexCollection(g.graph, name)
	}
	return result, nil
}

func (g *graphVertexCollections) getCollections(ctx context.Context) ([]string, error) {
	url := g.url()

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Collections           []string `json:"collections,omitempty"`
	}
	resp, err := connection.CallGet(ctx, g.graph.db.connection(), url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Collections, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (g *graphVertexCollections) CreateVertexCollection(ctx context.Context, name string, opts *CreateVertexCollectionOptions) (CreateVertexCollectionResponse, error) {
	url := g.url()

	var response CreateVertexCollectionResponse

	reqData := struct {
		Collection                    string `json:"collection,omitempty"`
		CreateVertexCollectionOptions `json:"options,omitempty"`
	}{
		Collection: name,
	}

	if opts != nil {
		reqData.CreateVertexCollectionOptions = *opts
	}

	resp, err := connection.CallPost(ctx, g.graph.db.connection(), url, &response, reqData, g.graph.db.modifiers...)
	if err != nil {
		return CreateVertexCollectionResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		response.VertexCollection = newVertexCollection(g.graph, name)
		return response, nil
	default:
		return CreateVertexCollectionResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (g *graphVertexCollections) DeleteVertexCollection(ctx context.Context, name string, opts *DeleteVertexCollectionOptions) (DeleteVertexCollectionResponse, error) {
	url := g.url(name)

	var response DeleteVertexCollectionResponse

	resp, err := connection.CallDelete(ctx, g.graph.db.connection(), url, &response, append(g.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return DeleteVertexCollectionResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return response, nil
	default:
		return DeleteVertexCollectionResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *DeleteVertexCollectionOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.DropCollection != nil {
		r.AddQuery("dropCollection", boolToString(*c.DropCollection))
	}

	return nil
}
