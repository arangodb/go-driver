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

package shared

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ResponseStruct_AsArangoErrorWithCode(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var response *ResponseStruct
		err := response.AsArangoErrorWithCode(400)
		require.Equal(t, 400, err.Code)
		require.True(t, err.HasError)
	})

	t.Run("updates the caller response", func(t *testing.T) {
		response := &ResponseStruct{}
		err := response.AsArangoErrorWithCode(404)
		require.Equal(t, 404, err.Code)
		require.True(t, err.HasError)
		require.NotNil(t, response.Code)
		require.Equal(t, 404, *response.Code)
		require.NotNil(t, response.Error)
		require.True(t, *response.Error)
	})
}
