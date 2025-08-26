//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package connection

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func NewJWTAuthWrapper(username, password string) Wrapper {
	var token string
	var expiry time.Time

	refresh := func(ctx context.Context, conn Connection) error {
		url := NewUrl("_open", "auth")

		var data jwtOpenResponse

		j := jwtOpenRequest{
			Username: username,
			Password: password,
		}

		resp, err := CallPost(ctx, conn, url, &data, j)
		if err != nil {
			return err
		}
		if resp.Code() != http.StatusOK {
			return NewError(resp.Code(), "unexpected code")
		}

		token = data.Token
		expiry, err = parseJWTExpiry(token)
		if err != nil {
			// Log for visibility but don't break functionality
			log.Printf("failed to parse JWT expiry: %v", err)
			expiry = time.Now().Add(1 * time.Minute) // fallback, so it will refresh immediately next time
		}
		return nil
	}

	return WrapAuthentication(func(ctx context.Context, conn Connection) (Authentication, error) {
		// First time fetch
		if token == "" || time.Now().After(expiry) {
			if err := refresh(ctx, conn); err != nil {
				return nil, err
			}
		}

		return NewHeaderAuth("Authorization", "bearer %s", token), nil
	})
}

type jwtOpenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type jwtOpenResponse struct {
	Token              string `json:"jwt"`
	ExpiresIn          int    `json:"expires_in,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}

func parseJWTExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, err
	}

	return time.Unix(claims.Exp, 0), nil
}

func NewSSOAuthWrapper(initialToken string) Wrapper {
	var token = initialToken
	var expiry time.Time
	// setToken updates the current JWT and its expiry time.
	// If expiry parsing fails, we log the error and fall back to a short 1-minute lifetime.
	// This ensures the token will be refreshed soon without breaking functionality.
	setToken := func(newToken string) {
		token = newToken
		expiryTime, err := parseJWTExpiry(newToken)
		if err != nil {
			// Log for visibility but don't break functionality
			log.Printf("failed to parse JWT expiry: %v", err)
			expiry = time.Now().Add(1 * time.Minute) // fallback, so it will refresh immediately next time
		} else {
			expiry = expiryTime
		}
	}

	// If we already have a token (from an SSO login), parse expiry now
	if token != "" {
		setToken(token)
	}

	return WrapAuthentication(func(ctx context.Context, conn Connection) (Authentication, error) {
		// No token yet or expired — let caller know they must login via SSO
		if token == "" || time.Now().After(expiry) {
			// Try a call to _open/auth just to see if server sends 307
			url := NewUrl("_open", "auth")
			var data jwtOpenResponse
			// Intentionally passing nil: in SSO mode, /_open/auth expects no body
			resp, err := CallPost(ctx, conn, url, &data, nil)
			if err != nil {
				return nil, err
			}

			switch resp.Code() {
			case http.StatusOK:
				setToken(data.Token)
			case http.StatusTemporaryRedirect:
				loc := resp.Header("Location")
				return nil, fmt.Errorf("SSO redirect: please authenticate via browser at %s", loc)
			default:
				return nil, NewError(resp.Code(), "unexpected code")
			}
		}

		return NewHeaderAuth("Authorization", "bearer %s", token), nil
	})
}
