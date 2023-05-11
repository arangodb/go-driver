//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// Test_DecoderBytes gets plain text response from the server
func Test_DecoderBytes(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		var output []byte

		url := connection.NewUrl("_admin", "metrics", "v2")
		_, err := connection.CallGet(context.Background(), client.Connection(), url, &output)

		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Contains(t, string(output), "arangodb_connection_pool")
	})
}
