//
// DISCLAIMER
//
// Copyright 2021-2024 ArangoDB GmbH, Cologne, Germany
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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func Test_CallStream(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			url := connection.NewUrl("_api", "version")

			resp, body, err := connection.CallStream(ctx, client.Connection(), http.MethodGet, url)
			require.NoError(t, err)
			defer body.Close()
			require.Equal(t, http.StatusOK, resp.Code())
			dec := client.Connection().Decoder(resp.Content())

			version := arangodb.VersionInfo{}
			require.NoError(t, dec.Decode(body, &version))
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			require.Len(t, data, 0)
			require.GreaterOrEqual(t, version.Version.Major(), 3)
		})
	})
}

func Test_CallWithChecks(t *testing.T) {
	t.Run("code-allowed", func(t *testing.T) {
		Wrap(t, func(t *testing.T, client arangodb.Client) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				url := connection.NewUrl("_api", "version")

				version := arangodb.VersionInfo{}

				resp, err := connection.CallWithChecks(ctx, client.Connection(),
					http.MethodGet, url, &version, []int{http.StatusOK})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.Code())
				require.GreaterOrEqual(t, version.Version.Major(), 3)
			})
		})
	})

	t.Run("code-disallowed", func(t *testing.T) {
		Wrap(t, func(t *testing.T, client arangodb.Client) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
				url := connection.NewUrl("_api", "non-such-endpoint")

				version := arangodb.VersionInfo{}

				resp, err := connection.CallWithChecks(ctx, client.Connection(), http.MethodGet, url, &version,
					[]int{http.StatusOK, http.StatusNoContent})
				require.Error(t, err)

				arangoErr, ok := err.(shared.ArangoError)
				require.True(t, ok)
				require.True(t, arangoErr.HasError)
				require.True(t, shared.IsArangoError(arangoErr))
				require.Equal(t, http.StatusNotFound, resp.Code())
			})
		})
	})
}

func Test_Compression_Builtin(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				query := "FOR i IN 1..10 RETURN i"

				testCases := []struct {
					compression connection.CompressionType
					level       int
					request     bool
					response    bool
				}{
					{connection.RequestCompressionTypeGzip, gzip.BestCompression, true, true},
					{connection.RequestCompressionTypeDeflate, gzip.BestCompression, true, true},
					{connection.RequestCompressionTypeGzip, gzip.DefaultCompression, true, false},
					{connection.RequestCompressionTypeGzip, gzip.BestCompression, true, false},
					{connection.RequestCompressionTypeDeflate, gzip.DefaultCompression, true, false},
					{connection.RequestCompressionTypeDeflate, gzip.BestCompression, true, false},
				}

				for _, tc := range testCases {
					config := client.Connection().GetConfiguration()
					config.Compression = &connection.CompressionConfig{
						CompressionType:            tc.compression,
						RequestCompressionEnabled:  tc.request,
						RequestCompressionLevel:    tc.level,
						ResponseCompressionEnabled: tc.response,
					}
					client.Connection().SetConfiguration(config)

					t.Run(fmt.Sprintf("compression: %s, %d", tc.compression, tc.level), func(t *testing.T) {
						var result []int
						q, err := db.QueryBatch(ctx, query, nil, &result)
						require.NoError(t, err)

						require.Len(t, result, 10)
						require.False(t, q.HasMoreBatches())

						require.NoError(t, q.Close())
					})
				}
			})
		})
	})
}

func Test_Compression_Raw(t *testing.T) {
	requireExtraDBFeatures(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				testCases := []struct {
					compression connection.CompressionType
					level       int
					request     bool
					response    bool
				}{
					{connection.RequestCompressionTypeGzip, gzip.BestCompression, true, true},
					{connection.RequestCompressionTypeDeflate, gzip.BestCompression, true, true},
				}

				for _, tc := range testCases {
					config := client.Connection().GetConfiguration()
					config.Compression = &connection.CompressionConfig{
						CompressionType:            tc.compression,
						RequestCompressionEnabled:  tc.request,
						RequestCompressionLevel:    tc.level,
						ResponseCompressionEnabled: tc.response,
					}
					client.Connection().SetConfiguration(config)

					t.Run(fmt.Sprintf("compression raw: %s, %d", tc.compression, tc.level), func(t *testing.T) {
						var request = struct {
							Query string `json:"query"`
						}{
							Query: "FOR i IN 1..10 RETURN i",
						}

						var result = struct {
							shared.ResponseStruct `json:",inline"`
							Result                []int `json:"result"`
						}{}

						resp, err := client.Post(ctx, &result, request, "_api", "cursor")
						require.NoError(t, err)
						require.Equal(t, http.StatusCreated, resp.Code())
						// This header is available only if the response is compressed and server supports it
						require.Contains(t, resp.RawResponse().Header.Get("Content-Encoding"), tc.compression)
						require.Len(t, result.Result, 10)
					})
				}
			})
		})
	})
}
