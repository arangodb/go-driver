//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package agency_test

import (
	"testing"

	"github.com/arangodb/go-driver/agency"
	"github.com/stretchr/testify/require"
)

func TestCreateSubKey(t *testing.T) {
	testCases := []struct {
		name     string
		elements []string
		key      agency.Key
	}{
		{
			name:     "Create a new key based on not empty key with not empty elements",
			key:      agency.Key{"level1", "level2"},
			elements: []string{"level3"},
		},
		{
			name: "Create a new key based on not empty key with empty elements",
			key:  agency.Key{"level1", "level2"},
		},
		{
			name:     "Create a new key based on empty key",
			elements: []string{"level3"},
		},
		{
			name: "Create a new key based on empty key with empty elements",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newKey := testCase.key.CreateSubKey(testCase.elements...)

			require.Len(t, newKey, len(testCase.key)+len(testCase.elements))
			if len(testCase.key) > 0 && &testCase.key[0] == &newKey[0] {
				require.Fail(t, "New key should have always different address")
			}

			for i, s := range testCase.key {
				require.Equal(t, s, newKey[i])
			}
			for i, s := range testCase.elements {
				require.Equal(t, s, newKey[i+len(testCase.key)])
			}
		})
	}

}
