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

package connection

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RequestDBNameValueExtractor(t *testing.T) {
	testCases := []struct {
		method      string
		path        string
		expectedVal string
	}{
		{"GET", "/_db/mydb_a/info", "mydb_a"},
		{"GET", "/_db/mydb_b/info", "mydb_b"},
		{"GET", "/_db/mydb_c/info", "mydb_c"},
		{"GET", "/_db/mydb_c/_api/info", "mydb_c"},
		{"GET", "/_db/somedb", "GET_/_db/somedb"},
		{"GET", "/version", "GET_/version"},
		{"GET", "/_db", "GET_/_db"},
		{"GET", "/part1/part2/part3", "GET_/part1/part2/part3"},
	}
	for i, tc := range testCases {
		val, err := RequestDBNameValueExtractor(tc.method, tc.path)
		require.NoError(t, err, i)
		require.Equal(t, tc.expectedVal, val, i)
	}
}

func Test_maglevHashEndpoints_New(t *testing.T) {
	testCases := [][]string{
		{"a", "b", "c"},                              // len is prime
		{"a", "b", "c", "d"},                         // len is not prime
		{"a", "b", "c", "d", "e"},                    // len is prime
		{"a", "b", "c", "d", "a2", "b2", "c2", "d2"}, // len is not prime
	}

	for i, tc := range testCases {
		e, err := NewMaglevHashEndpoints(tc, RequestDBNameValueExtractor)
		require.NoError(t, err, i)
		require.NotNil(t, e, i)
	}
}

func Test_maglevHashEndpoints_Get(t *testing.T) {
	eps := []string{"a", "b", "c"}
	maglevEndpoints, err := NewMaglevHashEndpoints(eps, RequestDBNameValueExtractor)
	require.NoError(t, err)

	testCases := []struct {
		method        string
		path          string
		expectedIndex int
	}{
		{"GET", "/_db/mydb_a/info", 0},
		{"POST", "/_db/mydb_a/info", 0},
		{"POST", "/_db/mydb_a/_api/indexes", 0},
		{"POST", "/_db/mydb_a/_api/views", 0},
		{"GET", "/_db/mydb_b/info", 0},
		{"POST", "/_db/mydb_b/info", 0},
		{"POST", "/_db/mydb_b/_api/indexes", 0},
		{"POST", "/_db/mydb_b/_api/views", 0},
		{"GET", "/_db/mydb_c/info", 2},
		{"POST", "/_db/mydb_c/info", 2},
		{"POST", "/_db/mydb_c/_api/indexes", 2},
		{"POST", "/_db/mydb_c/_api/views", 2},
	}
	for i := 0; i < 1; i++ {
		// Try three times to ensure requests order does not affect the result
		rand.Shuffle(len(testCases), func(i, j int) {
			temp := testCases[j]
			testCases[j] = testCases[i]
			testCases[i] = temp
		})
		for j, tc := range testCases {
			t.Run(fmt.Sprintf("iter_%d_tc_%d", i, j), func(t *testing.T) {
				ep, err := maglevEndpoints.Get("", tc.method, tc.path)
				require.NoError(t, err)
				epIndex := -1
				for index, e := range eps {
					if e == ep {
						epIndex = index
						break
					}
				}
				require.Equal(t, tc.expectedIndex, epIndex, "path %s. expected %s, got %s", tc.path, eps[tc.expectedIndex], ep)
			})
		}
	}
}
