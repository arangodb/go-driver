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

package tests

import (
	"github.com/arangodb/go-driver/v2/arangodb"
)

func sampleGraph() *arangodb.GraphDefinition {
	return &arangodb.GraphDefinition{
		NumberOfShards:      newInt(3),
		SmartGraphAttribute: "key",
		IsSmart:             true,
	}
}

func sampleGraphWithEdges(db arangodb.Database) (*arangodb.GraphDefinition, []string) {
	edge := db.Name() + "_edge"
	to := db.Name() + "_to-coll"
	from := db.Name() + "_from-coll"

	g := sampleGraph()
	g.EdgeDefinitions = []arangodb.EdgeDefinition{
		{
			Collection: edge,
			To:         []string{to},
			From:       []string{from},
		},
	}
	g.OrphanCollections = []string{"orphan1", "orphan2"}

	return g, []string{to, from}
}
