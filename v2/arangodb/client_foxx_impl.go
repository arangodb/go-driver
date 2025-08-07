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

package arangodb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/pkg/errors"
)

var _ ClientFoxx = &clientFoxx{}

type clientFoxx struct {
	client *client
}

func newClientFoxx(client *client) *clientFoxx {
	return &clientFoxx{
		client: client,
	}
}

// InstallFoxxService installs a new service at a given mount path.
func (c *clientFoxx) InstallFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxCreateOptions) error {

	url := connection.NewUrl("_db", dbName, "_api/foxx")
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &InstallFoxxServiceRequest{}
	if opts != nil {
		request.FoxxCreateOptions = *opts
	}

	bytes, err := os.ReadFile(zipFile)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, bytes, request.modifyRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}

}

// UninstallFoxxService uninstalls service at a given mount path.
func (c *clientFoxx) UninstallFoxxService(ctx context.Context, dbName string, opts *FoxxDeleteOptions) error {
	url := connection.NewUrl("_db", dbName, "_api/foxx/service")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &UninstallFoxxServiceRequest{}
	if opts != nil {
		request.FoxxDeleteOptions = *opts
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, nil, request.modifyRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

// GetInstalledFoxxService retrieves the list of Foxx services.
func (c *clientFoxx) GetInstalledFoxxService(ctx context.Context, dbName string, excludeSystem *bool) ([]FoxxServiceObject, error) {
	// Ensure the URL starts with a slash
	urlEndpoint := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "foxx")

	// Append query param if needed
	if excludeSystem != nil {
		urlEndpoint += fmt.Sprintf("?excludeSystem=%t", *excludeSystem)
	}

	// Use json.RawMessage to capture raw response for debugging
	var rawResult json.RawMessage
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &rawResult)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Try to unmarshal as array first
		var result []FoxxServiceObject
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

		// If array unmarshaling fails, try as object with result field
		var objResult struct {
			Result []FoxxServiceObject `json:"result"`
			Error  bool                `json:"error"`
			Code   int                 `json:"code"`
		}

		if err := json.Unmarshal(rawResult, &objResult); err == nil {
			if objResult.Error {
				return nil, fmt.Errorf("ArangoDB API error: code %d", objResult.Code)
			}
			return objResult.Result, nil
		}

		// If both fail, return the unmarshal error
		return nil, fmt.Errorf("cannot unmarshal response into []FoxxServiceObject or object with result field: %s", string(rawResult))
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}
