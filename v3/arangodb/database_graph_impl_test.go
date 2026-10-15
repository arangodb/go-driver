//
// DISCLAIMER
//
// Copyright 2023-2026 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCreateGraphBodyPlacesSmartFlagsUnderOptions(t *testing.T) {
	shards := 9
	body := newCreateGraphBody("disjointSmartGraph", &GraphDefinition{
		EdgeDefinitions: []EdgeDefinition{{
			Collection: "edges",
			From:       []string{"startNodes"},
			To:         []string{"endNodes"},
		}},
		OrphanCollections:   []string{"orphanNodes"},
		IsSmart:             true,
		IsDisjoint:          true,
		SmartGraphAttribute: "region",
		NumberOfShards:      &shards,
	}, &CreateGraphOptions{Satellites: []string{"endNodes"}})

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	_, hasTopLevelSmart := decoded["isSmart"]
	_, hasTopLevelDisjoint := decoded["isDisjoint"]
	require.False(t, hasTopLevelSmart)
	require.False(t, hasTopLevelDisjoint)
	require.Equal(t, "disjointSmartGraph", decoded["name"])

	options, ok := decoded["options"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, options["isSmart"])
	require.Equal(t, true, options["isDisjoint"])
	require.Equal(t, "region", options["smartGraphAttribute"])
	require.Equal(t, []any{"endNodes"}, options["satellites"])
}

func TestNewCreateGraphBodyOmitsDefaultSmartFlags(t *testing.T) {
	body := newCreateGraphBody("myGraph", &GraphDefinition{
		EdgeDefinitions: []EdgeDefinition{{
			Collection: "edges",
			From:       []string{"startNodes"},
			To:         []string{"endNodes"},
		}},
	}, nil)

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	options, ok := decoded["options"].(map[string]any)
	require.True(t, ok)
	_, hasSmart := options["isSmart"]
	_, hasDisjoint := options["isDisjoint"]
	require.False(t, hasSmart)
	require.False(t, hasDisjoint)
}
