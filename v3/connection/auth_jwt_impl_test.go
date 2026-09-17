//
// DISCLAIMER
//
// Copyright 2026 ArangoDB GmbH, Cologne, Germany
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type jwtVersionResponse struct {
	Version string `json:"version"`
}

func TestNewJWTAuthWrapper_RefreshesRejectedUnexpiredToken(t *testing.T) {
	var mu sync.Mutex
	authCalls := 0
	accepted := ""
	issue := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_open/auth"):
			mu.Lock()
			authCalls++
			issue++
			token := fakeJWT(fmt.Sprintf("tok-%d", issue), time.Now().Add(time.Hour))
			accepted = token
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"jwt": token}))
		default:
			mu.Lock()
			want := accepted
			mu.Unlock()
			got := r.Header.Get("Authorization")
			if got != "bearer "+want || want == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":true,"errorMessage":"not authorized to execute this request","code":401,"errorNum":401}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"server":"arango","version":"4.0"}`))
		}
	}))
	defer srv.Close()

	conn := NewJWTAuthWrapper("root", "passwd")(NewHttpConnection(HttpConfiguration{
		Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
		ContentType: ApplicationJSON,
	}))

	ctx := context.Background()
	var out jwtVersionResponse

	resp, err := CallGet(ctx, conn, NewUrl("_api", "version"), &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code())
	require.Equal(t, "4.0", out.Version)

	mu.Lock()
	require.Equal(t, 1, authCalls)
	// Simulate JWT secret reload: cached token is still unexpired but no longer accepted.
	accepted = "rotated-secret-rejects-old-jwt"
	mu.Unlock()

	out = jwtVersionResponse{}
	resp, err = CallGet(ctx, conn, NewUrl("_api", "version"), &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Code())
	require.Equal(t, "4.0", out.Version)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 2, authCalls, "401 must force a new /_open/auth even when exp is still in the future")
}

func TestNewJWTAuthWrapper_DoesNotLoginOnEverySuccessfulRequest(t *testing.T) {
	var mu sync.Mutex
	authCalls := 0
	accepted := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_open/auth") {
			mu.Lock()
			authCalls++
			accepted = fakeJWT("stable", time.Now().Add(time.Hour))
			token := accepted
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"jwt": token}))
			return
		}
		mu.Lock()
		want := accepted
		mu.Unlock()
		if r.Header.Get("Authorization") != "bearer "+want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"4.0"}`))
	}))
	defer srv.Close()

	conn := NewJWTAuthWrapper("root", "passwd")(NewHttpConnection(HttpConfiguration{
		Endpoint:    NewRoundRobinEndpoints([]string{srv.URL}),
		ContentType: ApplicationJSON,
	}))

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		var out jwtVersionResponse
		resp, err := CallGet(ctx, conn, NewUrl("_api", "version"), &out)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.Code())
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, authCalls, "successful requests must reuse the JWT on the connection")
}

func fakeJWT(sub string, exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"sub": sub,
		"exp": exp.Unix(),
	})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
