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

import "context"

// ClientAccessTokens defines access token API methods.
type ClientAccessTokens interface {
	// CreateAccessToken creates a new access token for the specified user.
	// Permissions:
	//   - You can always create an access token for yourself.
	//   - To create a token for another user, you need admin access
	//     to the _system database.
	CreateAccessToken(ctx context.Context, user *string, req AccessTokenRequest) (CreateAccessTokenResponse, error)

	DeleteAccessToken(ctx context.Context, user *string, tokenId *int) error

	GetAllAccessToken(ctx context.Context, user *string) (AccessTokenResponse, error)
}

// AccessTokenRequest represents the input required to create a new access token.
type AccessTokenRequest struct {
	// Name is a descriptive name for the access token (e.g., "Token for Service A").
	// This helps identify the token later. Required field.
	Name *string `json:"name,omitempty"`

	// ValidUntil is the Unix timestamp (seconds since epoch) until which the token remains valid.
	// Required field. After this time, the token will automatically expire.
	ValidUntil *int64 `json:"valid_until,omitempty"`
}

// AccessTokenInfo contains metadata about an access token.
// Embeds AccessTokenRequest to include the name and validity period in the response.
type AccessTokenInfo struct {
	// Id is the unique identifier for the access token.
	Id *int `json:"id,omitempty"`

	// Embed the AccessTokenRequest fields (Name and ValidUntil) for reference.
	AccessTokenRequest

	// CreatedAt is the Unix timestamp when the token was created.
	CreatedAt *int64 `json:"created_at,omitempty"`

	// Fingerprint is a unique string associated with the token,
	// useful for tracking or verifying the token without revealing it.
	Fingerprint *string `json:"fingerprint,omitempty"`

	// Active indicates whether the token is currently active or has been revoked/expired.
	Active *bool `json:"active,omitempty"`
}

// CreateAccessTokenResponse represents the response returned when creating a new access token.
type CreateAccessTokenResponse struct {
	// Embed the AccessTokenInfo metadata.
	AccessTokenInfo

	// Token is the actual access token string.
	// It is only returned once at creation and should be stored securely.
	Token *string `json:"token,omitempty"`
}

// AccessTokenResponse represents a list of access tokens for a user.
type AccessTokenResponse struct {
	// Tokens is a list of all access tokens associated with a user.
	Tokens []AccessTokenInfo `json:"tokens,omitempty"`
}
