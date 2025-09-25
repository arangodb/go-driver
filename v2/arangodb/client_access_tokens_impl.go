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
//

package arangodb

import (
	"context"
	"net/http"
	"strconv"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/pkg/errors"
)

type clientAccessTokens struct {
	client *client
}

func newClientAccessTokens(client *client) *clientAccessTokens {
	return &clientAccessTokens{
		client: client,
	}
}

var _ ClientAccessTokens = &clientAccessTokens{}

func validateAccessTokenReqParams(req AccessTokenRequest) (map[string]interface{}, error) {
	reqParams := make(map[string]interface{})
	if req.Name == nil {
		return nil, RequiredFieldError("name")
	}
	if req.ValidUntil == nil {
		return nil, RequiredFieldError("valid_until")
	}
	reqParams["name"] = *req.Name
	reqParams["valid_until"] = *req.ValidUntil
	return reqParams, nil
}

// CreateAccessToken creates a new access token for the specified user.
// Permissions:
//   - You can always create an access token for yourself.
//   - To create a token for another user, you need admin access
//     to the _system database.
func (c *clientAccessTokens) CreateAccessToken(ctx context.Context, user *string, req AccessTokenRequest) (CreateAccessTokenResponse, error) {
	if user == nil {
		return CreateAccessTokenResponse{}, RequiredFieldError("user")
	}
	// Build the URL for the JWT secrets endpoint, safely escaping the database name
	url := connection.NewUrl("_api", "token", *user)

	var response struct {
		shared.ResponseStruct     `json:",inline"`
		CreateAccessTokenResponse `json:",inline"`
	}

	reqParams, err := validateAccessTokenReqParams(req)
	if err != nil {
		return CreateAccessTokenResponse{}, errors.WithStack(err)
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, reqParams)
	if err != nil {
		return CreateAccessTokenResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CreateAccessTokenResponse, nil
	default:
		return CreateAccessTokenResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// DeleteAccessToken deletes a specific access token for a given user.
func (c *clientAccessTokens) DeleteAccessToken(ctx context.Context, user *string, tokenId *int) error {
	if user == nil {
		return RequiredFieldError("user")
	}
	if tokenId == nil {
		return RequiredFieldError("token-id")
	}
	// Build the URL for the JWT secrets endpoint, safely escaping the database name
	url := connection.NewUrl("_api", "token", *user, strconv.Itoa(*tokenId))

	resp, err := connection.CallDelete(ctx, c.client.connection, url, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

// GetAllAccessToken retrieves all access tokens for a given user.
func (c *clientAccessTokens) GetAllAccessToken(ctx context.Context, user *string) (AccessTokenResponse, error) {
	if user == nil {
		return AccessTokenResponse{}, RequiredFieldError("user")
	}
	// Build the URL for the JWT secrets endpoint, safely escaping the database name
	url := connection.NewUrl("_api", "token", *user)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		AccessTokenResponse   `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return AccessTokenResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.AccessTokenResponse, nil
	default:
		return AccessTokenResponse{}, response.AsArangoErrorWithCode(code)
	}
}
