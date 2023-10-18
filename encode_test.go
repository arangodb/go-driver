//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

package driver_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/vst"
)

func TestPathEscape(t *testing.T) {
	t.Run("pathUnescape - HTTP", func(t *testing.T) {
		tests := map[string]string{
			"abc":        "abc",
			"The Donald": "The%20Donald",
		}
		for input, expected := range tests {
			result := driver.PathEscape(input, prepareHTTPConnection())
			require.Equal(t, expected, result)
		}
	})
	t.Run("pathUnescape - VST", func(t *testing.T) {
		tests := map[string]string{
			"abc":        "abc",
			"The Donald": "The Donald",
		}
		for input, expected := range tests {
			result := driver.PathEscape(input, prepareVSTConnection())
			require.Equal(t, expected, result)
		}

	})
}

func prepareHTTPConnection() driver.Connection {
	config := http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	}
	conn, _ := http.NewConnection(config)
	return conn
}

func prepareVSTConnection() driver.Connection {
	config := vst.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	}
	conn, _ := vst.NewConnection(config)
	return conn
}
