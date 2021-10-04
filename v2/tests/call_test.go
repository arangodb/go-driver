//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

func Test_CallStream(t *testing.T) {
	conn := connectionJsonHttp(t)
	waitForConnection(t, arangodb.NewClient(conn))

	url := connection.NewUrl("_api", "version")

	resp, body, err := connection.CallStream(context.Background(), conn, http.MethodGet, url)
	require.NoError(t, err)
	defer body.Close()
	require.Equal(t, http.StatusOK, resp.Code())
	dec := json.NewDecoder(body)

	version := arangodb.VersionInfo{}
	dec.Decode(&version)
	require.Equal(t, false, dec.More())
	require.GreaterOrEqual(t, version.Version.Major(), 3)
}
