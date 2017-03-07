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

package test

import (
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestCreateEdge creates an edge and then checks that it exists.
func TestCreateEdge(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "edge_test", nil, t)
	g := ensureGraph(nil, db, "create_edge_test", nil, t)
	ec := ensureEdgeCollection(nil, g, "citiesPerState", []string{"city"}, []string{"state"}, t)
	cities, err := db.Collection(nil, "city")
	assertOK(err, t)
	states, err := db.Collection(nil, "state")
	assertOK(err, t)
	from, err := cities.CreateDocument(nil, map[string]interface{}{"name": "Venlo"})
	assertOK(err, t)
	to, err := states.CreateDocument(nil, map[string]interface{}{"name": "Limburg"})
	assertOK(err, t)
	meta, err := ec.CreateDocument(nil, driver.EdgeDocument{From: from.ID, To: to.ID})
	if err != nil {
		t.Fatalf("Failed to create new edge: %s", describe(err))
	}
	// Document must exists now
	if _, err := ec.ReadDocument(nil, meta.Key, nil); err != nil {
		t.Fatalf("Failed to read edge '%s': %s", meta.Key, describe(err))
	}
}
