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

	if _, err := os.Stat(zipFile); os.IsNotExist(err) {
		return errors.WithStack(err)
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

	if _, err := os.Stat(zipFile); os.IsNotExist(err) {
		return errors.WithStack(err)
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

	if _, err := os.Stat(zipFile); os.IsNotExist(err) {
		return errors.WithStack(err)
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

func (c *clientFoxx) callFoxxServiceAPI(
	ctx context.Context,
	dbName string,
	mount *string,
	path string, // "configuration" or "dependencies"
	method string, // "GET", "PATCH", "PUT"
	body map[string]interface{}, // nil for GET
) (map[string]interface{}, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{path}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage
	var resp connection.Response
	var err error

	switch method {
	case http.MethodGet:
		resp, err = connection.CallGet(ctx, c.client.connection, urlEndpoint, &rawResult)
	case http.MethodPatch:
		resp, err = connection.CallPatch(ctx, c.client.connection, urlEndpoint, &rawResult, body)
	case http.MethodPut:
		resp, err = connection.CallPut(ctx, c.client.connection, urlEndpoint, &rawResult, body)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

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

		return nil, fmt.Errorf(
			"cannot unmarshal response into map or object with result field: %s",
			string(rawResult),
		)
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) GetFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "configuration", http.MethodGet, nil)
}

func (c *clientFoxx) UpdateFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "configuration", http.MethodPatch, opt)
}

func (c *clientFoxx) ReplaceFoxxServiceConfiguration(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "configuration", http.MethodPut, opt)
}

func (c *clientFoxx) GetFoxxServiceDependencies(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "dependencies", http.MethodGet, nil)
}

func (c *clientFoxx) UpdateFoxxServiceDependencies(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "dependencies", http.MethodPatch, opt)
}

func (c *clientFoxx) ReplaceFoxxServiceDependencies(ctx context.Context, dbName string, mount *string, opt map[string]interface{}) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "dependencies", http.MethodPut, opt)
}

func (c *clientFoxx) GetFoxxServiceScripts(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {
	return c.callFoxxServiceAPI(ctx, dbName, mount, "scripts", http.MethodGet, nil)
}

func (c *clientFoxx) RunFoxxServiceScript(ctx context.Context, dbName string, name string, mount *string, body map[string]interface{}) (map[string]interface{}, error) {

	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"scripts", name}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage
	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &rawResult, body)

	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

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

		return nil, fmt.Errorf(
			"cannot unmarshal response into map or object with result field: %s",
			string(rawResult),
		)
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) RunFoxxServiceTests(ctx context.Context, dbName string, opt FoxxTestOptions) (map[string]interface{}, error) {

	if opt.Mount == nil || *opt.Mount == "" {
		return nil, RequiredFieldError("mount")
	}

	queryParams := map[string]interface{}{
		"mount": *opt.Mount,
	}

	if opt.Reporter != nil {
		queryParams["reporter"] = *opt.Reporter
	}
	if opt.Idiomatic != nil {
		queryParams["idiomatic"] = *opt.Idiomatic
	}
	if opt.Filter != nil {
		queryParams["filter"] = *opt.Filter
	}

	urlEndpoint := c.url(dbName, []string{"tests"}, queryParams)

	var rawResult json.RawMessage
	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &rawResult, nil)

	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

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

		return nil, fmt.Errorf(
			"cannot unmarshal response into map or object with result field: %s",
			string(rawResult),
		)
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) EnableDevelopmentMode(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {

	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"development"}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage
	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &rawResult, nil)

	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

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

		return nil, fmt.Errorf(
			"cannot unmarshal response into map or object with result field: %s",
			string(rawResult),
		)
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) DisableDevelopmentMode(ctx context.Context, dbName string, mount *string) (map[string]interface{}, error) {

	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"development"}, map[string]interface{}{
		"mount": *mount,
	})

	var rawResult json.RawMessage
	resp, err := connection.CallDelete(ctx, c.client.connection, urlEndpoint, &rawResult)

	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var result map[string]interface{}
		if err := json.Unmarshal(rawResult, &result); err == nil {
			return result, nil
		}

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

		return nil, fmt.Errorf(
			"cannot unmarshal response into map or object with result field: %s",
			string(rawResult),
		)
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) GetFoxxServiceReadme(ctx context.Context, dbName string, mount *string) ([]byte, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"readme"}, map[string]interface{}{
		"mount": *mount,
	})

	// Create request
	req, err := c.client.Connection().NewRequest(http.MethodGet, urlEndpoint)
	if err != nil {
		return nil, err
	}
	var data []byte
	// Call Do with nil result (we'll handle body manually)
	resp, err := c.client.Connection().Do(ctx, req, &data, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return nil, err
	}

	switch resp.Code() {
	case http.StatusOK:
		return data, nil
	case http.StatusNoContent:
		return nil, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

func (c *clientFoxx) GetFoxxServiceSwagger(ctx context.Context, dbName string, mount *string) (SwaggerResponse, error) {
	if mount == nil || *mount == "" {
		return SwaggerResponse{}, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"swagger"}, map[string]interface{}{
		"mount": *mount,
	})

	var result struct {
		shared.ResponseStruct `json:",inline"`
		SwaggerResponse       `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &result)
	if err != nil {
		return SwaggerResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return result.SwaggerResponse, nil
	default:
		return SwaggerResponse{}, result.AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) CommitFoxxService(ctx context.Context, dbName string, replace *bool) error {
	queryParams := make(map[string]interface{})
	if replace != nil {
		queryParams["replace"] = *replace
	}

	urlEndpoint := c.url(dbName, []string{"commit"}, queryParams)

	var result struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &result, nil)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusNoContent: // 204 expected
		return nil
	default:
		return result.AsArangoErrorWithCode(code)
	}
}

func (c *clientFoxx) DownloadFoxxServiceBundle(ctx context.Context, dbName string, mount *string) ([]byte, error) {
	if mount == nil || *mount == "" {
		return nil, RequiredFieldError("mount")
	}

	urlEndpoint := c.url(dbName, []string{"download"}, map[string]interface{}{
		"mount": *mount,
	})
	// Create request
	req, err := c.client.Connection().NewRequest(http.MethodPost, urlEndpoint)
	if err != nil {
		return nil, err
	}
	var data []byte
	// Call Do with nil result (we'll handle body manually)
	resp, err := c.client.Connection().Do(ctx, req, &data, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return nil, err
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return data, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}
