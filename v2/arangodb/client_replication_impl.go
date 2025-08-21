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
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

type clientReplication struct {
	client *client
}

func newClientReplication(client *client) *clientReplication {
	return &clientReplication{
		client: client,
	}
}

var _ ClientReplication = &clientReplication{}

func (c *clientReplication) url(dbName string, pathSegments []string, queryParams map[string]interface{}) string {

	base := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "replication")
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

func (c *clientReplication) CreateNewBatch(ctx context.Context, dbName string, DBserver *string, state *bool, opt CreateNewBatchOptions) (CreateNewBatchResponse, error) {
	// Build query params
	queryParams := map[string]interface{}{}
	if state != nil {
		queryParams["state"] = *state
	}

	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return CreateNewBatchResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		if DBserver == nil || *DBserver == "" {
			return CreateNewBatchResponse{}, errors.New("DBserver must be specified when creating a batch on a coordinator")
		}
		queryParams["DBserver"] = *DBserver
	}

	// Build URL
	url := c.url(dbName, []string{"batch"}, queryParams)

	// Prepare response wrapper
	var response struct {
		shared.ResponseStruct  `json:",inline"`
		CreateNewBatchResponse `json:",inline"`
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, opt)
	if err != nil {
		return CreateNewBatchResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.CreateNewBatchResponse, nil
	default:
		return CreateNewBatchResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) GetInventory(ctx context.Context, dbName string, params InventoryQueryParams) (InventoryResponse, error) {
	// Build query params
	queryParams := map[string]interface{}{}

	if params.IncludeSystem == nil {
		queryParams["includeSystem"] = utils.NewType(true)
	} else {
		queryParams["includeSystem"] = *params.IncludeSystem
	}

	if params.Global == nil {
		queryParams["global"] = utils.NewType(false)
	} else {
		queryParams["global"] = *params.Global
	}

	if params.BatchID == "" {
		return InventoryResponse{}, errors.New("batchId must be specified when querying inventory")
	}
	queryParams["batchId"] = params.BatchID

	if params.Collection != nil {
		queryParams["collection"] = *params.Collection
	}

	// Check server role
	serverRole, err := c.client.ServerRole(ctx)
	if err != nil {
		return InventoryResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		if params.DBserver == nil || *params.DBserver == "" {
			return InventoryResponse{}, errors.New("DBserver must be specified when querying inventory on a coordinator")
		}
		queryParams["DBserver"] = *params.DBserver
	}

	// Build URL
	url := c.url(dbName, []string{"inventory"}, queryParams)

	// Prepare response wrapper
	var response struct {
		shared.ResponseStruct `json:",inline"`
		InventoryResponse     `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return InventoryResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.InventoryResponse, nil
	default:
		return InventoryResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) DeleteBatch(ctx context.Context, dbName string, DBserver *string, batchId string) error {
	params := map[string]interface{}{}
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		if DBserver == nil || *DBserver == "" {
			return errors.New("DBserver must be specified when querying inventory on a coordinator")
		}
		params["DBserver"] = *DBserver
	}

	// Build URL
	url := c.url(dbName, []string{"batch", batchId}, params)

	// Prepare response wrapper
	// var response shared.ResponseStruct
	resp, err := connection.CallDelete(ctx, c.client.connection, url, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusNoContent:
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}
