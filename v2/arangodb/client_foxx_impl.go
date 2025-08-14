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

func (c *clientFoxx) url(dbName string, pathSegments []string, queryParams map[string]interface{}) string {

	base := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "foxx")
	for _, seg := range pathSegments {
		base = fmt.Sprintf("%s/%s", base, url.PathEscape(seg))
	}

	if len(queryParams) > 0 {
		q := url.Values{}
		for k, v := range queryParams {
			switch val := v.(type) {
			case string:
				q.Set(k, val)
			case bool:
				q.Set(k, fmt.Sprintf("%t", val))
			case int, int64, float64:
				q.Set(k, fmt.Sprintf("%v", val))
			default:
				// skip unsupported types or handle as needed
			}
		}
		base = fmt.Sprintf("%s?%s", base, q.Encode())
	}
	return base
}

// InstallFoxxService installs a new service at a given mount path.
func (c *clientFoxx) InstallFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxDeploymentOptions) error {

	url := connection.NewUrl("_db", dbName, "_api/foxx")
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &DeployFoxxServiceRequest{}
	if opts != nil {
		request.FoxxDeploymentOptions = *opts
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
	case http.StatusCreated:
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

	resp, err := connection.CallDelete(ctx, c.client.connection, url, &response, request.modifyRequest)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusNoContent:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

// ListInstalledFoxxServices retrieves the list of Foxx services.
func (c *clientFoxx) ListInstalledFoxxServices(ctx context.Context, dbName string, excludeSystem *bool) ([]FoxxServiceListItem, error) {
	query := map[string]interface{}{}
	// query params
	if excludeSystem != nil {
		query["excludeSystem"] = *excludeSystem
	}

	urlEndpoint := c.url(dbName, nil, query)
	// Use json.RawMessage to capture raw response for debugging
	var rawResult json.RawMessage
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &rawResult)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Try to unmarshal as array first
		var result []FoxxServiceListItem
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

		// If array unmarshaling fails, try as object with result field
		var objResult struct {
			Result []FoxxServiceListItem `json:"result"`
			Error  bool                  `json:"error"`
			Code   int                   `json:"code"`
		}

		if err := json.Unmarshal(rawResult, &objResult); err == nil {
			if objResult.Error {
				return nil, fmt.Errorf("ArangoDB API error: code %d", objResult.Code)
			}
			return objResult.Result, nil
		}

		// If both fail, return the unmarshal error
		return nil, fmt.Errorf("cannot unmarshal response into []FoxxServiceListItem or object with result field: %s", string(rawResult))
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

// GetInstalledFoxxService retrieves detailed information about a specific Foxx service
func (c *clientFoxx) GetInstalledFoxxService(ctx context.Context, dbName string, mount *string) (FoxxServiceObject, error) {

	if mount == nil || *mount == "" {
		return FoxxServiceObject{}, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"service"}, map[string]interface{}{
		"mount": *mount,
	})

	// Use json.RawMessage to capture raw response for debugging
	var result struct {
		shared.ResponseStruct `json:",inline"`
		FoxxServiceObject     `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &result)
	if err != nil {
		return FoxxServiceObject{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return result.FoxxServiceObject, nil
	default:
		return FoxxServiceObject{}, result.AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) ReplaceFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxDeploymentOptions) error {

	// url := connection.NewUrl("_db", dbName, "_api/foxx/service")
	url := c.url(dbName, []string{"service"}, nil)
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &DeployFoxxServiceRequest{}
	if opts != nil {
		request.FoxxDeploymentOptions = *opts
	}

	bytes, err := os.ReadFile(zipFile)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, bytes, request.modifyRequest)
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

func (c *clientFoxx) UpgradeFoxxService(ctx context.Context, dbName string, zipFile string, opts *FoxxDeploymentOptions) error {

	// url := connection.NewUrl("_db", dbName, "_api/foxx/service")
	url := c.url(dbName, []string{"service"}, nil)
	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	request := &DeployFoxxServiceRequest{}
	if opts != nil {
		request.FoxxDeploymentOptions = *opts
	}

	bytes, err := os.ReadFile(zipFile)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := connection.CallPatch(ctx, c.client.connection, url, &response, bytes, request.modifyRequest)
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

// GetFoxxServiceConfiguration retrieves the configuration for a specific Foxx service.
func (c *clientFoxx) GetFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"configuration"}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &rawResult)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Try to unmarshal as array first
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

		// If array unmarshaling fails, try as object with result field
		var objResult struct {
			Result map[string]interface{} `json:"result"`
			Error  bool                   `json:"error"`
			Code   int                    `json:"code"`
		}

		if err := json.Unmarshal(rawResult, &objResult); err == nil {
			if objResult.Error {
				return nil, fmt.Errorf("ArangoDB API error: code %d", objResult.Code)
			}
			return objResult.Result, nil
		}

		// If both fail, return the unmarshal error
		return nil, fmt.Errorf("cannot unmarshal response into []FoxxServiceListItem or object with result field: %s", string(rawResult))
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) UpdateFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"configuration"}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage

	resp, err := connection.CallPatch(ctx, c.client.connection, urlEndpoint, &rawResult, opt)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Try to unmarshal as array first
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

		// If array unmarshaling fails, try as object with result field
		var objResult struct {
			Result map[string]interface{} `json:"result"`
			Error  bool                   `json:"error"`
			Code   int                    `json:"code"`
		}

		if err := json.Unmarshal(rawResult, &objResult); err == nil {
			if objResult.Error {
				return nil, fmt.Errorf("ArangoDB API error: code %d", objResult.Code)
			}
			return objResult.Result, nil
		}

		// If both fail, return the unmarshal error
		return nil, fmt.Errorf("cannot unmarshal response into []FoxxServiceListItem or object with result field: %s", string(rawResult))
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) ReplaceFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"configuration"}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage

	resp, err := connection.CallPut(ctx, c.client.connection, urlEndpoint, &rawResult, opt)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		// Try to unmarshal as array first
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

		// If array unmarshaling fails, try as object with result field
		var objResult struct {
			Result map[string]interface{} `json:"result"`
			Error  bool                   `json:"error"`
			Code   int                    `json:"code"`
		}

		if err := json.Unmarshal(rawResult, &objResult); err == nil {
			if objResult.Error {
				return nil, fmt.Errorf("ArangoDB API error: code %d", objResult.Code)
			}
			return objResult.Result, nil
		}

		// If both fail, return the unmarshal error
		return nil, fmt.Errorf("cannot unmarshal response into []FoxxServiceListItem or object with result field: %s", string(rawResult))
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}
