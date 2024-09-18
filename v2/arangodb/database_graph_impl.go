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
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newDatabaseGraph(db *database) *databaseGraph {
	return &databaseGraph{
		db: db,
	}
}

var _ DatabaseGraph = &databaseGraph{}

type databaseGraph struct {
	db *database
}

func (d *databaseGraph) GetEdges(ctx context.Context, name, vertex string, options *GetEdgesOptions) ([]EdgeDetails, error) {
	if name == "" || vertex == "" {
		return nil, errors.WithStack(errors.New("edge collection name and vertex must be set"))
	}

	urlEndpoint := d.db.url("_api", "edges", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Edges                 []EdgeDetails `json:"edges,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response, append(d.db.modifiers, options.modifyRequest, connection.WithQuery("vertex", vertex))...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Edges, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d *databaseGraph) Graph(ctx context.Context, name string, options *GetGraphOptions) (Graph, error) {
	urlEndpoint := d.db.url("_api", "gharial", url.PathEscape(name))

	if options != nil && options.SkipExistCheck {
		return nil, nil
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
		GraphDefinition       `json:"graph,omitempty"`
	}
	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newGraph(d.db, response.GraphDefinition), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d *databaseGraph) GraphExists(ctx context.Context, name string) (bool, error) {
	urlEndpoint := d.db.url("_api", "gharial", url.PathEscape(name))

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return false, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return true, nil
	default:
		err = response.AsArangoErrorWithCode(code)
		if shared.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
}

func (d *databaseGraph) Graphs(ctx context.Context) (GraphsResponseReader, error) {
	urlEndpoint := d.db.url("_api", "gharial")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Graphs                connection.Array `json:"graphs,omitempty"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return newGraphsResponseReader(d.db, &response.Graphs), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

type createGraphOptions struct {
	Name              string                        `json:"name"`
	EdgeDefinitions   []EdgeDefinition              `json:"edgeDefinitions,omitempty"`
	IsSmart           bool                          `json:"isSmart,omitempty"`
	OrphanCollections []string                      `json:"orphanCollections,omitempty"`
	Options           *createGraphAdditionalOptions `json:"options,omitempty"`
}

type createGraphAdditionalOptions struct {
	IsDisjoint          bool                   `json:"isDisjoint,omitempty"`
	SmartGraphAttribute string                 `json:"smartGraphAttribute,omitempty"`
	NumberOfShards      *int                   `json:"numberOfShards,omitempty"`
	ReplicationFactor   graphReplicationFactor `json:"replicationFactor,omitempty"`
	WriteConcern        *int                   `json:"writeConcern,omitempty"`
	Satellites          []string               `json:"satellites,omitempty"`
}

func (d *databaseGraph) CreateGraph(ctx context.Context, name string, graph *GraphDefinition, options *CreateGraphOptions) (Graph, error) {
	urlEndpoint := d.db.url("_api", "gharial")

	input := createGraphOptions{
		Name: name,
	}
	if graph != nil {
		input.EdgeDefinitions = graph.EdgeDefinitions
		input.IsSmart = graph.IsSmart
		input.OrphanCollections = graph.OrphanCollections

		input.Options = &createGraphAdditionalOptions{
			IsDisjoint:          graph.IsDisjoint,
			SmartGraphAttribute: graph.SmartGraphAttribute,
			NumberOfShards:      graph.NumberOfShards,
			ReplicationFactor:   graph.ReplicationFactor,
			WriteConcern:        graph.WriteConcern,
		}

		if options != nil {
			input.Options.Satellites = options.Satellites
		}
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
		GraphDefinition       `json:"graph,omitempty"`
	}

	resp, err := connection.CallPost(ctx, d.db.connection(), urlEndpoint, &response, input)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusCreated, http.StatusAccepted:
		return newGraph(d.db, response.GraphDefinition), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func newGraphsResponseReader(db *database, arr *connection.Array) GraphsResponseReader {
	return &graphsResponseReader{
		array: arr,
		db:    db,
	}
}

type graphsResponseReader struct {
	array *connection.Array
	db    *database
}

func (reader *graphsResponseReader) Read() (Graph, error) {
	if !reader.array.More() {
		return nil, shared.NoMoreDocumentsError{}
	}

	graphResponse := GraphDefinition{}

	if err := reader.array.Unmarshal(newUnmarshalInto(&graphResponse)); err != nil {
		if err == io.EOF {
			return nil, shared.NoMoreDocumentsError{}
		}
		return nil, err
	}

	return newGraph(reader.db, graphResponse), nil
}
