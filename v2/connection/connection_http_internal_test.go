//
// DISCLAIMER
//
// Copyright 2021-2023 ArangoDB GmbH, Cologne, Germany
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

package connection

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_httpConnection_Decoder(t *testing.T) {
	tests := map[string]struct {
		contentType string
		conn        httpConnection
		wantDecoder Decoder
	}{
		"JSON response decoder": {
			contentType: ApplicationJSON,
			wantDecoder: getJsonDecoder(),
		},
		"Bytes response decoder": {
			contentType: PlainText,
			wantDecoder: getBytesDecoder(),
		},
		"JSON HTTP connection decoder": {
			conn: httpConnection{
				contentType: ApplicationJSON,
			},
			wantDecoder: getJsonDecoder(),
		},
		"Bytes HTTP connection decoder": {
			conn: httpConnection{
				contentType: PlainText,
			},
			wantDecoder: getBytesDecoder(),
		},
		"default decoder": {
			wantDecoder: getJsonDecoder(),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			decoder := test.conn.Decoder(test.contentType)

			require.NotNil(t, decoder)
			assert.Equal(t, test.wantDecoder, decoder)
		})
	}
}

func Test_httpConnection_NewRequest(t *testing.T) {
	eps := []string{
		"https://a:8529", "https://a:8539", "https://b:8529",
	}

	c := httpConnection{
		endpoint: NewRoundRobinEndpoints(eps),
	}

	j := 0
	for i := 0; i < 10; i++ {
		expectedEp := eps[j]
		req, err := c.NewRequest(http.MethodGet, "_api/version")
		require.NoError(t, err)
		require.Equal(t, expectedEp, req.Endpoint())
		require.True(t, strings.HasPrefix(req.URL(), expectedEp))
		j++
		if j >= len(eps) {
			j = 0
		}
	}
}

func Test_httpConnection_NewRequestWithEndpoint(t *testing.T) {
	c := httpConnection{
		endpoint: NewRoundRobinEndpoints([]string{"https://a:8529", "https://a:8539", "https://b:8529"}),
	}

	for i := 0; i < 10; i++ {
		ep := "https://a:8539"
		req, err := c.NewRequestWithEndpoint(ep, http.MethodGet, "_api/version")
		require.NoError(t, err)
		require.Equal(t, ep, req.Endpoint())
		require.True(t, strings.HasPrefix(req.URL(), ep))
	}
}
