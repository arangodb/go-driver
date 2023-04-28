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

// Test_DecoderBytesWithJSONConnection gets plain text response from the server using JSON connection.
func Test_DecoderBytesWithJSONConnection(t *testing.T) {
	conn := connectionJsonHttp(t)
	waitForConnection(t, arangodb.NewClient(conn))

	var output []byte

	url := connection.NewUrl("_admin", "metrics", "v2")
	_, err := connection.CallGet(context.Background(), conn, url, &output)
	require.NoError(t, err)
	require.NotNil(t, output)
	// Check the e
	assert.Contains(t, string(output), "arangodb_connection_pool")
	output = nil
}

// Test_DecoderBytesWithPlainConnection gets plain text response from the server using plain connection.
func Test_DecoderBytesWithPlainConnection(t *testing.T) {
	conn := connectionPlainHttp(t)
	client := newClient(t, conn)

	// Check if the JSON deserializer worked.
	version, err := client.Version(context.Background())
	require.NoErrorf(t, err, "can not fetch a version with a plain connection: `%v`", err)
	require.Equalf(t, true, version.Version.Major() > 0, "can not fetch a version with a plain connection")

	var output []byte

	url := connection.NewUrl("_admin", "metrics", "v2")
	_, err = connection.CallGet(context.Background(), conn, url, &output)
	require.NoError(t, err)
	require.NotNil(t, output)
	// Check the e
	assert.Contains(t, string(output), "arangodb_connection_pool")
	output = nil
}
