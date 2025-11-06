//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
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

package driver_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

func TestNewClient(t *testing.T) {
	mockConn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"localhost"},
	})
	require.NoError(t, err)

	cfg := driver.ClientConfig{
		Connection:                   mockConn,
		SynchronizeEndpointsInterval: time.Second * 20,
	}

	var clients = make(map[int]driver.Client)

	before := runtime.NumGoroutine()
	const iterations = 30
	for i := 0; i < iterations; i++ {
		c, err := driver.NewClient(cfg)
		require.NoError(t, err, "iter %d", i)

		clients[i] = c
	}

	after := runtime.NumGoroutine()

	// SynchronizeEndpointsInterval feature has a bug where new go-routine would be created per each call to NewClient
	// This feature should not be used. The test is present here only to document this behaviour.
	require.Equal(t, iterations, after-before)
}
