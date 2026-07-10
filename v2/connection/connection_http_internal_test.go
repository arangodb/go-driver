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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewHttpConnection_RedirectConfiguration(t *testing.T) {
	eps := NewRoundRobinEndpoints([]string{"https://a:8529"})

	t.Run("default redirect behavior", func(t *testing.T) {
		conn := NewHttpConnection(HttpConfiguration{Endpoint: eps})
		hc, ok := conn.(*httpConnection)
		require.True(t, ok)
		require.Nil(t, hc.client.CheckRedirect)
	})

	t.Run("dont follow redirect", func(t *testing.T) {
		conn := NewHttpConnection(HttpConfiguration{
			Endpoint:           eps,
			DontFollowRedirect: true,
		})
		hc, ok := conn.(*httpConnection)
		require.True(t, ok)
		require.NotNil(t, hc.client.CheckRedirect)

		err := hc.client.CheckRedirect(&http.Request{}, nil)
		require.ErrorIs(t, err, http.ErrUseLastResponse)
	})
}

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

// Test_httpConnection_Do_NonJSONContentType documents Michele's Category D finding:
// unknown Content-Types (e.g. text/html) fall back to the JSON decoder.
// Depending on whether the status is in allowedStatusCodes, callers may see either a
// JSON parse error or an ArangoError built after a failed (ignored) HTML decode.
func Test_httpConnection_Do_NonJSONContentType(t *testing.T) {
	type versionOut struct {
		Server  string `json:"server"`
		Version string `json:"version"`
	}

	t.Run("text/html 502 with no status filter returns JSON parse error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "<html><body>502 Bad Gateway</body></html>")
		}))
		t.Cleanup(srv.Close)
		t.Logf("srv.URL: %s", srv.URL)
		conn := NewHttpConnection(HttpConfiguration{
			Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
			ContentType: ApplicationJSON,
		})
		req, err := conn.NewRequest(http.MethodGet, "_api/version")
		require.NoError(t, err)

		var out versionOut
		resp, err := conn.Do(context.Background(), req, &out)
		require.Error(t, err)
		t.Logf("error: %v, error message: %s, response code: %d, response content: %s", err, err.Error(), resp.Code(), resp.Content())
		require.Contains(t, err.Error(), "invalid character '<'")
		require.NotNil(t, resp)
		require.Equal(t, http.StatusBadGateway, resp.Code())
		require.Equal(t, "text/html", resp.Content())
		require.Empty(t, out)

		var syntax *json.SyntaxError
		require.ErrorAs(t, err, &syntax)
	})

	t.Run("text/html 502 with allowed 200 swallows decode error and returns ArangoError(502)", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "<html><body>502 Bad Gateway</body></html>")
		}))
		t.Cleanup(srv.Close)

		conn := NewHttpConnection(HttpConfiguration{
			Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
			ContentType: ApplicationJSON,
		})
		req, err := conn.NewRequest(http.MethodGet, "_api/version")
		require.NoError(t, err)

		var out versionOut
		resp, err := conn.Do(context.Background(), req, &out, http.StatusOK)
		require.Error(t, err)
		t.Logf("error: %v, error message: %s, response code: %d, response content: %s", err, err.Error(), resp.Code(), resp.Content())
		// Decode of HTML into shared.Response fails silently (_ = Decode...),
		// then AsArangoErrorWithCode(502) is returned — not the JSON syntax error.
		require.NotContains(t, err.Error(), "invalid character")
		require.NotNil(t, resp)
		require.Equal(t, http.StatusBadGateway, resp.Code())
		require.Equal(t, "text/html", resp.Content())
		t.Logf("error with allowedStatusCodes=[200]: %v", err)
	})

	t.Run("text/plain success uses bytes decoder (not JSON)", func(t *testing.T) {
		body := "not-json-plain-text"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", PlainText)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, body)
		}))
		t.Cleanup(srv.Close)

		conn := NewHttpConnection(HttpConfiguration{
			Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
			ContentType: ApplicationJSON,
		})
		req, err := conn.NewRequest(http.MethodGet, "_api/version")
		require.NoError(t, err)

		var out []byte
		resp, err := conn.Do(context.Background(), req, &out, http.StatusOK)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.Code())
		require.Equal(t, PlainText, resp.Content())
		require.Equal(t, []byte(body), out)
	})

	t.Run("text/plain HTML-looking body still succeeds into []byte", func(t *testing.T) {
		body := "<html>oops</html>"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", PlainText)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, body)
		}))
		t.Cleanup(srv.Close)

		conn := NewHttpConnection(HttpConfiguration{
			Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
			ContentType: ApplicationJSON,
		})
		req, err := conn.NewRequest(http.MethodGet, "_api/version")
		require.NoError(t, err)

		var out []byte
		resp, err := conn.Do(context.Background(), req, &out)
		require.NoError(t, err)
		require.Equal(t, PlainText, resp.Content())
		require.Equal(t, []byte(body), out)
	})
}

func Test_httpConnection_Decoder_textHTMLFallsBackToJSON(t *testing.T) {
	conn := httpConnection{contentType: ApplicationJSON}

	// text/html is unknown → falls back through connection content-type → JSON decoder.
	require.Equal(t, getJsonDecoder(), conn.Decoder("text/html"))
	require.Equal(t, getJsonDecoder(), conn.Decoder("text/html; charset=UTF-8"))

	// Known non-JSON types do not fall back.
	require.Equal(t, getBytesDecoder(), conn.Decoder(PlainText))
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
