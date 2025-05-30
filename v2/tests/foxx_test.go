//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

func Test_FoxxItzpapalotlService(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			t.Run("Install and uninstall Foxx", func(t *testing.T) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {

					if os.Getenv("TEST_CONNECTION") == "vst" {
						skipBelowVersion(client, ctx, "3.6", t)
					}

					// /tmp/resources/ directory is provided by .travis.yml
					zipFilePath := "/tmp/resources/itzpapalotl-v1.2.0.zip"
					if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
						// Test works only via travis pipeline unless the above file exists locally
						t.Skipf("file %s does not exist", zipFilePath)
					}

					timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
					mountName := "test"
					options := &arangodb.FoxxCreateOptions{
						Mount: utils.NewType[string]("/" + mountName),
					}
					err := client.InstallFoxxService(timeoutCtx, db.Name(), zipFilePath, options)
					cancel()
					require.NoError(t, err)

					timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*30)
					resp, err := connection.CallGet(ctx, client.Connection(), "_db/_system/"+mountName+"/random", nil, nil, nil)
					require.NotNil(t, resp)

					value, ok := resp, true
					require.Equal(t, true, ok)
					require.NotEmpty(t, value)
					cancel()

					timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*30)
					deleteOptions := &arangodb.FoxxDeleteOptions{
						Mount:    utils.NewType[string]("/" + mountName),
						Teardown: utils.NewType[bool](true),
					}
					err = client.UninstallFoxxService(timeoutCtx, db.Name(), deleteOptions)
					cancel()
					require.NoError(t, err)
				})
			})
		})
	})
}
