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
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newGraph(db *database, def GraphDefinition, modifiers ...connection.RequestModifier) *graph {
	g := &graph{db: db, input: def, modifiers: append(db.modifiers, modifiers...)}

	g.graphVertexCollections = newGraphVertexCollections(g)
	g.graphEdgeDefinitions = newGraphEdgeDefinitions(g)

	return g
}

var _ Graph = &graph{}

type graph struct {
	db *database

	input     GraphDefinition
	modifiers []connection.RequestModifier

	*graphVertexCollections
	*graphEdgeDefinitions
}

// creates the relative path to this graph (`_db/<db-name>/_api/gharial/<graph-name>`)
func (g *graph) url(parts ...string) string {
	p := append([]string{"_api", "gharial", url.PathEscape(g.Name())}, parts...)
	return g.db.url(p...)
}

func (g *graph) Name() string {
	return g.input.Name
}

func (g *graph) EdgeDefinitions() []EdgeDefinition {
	return g.input.EdgeDefinitions
}

func (g *graph) SmartGraphAttribute() string {
	return g.input.SmartGraphAttribute
}

func (g *graph) IsSmart() bool {
	return g.input.IsSmart
}

func (g *graph) IsDisjoint() bool {
	return g.input.IsDisjoint
}

func (g *graph) IsSatellite() bool {
	return g.input.IsSatellite
}

func (g *graph) NumberOfShards() *int {
	return g.input.NumberOfShards
}

func (g *graph) OrphanCollections() []string {
	return g.input.OrphanCollections
}

func (g *graph) ReplicationFactor() int {
	return int(g.input.ReplicationFactor)
}

func (g *graph) WriteConcern() *int {
	return g.input.WriteConcern
}

func (g *graph) Remove(ctx context.Context, opts *RemoveGraphOptions) error {
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, g.db.connection(), g.url(), &response, append(g.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (o *RemoveGraphOptions) modifyRequest(r connection.Request) error {
	if o == nil {
		return nil
	}
	if o.DropCollections {
		r.AddQuery("dropCollections", boolToString(true))
	}
	return nil
}

// graphReplicationFactor wraps the replication factor of a graph.
type graphReplicationFactor int

func (g graphReplicationFactor) MarshalJSON() ([]byte, error) {
	switch g {
	case SatelliteGraph:
		return json.Marshal(replicationFactorSatelliteString)
	default:
		return json.Marshal(int(g))
	}
}

func (g *graphReplicationFactor) UnmarshalJSON(data []byte) error {
	var d int

	if err := json.Unmarshal(data, &d); err == nil {
		*g = graphReplicationFactor(d)
		return nil
	}

	var s string

	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case replicationFactorSatelliteString:
		*g = graphReplicationFactor(SatelliteGraph)
		return nil
	default:
		return errors.Errorf("Unsupported type %s", s)
	}
}
