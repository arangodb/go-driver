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

func ptr[T any](v T) *T { return &v }

func TestVectorNListsJSON(t *testing.T) {
	t.Run("marshal and unmarshal fixed count", func(t *testing.T) {
		n := NewVectorNLists(100)
		data, err := json.Marshal(n)
		require.NoError(t, err)
		require.JSONEq(t, "100", string(data))

		var got VectorNLists
		require.NoError(t, json.Unmarshal(data, &got))
		require.NotNil(t, got.Fixed)
		require.Equal(t, 100, *got.Fixed)
		require.Nil(t, got.Scaling)
	})

	t.Run("marshal and unmarshal scaling object", func(t *testing.T) {
		n := NewVectorNListsScaling(VectorNListsScaling{
			Strategy:   ptr(VectorNListsStrategyAutoSqrt),
			Multiplier: ptr(4),
			MinNLists:  ptr(2),
			Tiers: []VectorNListsTier{
				{Threshold: ptr(1000000), FixedValue: ptr(16384)},
			},
		})
		data, err := json.Marshal(n)
		require.NoError(t, err)

		var got VectorNLists
		require.NoError(t, json.Unmarshal(data, &got))
		require.Nil(t, got.Fixed)
		require.NotNil(t, got.Scaling)
		require.NotNil(t, got.Scaling.Strategy)
		require.Equal(t, VectorNListsStrategyAutoSqrt, *got.Scaling.Strategy)
		require.NotNil(t, got.Scaling.Multiplier)
		require.Equal(t, 4, *got.Scaling.Multiplier)
		require.NotNil(t, got.Scaling.MinNLists)
		require.Equal(t, 2, *got.Scaling.MinNLists)
		require.Len(t, got.Scaling.Tiers, 1)
		require.NotNil(t, got.Scaling.Tiers[0].Threshold)
		require.Equal(t, 1000000, *got.Scaling.Tiers[0].Threshold)
		require.NotNil(t, got.Scaling.Tiers[0].FixedValue)
		require.Equal(t, 16384, *got.Scaling.Tiers[0].FixedValue)
	})

	t.Run("unmarshal null clears existing value", func(t *testing.T) {
		tests := map[string]VectorNLists{
			"fixed":   *NewVectorNLists(100),
			"scaling": *NewVectorNListsScaling(VectorNListsScaling{}),
		}
		for name, got := range tests {
			t.Run(name, func(t *testing.T) {
				require.NoError(t, json.Unmarshal([]byte("null"), &got))
				require.Nil(t, got.Fixed)
				require.Nil(t, got.Scaling)
			})
		}
	})
}

func TestVectorParamsValidate(t *testing.T) {
	metric := VectorMetricCosine
	dim := 3

	t.Run("omitted nLists is valid", func(t *testing.T) {
		p := &VectorParams{Dimension: &dim, Metric: &metric}
		require.NoError(t, p.validate())
	})

	t.Run("fixed nLists must be positive", func(t *testing.T) {
		p := &VectorParams{Dimension: &dim, Metric: &metric, NLists: NewVectorNLists(0)}
		require.Error(t, p.validate())
	})

	t.Run("scaling nLists requires autoSqrt and bounds", func(t *testing.T) {
		p := &VectorParams{
			Dimension: &dim,
			Metric:    &metric,
			NLists: NewVectorNListsScaling(VectorNListsScaling{
				Strategy:   ptr(VectorNListsStrategy("nope")),
				Multiplier: ptr(4),
				MinNLists:  ptr(2),
			}),
		}
		require.Error(t, p.validate())

		p.NLists = NewVectorNListsScaling(VectorNListsScaling{
			Strategy:   ptr(VectorNListsStrategyAutoSqrt),
			Multiplier: ptr(0),
			MinNLists:  ptr(2),
		})
		require.Error(t, p.validate())

		p.NLists = NewVectorNListsScaling(VectorNListsScaling{
			Strategy:   ptr(VectorNListsStrategyAutoSqrt),
			Multiplier: ptr(4),
			MinNLists:  ptr(2),
		})
		require.NoError(t, p.validate())
	})

	t.Run("numberOfDocsPerCentroid must be >= 1 when set", func(t *testing.T) {
		zero := 0
		p := &VectorParams{Dimension: &dim, Metric: &metric, NumberOfDocsPerCentroid: &zero}
		require.Error(t, p.validate())
		ok := 100
		p.NumberOfDocsPerCentroid = &ok
		require.NoError(t, p.validate())
	})
}

func TestIndexResponseVectorUnmarshal(t *testing.T) {
	raw := []byte(`{
		"id": "coll/68",
		"name": "vector_l2",
		"type": "vector",
		"fields": ["embedding"],
		"storedValues": ["text"],
		"trainingState": "ready",
		"errorMessage": "",
		"params": {
			"metric": "l2",
			"dimension": 544,
			"nLists": {"strategy": "autoSqrt", "multiplier": 4, "minNLists": 2},
			"numberOfDocsPerCentroid": 100,
			"factory": "IVF{},Flat"
		},
		"shards": {
			"s10042": {"trainingState": "ready", "error": "", "resolvedNLists": 400}
		}
	}`)

	var idx IndexResponse
	require.NoError(t, json.Unmarshal(raw, &idx))
	require.Equal(t, VectorIndexType, idx.Type)
	require.Equal(t, []string{"embedding"}, idx.VectorFields)
	require.Equal(t, []string{"text"}, idx.VectorStoredValues)
	require.NotNil(t, idx.TrainingState)
	require.Equal(t, VectorIndexTrainingStateReady, *idx.TrainingState)
	require.NotNil(t, idx.VectorIndex)
	require.NotNil(t, idx.VectorIndex.NLists)
	require.NotNil(t, idx.VectorIndex.NLists.Scaling)
	require.NotNil(t, idx.VectorIndex.NLists.Scaling.Strategy)
	require.Equal(t, VectorNListsStrategyAutoSqrt, *idx.VectorIndex.NLists.Scaling.Strategy)
	require.NotNil(t, idx.VectorIndex.NumberOfDocsPerCentroid)
	require.Equal(t, 100, *idx.VectorIndex.NumberOfDocsPerCentroid)
	require.NotNil(t, idx.VectorIndex.Factory)
	require.Equal(t, "IVF{},Flat", *idx.VectorIndex.Factory)
	require.Contains(t, idx.VectorShards, "s10042")
	require.NotNil(t, idx.VectorShards["s10042"].ResolvedNLists)
	require.Equal(t, 400, *idx.VectorShards["s10042"].ResolvedNLists)
}
