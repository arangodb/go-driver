//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Test_AccessTokens validates the full lifecycle of access tokens including creation, retrieval, duplication, deletion, and error handling for invalid/missing parameters.
func Test_AccessTokens(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

			var tokenResp *arangodb.CreateAccessTokenResponse
			expiresAt := time.Now().Add(5 * time.Minute).Unix()
			user := "root"

			t.Run("Create Access Token With All valid data", func(t *testing.T) {
				var err error
				maxRetries := 3

				for i := 0; i < maxRetries; i++ {
					tokenName := fmt.Sprintf("Token-%s", uuid.New().String())
					cleanupToken(ctx, t, client, user, tokenName)

					req := arangodb.AccessTokenRequest{
						Name:       utils.NewType(tokenName),
						ValidUntil: utils.NewType(expiresAt),
					}

					resp, err := client.CreateAccessToken(ctx, &user, req)
					if err == nil {
						tokenResp = &resp
						require.NotNil(t, tokenResp)
						require.NotNil(t, tokenResp.Id)
						require.NotNil(t, tokenResp.Token)
						require.NotNil(t, tokenResp.Fingerprint)
						require.Equal(t, tokenName, *tokenResp.Name)
						require.Equal(t, true, *tokenResp.Active)
						require.Equal(t, expiresAt, *tokenResp.ValidUntil)
						break // success
					}

					// if conflict, retry; else fail immediately
					var arangoErr shared.ArangoError
					if errors.As(err, &arangoErr) && arangoErr.Code == 409 {
						t.Logf("Conflict detected, retrying token creation... attempt %d\n", i+1)
						continue
					} else {
						break
					}
				}
				require.NoError(t, err)
			})

			t.Run("Get All Access Tokens", func(t *testing.T) {
				if tokenResp == nil || tokenResp.Id == nil {
					t.Skip("Skipping test because token creation failed")
				}

				// Retry logic to handle eventual consistency in cluster mode
				// The token may not appear immediately after creation due to replication lag
				var found bool
				err := NewTimeout(func() error {
					tokens, err := client.GetAllAccessToken(ctx, &user)
					if err != nil {
						return err
					}
					if tokens.Tokens == nil {
						return nil
					}

					t.Logf("Tokens size %d", len(tokens.Tokens))
					for _, token := range tokens.Tokens {
						if token.Id != nil && tokenResp.Id != nil && *token.Id == *tokenResp.Id {
							require.Equal(t, tokenResp.Id, token.Id)
							require.Equal(t, tokenResp.Name, token.Name)
							require.Equal(t, tokenResp.Fingerprint, token.Fingerprint)
							require.Equal(t, tokenResp.Active, token.Active)
							require.Equal(t, tokenResp.CreatedAt, token.CreatedAt)
							require.Equal(t, tokenResp.ValidUntil, token.ValidUntil)
							found = true
							return Interrupt{} // Success - stop retrying
						}
					}
					return nil // Token not found yet, retry
				}).Timeout(15*time.Second, 250*time.Millisecond)

				require.NoError(t, err, "Failed to find created token in the list after retries")
				require.True(t, found, "Created token should be present in the list")
			})

			t.Run("Client try to create duplicate access token name", func(t *testing.T) {
				if tokenResp == nil || tokenResp.Name == nil {
					t.Skip("Skipping delete test because token creation failed")
				}
				t.Logf("Client try to create duplicate access token name - token name: %s\n", *tokenResp.Name)
				req := arangodb.AccessTokenRequest{
					Name:       utils.NewType(*tokenResp.Name),
					ValidUntil: utils.NewType(expiresAt),
				}
				_, err := client.CreateAccessToken(ctx, &user, req)
				require.Error(t, err)
				if err != nil {
					var arangoErr shared.ArangoError
					if errors.As(err, &arangoErr) {
						t.Logf("Arango validation error: code=%d, msg=%s", arangoErr.Code, arangoErr.ErrorMessage)
						t.Logf("CreateAccessToken failed: %v", err)
					}
				}
			})

			t.Run("Delete Access Token", func(t *testing.T) {
				if tokenResp == nil || tokenResp.Id == nil {
					t.Skip("Skipping delete test because token creation failed")
				}
				err := client.DeleteAccessToken(ctx, &user, tokenResp.Id)
				if err != nil {
					t.Logf("DeleteAccessToken failed: %v", err)
				}
				require.NoError(t, err)
			})

			t.Run("Create Access Token With invalid user", func(t *testing.T) {
				invalidUser := "roothyd"
				tokenName := fmt.Sprintf("Token-%s", uuid.New().String())
				t.Logf("Create Access Token With invalid user - Creating token with name: %s\n", tokenName)
				req := arangodb.AccessTokenRequest{
					Name:       utils.NewType(tokenName),
					ValidUntil: utils.NewType(expiresAt),
				}

				_, err := client.CreateAccessToken(ctx, &invalidUser, req)
				require.Error(t, err)
				if err != nil {
					var arangoErr shared.ArangoError
					if errors.As(err, &arangoErr) {
						t.Logf("Arango validation error: code=%d, msg=%s", arangoErr.Code, arangoErr.ErrorMessage)
						t.Logf("CreateAccessToken failed: %v", err)
					}
				}
			})

			t.Run("Create Access Token With missing user", func(t *testing.T) {
				tokenName := fmt.Sprintf("Token-%s", uuid.New().String())
				t.Logf("Create Access Token With missing user - Creating token with name: %s\n", tokenName)
				localExpiresAt := time.Now().Add(5 * time.Minute).Unix()
				req := arangodb.AccessTokenRequest{
					Name:       utils.NewType(tokenName),
					ValidUntil: utils.NewType(localExpiresAt),
				}

				_, err := client.CreateAccessToken(ctx, nil, req)
				require.Error(t, err)
				if err != nil {
					var clientErr *arangodb.ClientError
					if errors.As(err, &clientErr) {
						t.Logf("Client validation error: code=%d, msg=%s", clientErr.Code, clientErr.Message)
						t.Logf("CreateAccessToken failed: %v", err)
					}
				}
			})

			t.Run("Create Access Token With missing name", func(t *testing.T) {
				req := arangodb.AccessTokenRequest{
					ValidUntil: utils.NewType(expiresAt),
				}

				_, err := client.CreateAccessToken(ctx, &user, req)
				require.Error(t, err)
				if err != nil {
					var clientErr *arangodb.ClientError
					if errors.As(err, &clientErr) {
						t.Logf("Client validation error: code=%d, msg=%s", clientErr.Code, clientErr.Message)
						t.Logf("CreateAccessToken failed: %v", err)
					}
				}
			})
		})
	})
}

// Cleanup tokens with the same name
func cleanupToken(ctx context.Context, t *testing.T, client arangodb.Client, user string, tokenName string) {
	tokens, err := client.GetAllAccessToken(ctx, &user)
	if err != nil {
		t.Logf("Failed to list tokens for cleanup: %v", err)
		return
	}

	for _, token := range tokens.Tokens {
		if token.Name != nil && *token.Name == tokenName {
			if token.Id != nil {
				err := client.DeleteAccessToken(ctx, &user, token.Id)
				if err != nil {
					t.Logf("Failed to delete token %s: %v", *token.Name, err)
				}
			}
		}
	}
}
