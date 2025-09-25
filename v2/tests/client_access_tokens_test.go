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
	"github.com/stretchr/testify/require"
)

// Test_AccessTokens validates the full lifecycle of access tokens including creation, retrieval, duplication, deletion, and error handling for invalid/missing parameters.
func Test_AccessTokens(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {

			var tokenResp arangodb.CreateAccessTokenResponse
			// tokenName := fmt.Sprintf("Token-%d", time.Now().UnixNano())
			expiresAt := time.Now().Add(5 * time.Minute).Unix()
			user := "root"

			t.Run("Create Access Token With All valid data", func(t *testing.T) {
				tokenName := fmt.Sprintf("Token-%d", time.Now().UnixNano())
				req := arangodb.AccessTokenRequest{
					Name:       utils.NewType(tokenName),
					ValidUntil: utils.NewType(expiresAt),
				}
				var err error
				tokenResp, err = client.CreateAccessToken(ctx, &user, req)
				require.NoError(t, err)
				require.NotNil(t, tokenResp)
				require.NotNil(t, tokenResp.Id)
				require.NotNil(t, tokenResp.Token)
				require.NotNil(t, tokenResp.Fingerprint)
				require.Equal(t, tokenName, *tokenResp.Name)
				require.Equal(t, true, *tokenResp.Active)
				require.Equal(t, expiresAt, *tokenResp.ValidUntil)
			})

			t.Run("Get All Access Tokens", func(t *testing.T) {
				tokens, err := client.GetAllAccessToken(ctx, &user)
				require.NoError(t, err)
				if tokens.Tokens != nil || len(tokens.Tokens) > 0 {
					found := false
					for _, token := range tokens.Tokens {
						if token.Id != nil && tokenResp.Id != nil && *token.Id == *tokenResp.Id {
							require.Equal(t, tokenResp.Id, token.Id)
							require.Equal(t, tokenResp.Name, token.Name)
							require.Equal(t, tokenResp.Fingerprint, token.Fingerprint)
							require.Equal(t, tokenResp.Active, token.Active)
							require.Equal(t, tokenResp.CreatedAt, token.CreatedAt)
							require.Equal(t, tokenResp.ValidUntil, token.ValidUntil)
							found = true
							break
						}
					}
					require.True(t, found, "Created token should be present in the list")
				}
			})

			t.Run("Client try to create duplicate access token name", func(t *testing.T) {
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
				err := client.DeleteAccessToken(ctx, &user, tokenResp.Id)
				if err != nil {
					t.Logf("DeleteAccessToken failed: %v", err)
				}
				require.NoError(t, err)
			})

			t.Run("Create Access Token With invalid user", func(t *testing.T) {
				invalidUser := "roothyd"
				tokenName := fmt.Sprintf("Token-%d", time.Now().UnixNano())
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
				tokenName := fmt.Sprintf("Token-%d", time.Now().UnixNano())
				expiresAt := time.Now().Add(5 * time.Minute).Unix()
				req := arangodb.AccessTokenRequest{
					Name:       utils.NewType(tokenName),
					ValidUntil: utils.NewType(expiresAt),
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
