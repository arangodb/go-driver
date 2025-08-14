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
	"github.com/arangodb/go-driver/v2/utils"
)

func Test_FoxxItzpapalotlService(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {

		ctx := context.Background()
		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		if os.Getenv("TEST_CONNECTION") == "vst" {
			skipBelowVersion(client, ctx, "3.6", t)
		}

		// /tmp/resources/ directory is provided by .travis.yml
		zipFilePath := "/tmp/resources/itzpapalotl-v1.2.0.zip"
		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			// Test works only via travis pipeline unless the above file exists locally
			t.Skipf("file %s does not exist", zipFilePath)
		}
		mountName := "test"
		options := &arangodb.FoxxDeploymentOptions{
			Mount: utils.NewType[string]("/" + mountName),
		}

		// InstallFoxxService
		t.Run("Install and verify installed Foxx service", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)

				err = client.InstallFoxxService(timeoutCtx, db.Name(), zipFilePath, options)
				cancel()
				require.NoError(t, err)

				// Try to fetch random name from installed foxx sercice
				timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*30)
				connection := client.Connection()
				req, err := connection.NewRequest("GET", "_db/"+db.Name()+"/"+mountName+"/random")
				require.NoError(t, err)
				resp, err := connection.Do(timeoutCtx, req, nil)
				require.NoError(t, err)
				require.NotNil(t, resp)

				value, ok := resp, true
				require.Equal(t, true, ok)
				require.NotEmpty(t, value)
				cancel()
			})
		})

		// ReplaceFoxxService
		t.Run("Replace Foxx service", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				err = client.ReplaceFoxxService(timeoutCtx, db.Name(), zipFilePath, options)
				cancel()
				require.NoError(t, err)
			})
		})

		// UpgradeFoxxService
		t.Run("Upgrade Foxx service", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				err = client.UpgradeFoxxService(timeoutCtx, db.Name(), zipFilePath, options)
				cancel()
				require.NoError(t, err)
			})
		})

		// Foxx Service Configurations
		t.Run("Fetch Foxx service Configuration", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.GetFoxxServiceConfiguration(timeoutCtx, db.Name(), options.Mount)
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Update Foxx service Configuration
		t.Run("Update Foxx service Configuration", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.UpdateFoxxServiceConfiguration(timeoutCtx, db.Name(), options.Mount, map[string]interface{}{
					"apiKey":   "abcdef",
					"maxItems": 100,
				})
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Replace Foxx service Configuration
		t.Run("Replace Foxx service Configuration", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.ReplaceFoxxServiceConfiguration(timeoutCtx, db.Name(), options.Mount, map[string]interface{}{
					"apiKey":   "xyz987",
					"maxItems": 100,
				})
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Fetch Foxx Service Dependencies
		t.Run("Fetch Foxx Service Dependencies", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.GetFoxxServiceDependencies(timeoutCtx, db.Name(), options.Mount)
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Update Foxx Service Dependencies
		t.Run("Update Foxx Service Dependencies", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.UpdateFoxxServiceDependencies(timeoutCtx, db.Name(), options.Mount, map[string]interface{}{
					"title": "Auth Service",
				})
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Replace Foxx Service Dependencies
		t.Run("Replace Foxx Service Dependencies", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.ReplaceFoxxServiceDependencies(timeoutCtx, db.Name(), options.Mount, map[string]interface{}{
					"title":       "Auth Service",
					"description": "Service that handles authentication",
					"mount":       "/auth-v2",
				})
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Fetch Foxx Service Scripts
		t.Run("Fetch Foxx Service Scripts", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				resp, err := client.GetFoxxServiceScripts(timeoutCtx, db.Name(), options.Mount)
				cancel()
				require.NoError(t, err)
				require.NotNil(t, resp)
			})
		})

		// Run Foxx Service Script
		t.Run("Run Foxx Service Script", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				scriptName := "cleanupData"
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
				_, err := client.RunFoxxServiceScript(timeoutCtx, db.Name(), scriptName, options.Mount,
					map[string]interface{}{
						"cleanupData": "Cleanup Old Data",
					})
				cancel()
				require.Error(t, err)
			})
		})

		// UninstallFoxxService
		t.Run("Uninstall Foxx service", func(t *testing.T) {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
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
}

func Test_ListInstalledFoxxServices(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		// excludeSystem := false
		services, err := client.ListInstalledFoxxServices(ctx, db.Name(), nil)
		require.NoError(t, err)
		require.NotEmpty(t, services)
		require.GreaterOrEqual(t, len(services), 0)

		if len(services) == 0 {
			t.Log("No Foxx services found.")
			return
		}

		for _, service := range services {
			require.NotEmpty(t, service.Mount)
			require.NotEmpty(t, service.Name)
			require.NotEmpty(t, service.Version)
			require.NotNil(t, service.Development)
			require.NotNil(t, service.Provides)
			require.NotNil(t, service.Legacy)
		}
	})
}

func Test_GetInstalledFoxxService(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		mount := "/_api/foxx"
		serviceDetails, err := client.GetInstalledFoxxService(ctx, db.Name(), &mount)
		require.NoError(t, err)
		require.NotEmpty(t, serviceDetails)
		require.NotNil(t, serviceDetails.Mount)
		require.NotNil(t, serviceDetails.Name)
		require.NotNil(t, serviceDetails.Version)
		require.NotNil(t, serviceDetails.Development)
		require.NotNil(t, serviceDetails.Path)
		require.NotNil(t, serviceDetails.Legacy)
		require.NotNil(t, serviceDetails.Manifest)
	})
}
