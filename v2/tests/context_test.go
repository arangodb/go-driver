//
// DISCLAIMER
//
// Copyright 2017-2024 ArangoDB GmbH, Cologne, Germany
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

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func TestContextWithArangoQueueTimeoutParams(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, _ testing.TB) {
			skipBelowVersion(client, ctx, "3.9", t)
			t.Run("without timout", func(t *testing.T) {
				_, err := client.Version(context.Background())
				require.NoError(t, err)
			})

			t.Run("without timeout - if no queue timeout and no context deadline set", func(t *testing.T) {
				cfg := client.Connection().GetConfiguration()
				cfg.ArangoQueueTimeoutEnabled = true
				client.Connection().SetConfiguration(cfg)

				_, err := client.Version(ctx)
				require.NoError(t, err)
			})
		})
	})
}
