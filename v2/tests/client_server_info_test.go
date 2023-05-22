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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// Test_ServerRole tests server role for all instances.
func Test_ServerRole(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContext(time.Second*10, func(ctx context.Context) error {
			testMode := getTestMode()

			t.Run("user endpoint", func(t *testing.T) {
				role, err := client.ServerRole(ctx)
				require.NoError(t, err)

				if testMode == string(testModeCluster) {
					require.Equal(t, role, arangodb.ServerRoleCoordinator)
				} else if testMode == string(testModeSingle) {
					require.Equal(t, role, arangodb.ServerRoleSingle)
				} else {
					require.Equal(t, role, arangodb.ServerRoleSingleActive)
				}
			})

			return nil
		})
	})
}
