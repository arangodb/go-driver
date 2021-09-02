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
	"net/http"
)

func NewJWTAuthWrapper(username, password string) Wrapper {
	return WrapAuthentication(func(ctx context.Context, conn Connection) (authentication Authentication, err error) {
		url := NewUrl("_open", "auth")

		var data jwtOpenResponse

		j := jwtOpenRequest{
			Username: username,
			Password: password,
		}

		resp, err := CallPost(ctx, conn, url, &data, j)
		if err != nil {
			return nil, err
		}

		switch resp.Code() {
		case http.StatusOK:
			return NewHeaderAuth("Authorization", "bearer %s", data.Token), nil
		default:
			return nil, NewError(resp.Code(), "unexpected code")
		}
	})
}

type jwtOpenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type jwtOpenResponse struct {
	Token              string `json:"jwt"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}
