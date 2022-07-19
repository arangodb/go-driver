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

package arangodb

import (
	"context"
	"net/http"

	"github.com/arangodb/go-driver/v2/arangodb/shared"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/connection"
)

func newClientServerInfo(client *client) *clientServerInfo {
	return &clientServerInfo{
		client: client,
	}
}

var _ ClientServerInfo = &clientServerInfo{}

type clientServerInfo struct {
	client *client
}

func (c clientServerInfo) Version(ctx context.Context) (VersionInfo, error) {
	url := connection.NewUrl("_api", "version")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		VersionInfo
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return VersionInfo{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.VersionInfo, nil
	default:
		return VersionInfo{}, response.AsArangoErrorWithCode(code)
	}
}
