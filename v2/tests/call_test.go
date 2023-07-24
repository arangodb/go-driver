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

package tests

import (
	"context"
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
