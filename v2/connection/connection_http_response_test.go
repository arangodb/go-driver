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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_httpResponse_Content(t *testing.T) {

	t.Run("pure content type", func(t *testing.T) {
		j := httpResponse{
			response: &http.Response{
				Header: map[string][]string{},
			},
		}
		j.response.Header.Set(ContentType, "application/json")

		assert.Equal(t, "application/json", j.Content())
	})

	t.Run("content type with the arguments", func(t *testing.T) {
		j := httpResponse{
			response: &http.Response{
				Header: map[string][]string{},
			},
		}
		j.response.Header.Set(ContentType, "text/plain; charset=UTF-8")

		assert.Equal(t, "text/plain", j.Content())
	})

	t.Run("empty content type", func(t *testing.T) {
		j := httpResponse{
			response: &http.Response{
				Header: map[string][]string{},
			},
		}
		j.response.Header.Set(ContentType, "")

		assert.Equal(t, "", j.Content())
	})

	t.Run("content type header does not exist", func(t *testing.T) {
		j := httpResponse{
			response: &http.Response{
				Header: map[string][]string{},
			},
		}

		assert.Equal(t, "", j.Content())
	})
}

func Test_httpResponse_CheckStatus(t *testing.T) {
	j := httpResponse{
		response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}

	t.Run("code expected", func(t *testing.T) {
		require.NoError(t, j.CheckStatus(http.StatusOK))
	})

	t.Run("code not expected", func(t *testing.T) {
		require.Error(t, j.CheckStatus(http.StatusConflict, http.StatusInternalServerError))
	})
}
