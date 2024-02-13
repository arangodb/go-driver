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

func newGraphEdgeDefinitions(graph *graph) *graphEdgeDefinitions {
	return &graphEdgeDefinitions{
		graph: graph,
	}
}

var _ GraphEdgesDefinition = &graphEdgeDefinitions{}

type graphEdgeDefinitions struct {
	graph *graph
}

// creates the relative path to this Edge (`_db/<db-name>/_api/gharial/<graph-name>/edge`)
func (g *graphEdgeDefinitions) url(parts ...string) string {
	p := append([]string{"edge"}, parts...)
	return g.graph.url(p...)
}

func (g *graphEdgeDefinitions) EdgeDefinition(ctx context.Context, collection string) (Edge, error) {
	definitions, err := g.getEdgeDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	for _, n := range definitions {
		if n == collection {
			return newEdgeCollection(g.graph, collection), nil
		}
	}

	err = shared.ArangoError{
		HasError:     true,
		Code:         http.StatusNotFound,
		ErrorNum:     0,
		ErrorMessage: fmt.Sprintf("Edge definition with name: '%s' not found.", collection),
	}
	return nil, errors.WithStack(err)
}

func (g *graphEdgeDefinitions) EdgeDefinitionExists(ctx context.Context, collection string) (bool, error) {
	definitions, err := g.getEdgeDefinitions(ctx)
	if err != nil {
		return false, err
	}
	for _, n := range definitions {
		if n == collection {
			return true, nil
		}
	}
	return false, nil
}

func (g *graphEdgeDefinitions) GetEdgeDefinitions(ctx context.Context) ([]Edge, error) {
	definitions, err := g.getEdgeDefinitions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Edge, len(definitions))
	for id, name := range definitions {
		result[id] = newEdgeCollection(g.graph, name)
	}
	return result, nil
}

func (g *graphEdgeDefinitions) getEdgeDefinitions(ctx context.Context) ([]string, error) {
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

func (g *graphEdgeDefinitions) CreateEdgeDefinition(ctx context.Context, collection string, from, to []string, opts *CreateEdgeDefinitionOptions) (CreateEdgeDefinitionResponse, error) {
	url := g.url()

	var response CreateEdgeDefinitionResponse

	reqData := struct {
		Collection                  string   `json:"collection,omitempty"`
		To                          []string `json:"to,omitempty"`
		From                        []string `json:"from,omitempty"`
		CreateEdgeDefinitionOptions `json:"options,omitempty"`
	}{
		Collection: collection,
		To:         to,
		From:       from,
	}

	if opts != nil {
		reqData.CreateEdgeDefinitionOptions = *opts
	}

	resp, err := connection.CallPost(ctx, g.graph.db.connection(), url, &response, reqData, g.graph.db.modifiers...)
	if err != nil {
		return CreateEdgeDefinitionResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		response.Edge = newEdgeCollection(g.graph, collection)
		return response, nil
	default:
		return CreateEdgeDefinitionResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (g *graphEdgeDefinitions) ReplaceEdgeDefinition(ctx context.Context, collection string, from, to []string, opts *ReplaceEdgeOptions) (ReplaceEdgeDefinitionResponse, error) {
	url := g.url(collection)

	var response ReplaceEdgeDefinitionResponse

	reqData := struct {
		Collection         string   `json:"collection,omitempty"`
		To                 []string `json:"to,omitempty"`
		From               []string `json:"from,omitempty"`
		ReplaceEdgeOptions `json:"options,omitempty"`
	}{
		Collection: collection,
		To:         to,
		From:       from,
	}

	if opts != nil && opts.Satellites != nil {
		reqData.Satellites = opts.Satellites
	}

	resp, err := connection.CallPut(ctx, g.graph.db.connection(), url, &response, reqData, append(g.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return ReplaceEdgeDefinitionResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		response.Edge = newEdgeCollection(g.graph, collection)
		return response, nil
	default:
		return ReplaceEdgeDefinitionResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *ReplaceEdgeOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.DropCollection != nil {
		r.AddQuery("dropCollection", boolToString(*c.DropCollection))
	}

	if c.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WaitForSync))
	}

	return nil
}

func (g *graphEdgeDefinitions) DeleteEdgeDefinition(ctx context.Context, collection string, opts *DeleteEdgeDefinitionOptions) (DeleteEdgeDefinitionResponse, error) {
	url := g.url(collection)

	var response DeleteEdgeDefinitionResponse

	resp, err := connection.CallDelete(ctx, g.graph.db.connection(), url, &response, append(g.graph.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return DeleteEdgeDefinitionResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return response, nil
	default:
		return DeleteEdgeDefinitionResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (d *DeleteEdgeDefinitionOptions) modifyRequest(r connection.Request) error {
	if d == nil {
		return nil
	}

	if d.DropCollection != nil {
		r.AddQuery("dropCollections", boolToString(*d.DropCollection))
	}

	if d.WaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*d.WaitForSync))
	}

	return nil
}
