//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Jakub Wierzbowski
//

package tests

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/connection"

	"github.com/stretchr/testify/require"
)

func TestContextWithArangoQueueTimeoutParams(t *testing.T) {
	c := newClient(t, connectionJsonHttp(t))

	version, err := c.Version(context.Background())
	require.NoError(t, err)
	if version.Version.CompareTo("3.9.0") < 0 {
		t.Skipf("Version of the ArangoDB should be at least 3.9.0")
	}

	t.Run("without timout", func(t *testing.T) {
		_, err := c.Version(context.Background())
		require.NoError(t, err)
	})

	t.Run("without timeout - if no queue timeout and no context deadline set", func(t *testing.T) {
		ctx := connection.WithArangoQueueTimeout(context.Background(), true)

		_, err := c.Version(ctx)
		require.NoError(t, err)
	})

}
