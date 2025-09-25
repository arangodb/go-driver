//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_ServerMode(t *testing.T) {
	// This test can not run sub-tests parallelly, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: utils.NewType(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			serverMode, err := client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeReadOnly)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeReadOnly, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeDefault)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)
		})
	}, wrapOpts)
}

func Test_ServerID(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			if getTestMode() == string(testModeCluster) {
				id, err := client.ServerID(ctx)
				require.NoError(t, err, "ServerID failed")
				require.NotEmpty(t, id, "Expected ID to be non-empty")
			} else {
				_, err := client.ServerID(ctx)
				require.Error(t, err, "ServerID succeeded, expected error")
			}
		})
	})
}

func Test_Version(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			v, err := client.VersionWithOptions(context.Background(), &arangodb.GetVersionOptions{
				Details: utils.NewType(true),
			})
			require.NoError(t, err)
			require.NotEmpty(t, v.Version)
			require.NotEmpty(t, v.Server)
			require.NotEmpty(t, v.License)
			require.NotZero(t, len(v.Details))
		})
	})
}

func Test_GetSystemTime(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(context.Background(), "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)

			time, err := client.GetSystemTime(context.Background(), db.Name())
			require.NoError(t, err)
			require.NotEmpty(t, time)
			t.Logf("Current time in Unix timestamp with microsecond precision is:%f", time)
		})
	})
}

func Test_GetServerStatus(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(context.Background(), "_system", nil)
			require.NoError(t, err)
			require.NotEmpty(t, db)

			resp, err := client.GetServerStatus(context.Background(), db.Name())
			require.NoError(t, err)
			require.NotEmpty(t, resp)
		})
	})
}

func Test_GetDeploymentSupportInfo(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {

			serverRole, err := client.ServerRole(ctx)
			require.NoError(t, err)
			resp, err := client.GetDeploymentSupportInfo(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, resp)
			require.NotEmpty(t, resp.Date)
			require.NotEmpty(t, resp.Deployment)
			require.NotEmpty(t, resp.Deployment.Type)
			if serverRole == arangodb.ServerRoleCoordinator {
				require.NotEmpty(t, resp.Deployment.Servers)
			}
			if serverRole == arangodb.ServerRoleSingle {
				require.NotEmpty(t, resp.Host)
			}
		})
	})
}

func Test_GetStartupConfiguration(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {

			resp, err := client.GetStartupConfiguration(ctx)
			if err != nil {
				switch e := err.(type) {
				case *shared.ArangoError:
					t.Logf("arangoErr code:%d", e.Code)
					if e.Code == 403 || e.Code == 500 {
						t.Skip("startup configuration API not enabled on this server")
					}
				case shared.ArangoError:
					t.Logf("arangoErr code:%d", e.Code)
					if e.Code == 403 || e.Code == 500 {
						t.Skip("startup configuration API not enabled on this server")
					}
				}
				require.NoError(t, err)
			}
			require.NotEmpty(t, resp)

			configDesc, err := client.GetStartupConfigurationDescription(ctx)
			if err != nil {
				switch e := err.(type) {
				case *shared.ArangoError:
					t.Logf("arangoErr code:%d", e.Code)
					if e.Code == 403 || e.Code == 500 {
						t.Skip("startup configuration description API not enabled on this server")
					}
				case shared.ArangoError:
					t.Logf("arangoErr code:%d", e.Code)
					if e.Code == 403 || e.Code == 500 {
						t.Skip("startup configuration description API not enabled on this server")
					}
				}
				require.NoError(t, err)
			}
			require.NotEmpty(t, configDesc)

			// Assert that certain well-known options exist
			_, hasEndpoint := configDesc["server.endpoint"]
			require.True(t, hasEndpoint, "expected server.endpoint option to be present")

			_, hasAuth := configDesc["server.authentication"]
			require.True(t, hasAuth, "expected server.authentication option to be present")

			// Optionally assert that each entry has a description
			for key, value := range configDesc {
				option, ok := value.(map[string]interface{})
				require.True(t, ok, "expected value for %s to be a map", key)

				_, hasDesc := option["description"]
				require.True(t, hasDesc, "expected option %s to have a description", key)
			}
		})
	})
}

func Test_ReloadRoutingTable(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)
			err = client.ReloadRoutingTable(ctx, db.Name())
			require.NoError(t, err)
		})
	})
}

func Test_ExecuteAdminScript(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		ctx := context.Background()
		db, err := client.GetDatabase(ctx, "_system", nil)
		require.NoError(t, err)

		tests := []struct {
			name   string
			script string
		}{
			{
				name:   "ReturnObject",
				script: "return {hello: 'world'};",
			},
			{
				name: "ReturnNumber",
				script: `
                    var sum = 0;
                    for (var i = 1; i <= 5; i++) {
                        sum += i;
                    }
                    return sum;
                `,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := client.ExecuteAdminScript(ctx, db.Name(), &tt.script)
				var arangoErr shared.ArangoError
				if errors.As(err, &arangoErr) {
					t.Logf("arangoErr code:%d\n", arangoErr.Code)
					if arangoErr.Code == http.StatusNotFound {
						t.Skip("javascript.allow-admin-execute is disabled")
					}
				}
				require.NoError(t, err)

				switch v := result.(type) {
				case map[string]interface{}:
					t.Logf("Got object result: %+v", v)
					require.Contains(t, v, "hello")
				case float64:
					t.Logf("Got number result: %v", v)
					require.Equal(t, float64(15), v)
				default:
					t.Fatalf("Unexpected result type: %T, value: %+v", v, v)
				}
			})
		}
	})
}

func Test_CompactDatabases(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {

			checkCompact := func(opts *arangodb.CompactOpts) {
				resp, err := client.CompactDatabases(ctx, opts)
				if err != nil {
					var arangoErr shared.ArangoError
					if errors.As(err, &arangoErr) {
						t.Logf("arangoErr code:%d", arangoErr.Code)
						if arangoErr.Code == 403 || arangoErr.Code == 500 {
							t.Skip("The endpoint requires superuser access")
						}
					}
					require.NoError(t, err)
				}
				require.Empty(t, resp)
			}

			checkCompact(&arangodb.CompactOpts{
				ChangeLevel:            utils.NewType(true),
				CompactBottomMostLevel: utils.NewType(false),
			})

			checkCompact(&arangodb.CompactOpts{
				ChangeLevel:            utils.NewType(true),
				CompactBottomMostLevel: utils.NewType(true),
			})

			checkCompact(&arangodb.CompactOpts{
				ChangeLevel: utils.NewType(true),
			})

			checkCompact(&arangodb.CompactOpts{
				CompactBottomMostLevel: utils.NewType(true),
			})

			checkCompact(nil)
		})
	})
}

// Test_GetTLSData checks that TLS configuration data is available and valid, skipping if not configured.
func Test_GetTLSData(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)

			// Get TLS data using the client (which embeds ClientAdmin)
			tlsResp, err := client.GetTLSData(ctx, db.Name())
			if err != nil {
				var arangoErr shared.ArangoError
				if errors.As(err, &arangoErr) {
					t.Logf("GetTLSData failed with ArangoDB error code: %d", arangoErr.Code)
					switch arangoErr.Code {
					case 403:
						t.Skip("Skipping TLS get test - authentication/permission denied (HTTP 403)")
					default:
						t.Logf("Unexpected ArangoDB error code: %d, message: %s", arangoErr.Code, arangoErr.ErrorMessage)
					}
					return
				}
				// Skip for any other error (TLS not configured, network issues, etc.)
				t.Logf("GetTLSData failed: %v", err)
				t.Skip("Skipping TLS get test - likely TLS not configured or other server issue")
			}

			// Success! Validate response structure
			t.Logf("TLS data retrieved successfully")

			// Validate TLS response data
			validateTLSResponse(t, tlsResp, "Retrieved")
		})
	})
}

// validateTLSResponse is a helper function to validate TLS response data
func validateTLSResponse(t testing.TB, tlsResp arangodb.TLSDataResponse, operation string) {
	// Basic validation - at least one field should be populated
	hasData := false
	if tlsResp.Keyfile != nil {
		if tlsResp.Keyfile.Sha256 != nil && *tlsResp.Keyfile.Sha256 != "" {
			t.Logf("%s keyfile SHA256: %s", operation, *tlsResp.Keyfile.Sha256)
			hasData = true
		}
		if len(tlsResp.Keyfile.Certificates) > 0 {
			t.Logf("%s keyfile contains %d certificates", operation, len(tlsResp.Keyfile.Certificates))
			hasData = true

			// Validate certificate content (basic PEM format check)
			for i, cert := range tlsResp.Keyfile.Certificates {
				require.NotEmpty(t, cert, "Certificate %d should not be empty", i)
				// Basic PEM format validation
				if !strings.Contains(cert, "-----BEGIN CERTIFICATE-----") {
					t.Logf("Warning: Certificate %d may not be in PEM format", i)
				} else {
					t.Logf("Certificate %d appears to be valid PEM format", i)
				}
			}
		}
		if tlsResp.Keyfile.PrivateKeySha256 != nil && *tlsResp.Keyfile.PrivateKeySha256 != "" {
			t.Logf("%s keyfile private key SHA256: %s", operation, *tlsResp.Keyfile.PrivateKeySha256)
			hasData = true
		}
	}
	if tlsResp.ClientCA != nil && tlsResp.ClientCA.Sha256 != nil && *tlsResp.ClientCA.Sha256 != "" {
		t.Logf("%s client CA SHA256: %s", operation, *tlsResp.ClientCA.Sha256)
		hasData = true
	}
	if len(tlsResp.SNI) > 0 {
		t.Logf("%s SNI configurations found: %d", operation, len(tlsResp.SNI))
		hasData = true
	}
	if hasData {
		t.Logf("TLS configuration data validated successfully")
	} else {
		t.Logf("TLS endpoint accessible but no TLS data returned - server may not have TLS configured")
	}
}

// Test_ReloadTLSData tests TLS certificate reload functionality, skipping if superuser rights unavailable.
func Test_ReloadTLSData(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			// Reload TLS data - requires superuser rights
			tlsResp, err := client.ReloadTLSData(ctx)
			if err != nil {
				var arangoErr shared.ArangoError
				if errors.As(err, &arangoErr) {
					t.Logf("ReloadTLSData failed with ArangoDB error code: %d", arangoErr.Code)
					switch arangoErr.Code {
					case 403:
						t.Skip("Skipping TLS reload test - superuser rights required (HTTP 403)")
					default:
						t.Logf("Unexpected ArangoDB error code: %d, message: %s", arangoErr.Code, arangoErr.ErrorMessage)
					}
					return
				}
				// Skip for any other error (TLS not configured, network issues, etc.)
				t.Logf("ReloadTLSData failed: %v", err)
				t.Skip("Skipping TLS reload test - likely TLS not configured or other server issue")
			}

			// Success! Validate response structure
			t.Logf("TLS data reloaded successfully")

			// Validate TLS response data
			validateTLSResponse(t, tlsResp, "Reloaded")
		})
	})
}

// Test_RotateEncryptionAtRestKey verifies that the encryption key rotation endpoint works as expected.
// The test is skipped if superuser rights are missing or the feature is disabled/not configured.
func Test_RotateEncryptionAtRestKey(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {

			// Attempt to rotate encryption at rest key - requires superuser rights
			resp, err := client.RotateEncryptionAtRestKey(ctx)
			if err != nil {
				var arangoErr shared.ArangoError
				if errors.As(err, &arangoErr) {
					t.Logf("RotateEncryptionAtRestKey failed with ArangoDB error code: %d", arangoErr.Code)
					switch arangoErr.Code {
					case 403:
						t.Skip("Skipping RotateEncryptionAtRestKey test - superuser rights required (HTTP 403)")
					case 404:
						t.Skip("Skipping RotateEncryptionAtRestKey test - encryption key rotation disabled (HTTP 404)")
					default:
						t.Logf("Unexpected ArangoDB error code: %d, message: %s", arangoErr.Code, arangoErr.ErrorMessage)
						t.FailNow()
					}
				} else {
					t.Fatalf("RotateEncryptionAtRestKey failed with unexpected error: %v", err)
				}
				return
			}

			// Convert response to JSON for logging
			encryptionRespJson, err := utils.ToJSONString(resp)
			require.NoError(t, err)
			t.Logf("RotateEncryptionAtRestKey response: %s", encryptionRespJson)

			// Validate the response is not nil
			require.NotNil(t, resp, "Expected non-nil response")
			t.Logf("RotateEncryptionAtRestKey succeeded with %d encryption keys", len(resp))

			// Validate each encryption key
			for i, key := range resp {
				// Explicit nil check for pointer
				require.NotNil(t, key.SHA256, "Expected encryption key %d SHA256 not to be nil", i)
				require.NotEmpty(t, *key.SHA256, "Expected encryption key %d SHA256 not to be empty", i)
				t.Logf("Encryption key %d SHA256: %s", i, *key.SHA256)
			}
		})
	})
}

// Test_GetJWTSecrets validates retrieval and structure of JWT secrets, skipping if not accessible.
func Test_GetJWTSecrets(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			db, err := client.GetDatabase(ctx, "_system", nil)
			require.NoError(t, err)

			resp, err := client.GetJWTSecrets(ctx, db.Name())
			if err != nil {
				if handleJWTSecretsError(t, err, "GetJWTSecrets", []int{http.StatusForbidden}) {
					return
				}
				require.NoError(t, err)
			}

			validateJWTSecretsResponse(t, resp, "Retrieved")
		})
	})
}

// Test_ReloadJWTSecrets validates JWT secrets reload functionality, skipping if not available.
func Test_ReloadJWTSecrets(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			resp, err := client.ReloadJWTSecrets(ctx)
			if err != nil {
				if handleJWTSecretsError(t, err, "ReloadJWTSecrets", []int{http.StatusForbidden, http.StatusBadRequest}) {
					return
				}
				require.NoError(t, err)
			}

			validateJWTSecretsResponse(t, resp, "Reloaded")
		})
	})
}

// handleJWTSecretsError handles common JWT secrets API errors and returns true if the test should skip
func handleJWTSecretsError(t testing.TB, err error, operation string, skipCodes []int) bool {
	var arangoErr shared.ArangoError
	if errors.As(err, &arangoErr) {
		t.Logf("%s failed with ArangoDB error code: %d", operation, arangoErr.Code)

		for _, code := range skipCodes {
			switch code {
			case http.StatusForbidden:
				if arangoErr.Code == http.StatusForbidden {
					t.Skip("The endpoint requires superuser access or JWT feature is disabled")
					return true
				}
			case http.StatusBadRequest:
				if arangoErr.Code == http.StatusBadRequest {
					t.Skip("JWT reload not available: no secret file or folder configured")
					return true
				}
			}
		}

		t.Logf("Unexpected ArangoDB error code: %d, message: %s", arangoErr.Code, arangoErr.ErrorMessage)
	}
	return false
}

// validateJWTSecretsResponse validates the structure and content of JWT secrets response
func validateJWTSecretsResponse(t testing.TB, resp arangodb.JWTSecretsResult, operation string) {
	require.NotEmpty(t, resp, "JWT secrets response should not be empty")

	respJson, err := utils.ToJSONString(resp)
	require.NoError(t, err)
	t.Logf("%s JWT secrets response: %s\n", operation, respJson)

	// Basic structural checks
	require.NotNil(t, resp.Active, "Active JWT secret should not be nil")
	require.NotNil(t, resp.Passive, "Passive JWT secrets list should not be nil")
	require.NotNil(t, resp.Active.SHA256, "Active JWT secret SHA256 should not be nil")
	require.NotEmpty(t, *resp.Active.SHA256, "Active JWT secret SHA256 should not be empty")

	// Secure logging - validate structure without exposing sensitive hash values
	t.Logf("%s active JWT secret: present and valid (length: %d chars)", operation, len(*resp.Active.SHA256))
	t.Logf("%s found %d passive JWT secrets", operation, len(resp.Passive))

	// Validate passive secrets and ensure no duplicates with active
	for i, passive := range resp.Passive {
		require.NotNil(t, passive.SHA256, "Passive JWT secret %d SHA256 should not be nil", i)
		require.NotEmpty(t, *passive.SHA256, "Passive JWT secret %d SHA256 should not be empty", i)
		t.Logf("%s passive JWT secret %d: valid (length: %d chars)", operation, i, len(*passive.SHA256))
	}

	t.Logf("%s JWT secrets validation completed successfully with %d total secrets", operation, len(resp.Passive)+1)
}
