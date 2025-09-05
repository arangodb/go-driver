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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	if batchId == "" {
		return errors.New("batchId must be specified for delete batch")
	}
	// Build URL
	url := c.url(dbName, []string{"batch", batchId}, params)

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

func (c *clientReplication) ExtendBatch(ctx context.Context, dbName string, DBserver *string, batchId string, opt CreateNewBatchOptions) error {

	if batchId == "" {
		return errors.New("batchId must be specified for extend batch")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		if DBserver == nil || *DBserver == "" {
			return errors.New("DBserver must be specified when extending a batch on a coordinator")
		}
		queryParams["DBserver"] = *DBserver
	}

	// Build URL
	url := c.url(dbName, []string{"batch", batchId}, queryParams)

	resp, err := connection.CallPut(ctx, c.client.connection, url, nil, opt)
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

func (c *clientReplication) Dump(ctx context.Context, dbName string, params ReplicationDumpParams) ([]byte, error) {

	role, err := c.client.ServerRole(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if role != ServerRoleSingle {
		return nil, errors.Errorf("replication dump not supported on role %s", role)
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if params.ChunkSize != nil && *params.ChunkSize != 0 {
		queryParams["chunkSize"] = params.ChunkSize
	}
	if params.Collection == "" {
		return nil, errors.New("collection must be specified when querying replication dump")
	}
	queryParams["collection"] = params.Collection
	if params.BatchID == "" {
		return nil, errors.New("batchId must be specified when querying replication dump")
	}
	queryParams["batchId"] = params.BatchID

	// Build URL
	url := c.url(dbName, []string{"dump"}, queryParams)
	req, err := c.client.Connection().NewRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	var data []byte
	// Call Do with nil result (we'll handle body manually)
	resp, err := c.client.Connection().Do(ctx, req, &data, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return nil, err
	}
	defer resp.RawResponse().Body.Close()

	if resp.Code() == http.StatusNoContent {
		return nil, nil
	}
	if resp.Code() != http.StatusOK {
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}

	return io.ReadAll(resp.RawResponse().Body)
}

func (c *clientReplication) LoggerState(ctx context.Context, dbName string, DBserver *string) (LoggerStateResponse, error) {
	// Build query params
	queryParams := map[string]interface{}{}
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return LoggerStateResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		if DBserver == nil || *DBserver == "" {
			return LoggerStateResponse{}, errors.New("DBserver must be specified when creating a batch on a coordinator")
		}
		queryParams["DBserver"] = *DBserver
	}
	// Build URL
	url := c.url(dbName, []string{"logger-state"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		LoggerStateResponse   `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return LoggerStateResponse{}, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.LoggerStateResponse, nil
	default:
		return LoggerStateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) LoggerFirstTick(ctx context.Context, dbName string) (LoggerFirstTickResponse, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return LoggerFirstTickResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return LoggerFirstTickResponse{}, errors.New("replication logger-first-tick is not supported on Coordinators")
	}
	// Build URL
	url := c.url(dbName, []string{"logger-first-tick"}, nil)

	var response struct {
		shared.ResponseStruct   `json:",inline"`
		LoggerFirstTickResponse `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return LoggerFirstTickResponse{}, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.LoggerFirstTickResponse, nil
	default:
		return LoggerFirstTickResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) LoggerTickRange(ctx context.Context, dbName string) ([]LoggerTickRangeResponseObj, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return nil, errors.New("replication logger-tick-ranges is not supported on Coordinators")
	}
	// Build URL
	url := c.url(dbName, []string{"logger-tick-ranges"}, nil)

	var response []LoggerTickRangeResponseObj
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

func (c *clientReplication) GetApplierConfig(ctx context.Context, dbName string, global *bool) (ApplierConfigResponse, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierConfigResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierConfigResponse{}, errors.New("replication applier-config is not supported on Coordinators")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if global != nil {
		queryParams["global"] = *global
	}

	// Build URL
	url := c.url(dbName, []string{"applier-config"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierConfigResponse `json:",inline"`
	}
	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ApplierConfigResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierConfigResponse, nil
	default:
		return ApplierConfigResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func formApplierParams(opts ApplierOptions) (map[string]interface{}, error) {
	params := map[string]interface{}{}

	// Required
	if opts.Endpoint == nil || *opts.Endpoint == "" {
		return nil, RequiredFieldError("endpoint")
	}
	params["endpoint"] = *opts.Endpoint

	// Optional
	if opts.Database != nil {
		params["database"] = *opts.Database
	}
	if opts.Username != nil {
		params["username"] = *opts.Username
	}
	if opts.Password != nil {
		params["password"] = *opts.Password
	}
	if opts.MaxConnectRetries != nil {
		params["maxConnectRetries"] = *opts.MaxConnectRetries
	}
	if opts.ConnectTimeout != nil {
		params["connectTimeout"] = *opts.ConnectTimeout
	}
	if opts.RequestTimeout != nil {
		params["requestTimeout"] = *opts.RequestTimeout
	}
	if opts.IdleMinWaitTime != nil {
		params["idleMinWaitTime"] = *opts.IdleMinWaitTime
	}
	if opts.IdleMaxWaitTime != nil {
		params["idleMaxWaitTime"] = *opts.IdleMaxWaitTime
	}
	if opts.InitialSyncMaxWaitTime != nil {
		params["initialSyncMaxWaitTime"] = *opts.InitialSyncMaxWaitTime
	}
	if opts.IncludeSystem != nil {
		params["includeSystem"] = *opts.IncludeSystem
	}
	if opts.ChunkSize != nil {
		params["chunkSize"] = *opts.ChunkSize
	}
	if opts.AutoStart != nil {
		params["autoStart"] = *opts.AutoStart
	}
	if opts.RestrictCollections != nil {
		params["restrictCollections"] = *opts.RestrictCollections
	}
	if opts.RestrictType != nil {
		params["restrictType"] = *opts.RestrictType
	}
	if opts.AdaptivePolling != nil {
		params["adaptivePolling"] = *opts.AdaptivePolling
	}
	if opts.AutoResync != nil {
		params["autoResync"] = *opts.AutoResync
	}
	if opts.AutoResyncRetries != nil {
		params["autoResyncRetries"] = *opts.AutoResyncRetries
	}
	if opts.RequireFromPresent != nil {
		params["requireFromPresent"] = *opts.RequireFromPresent
	}
	if opts.Verbose != nil {
		params["verbose"] = *opts.Verbose
	}

	return params, nil
}

func (c *clientReplication) UpdateApplierConfig(ctx context.Context, dbName string, global *bool, opts ApplierOptions) (ApplierConfigResponse, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierConfigResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierConfigResponse{}, errors.New("replication applier-config is not supported on Coordinators")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if global != nil {
		queryParams["global"] = *global
	}

	// Build URL
	url := c.url(dbName, []string{"applier-config"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierConfigResponse `json:",inline"`
	}

	requestParams, err := formApplierParams(opts)
	if err != nil {
		return ApplierConfigResponse{}, errors.WithStack(err)
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, requestParams)
	if err != nil {
		return ApplierConfigResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierConfigResponse, nil
	default:
		return ApplierConfigResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) ApplierStart(ctx context.Context, dbName string, global *bool, from *string) (ApplierStateResp, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierStateResp{}, errors.New("replication applier-start is not supported on Coordinators")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if global != nil {
		queryParams["global"] = *global
	}
	if from != nil && *from != "" {
		queryParams["from"] = *from
	}

	// Build URL
	url := c.url(dbName, []string{"applier-start"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierStateResp      `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, nil)
	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierStateResp, nil
	default:
		return ApplierStateResp{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) ApplierStop(ctx context.Context, dbName string, global *bool) (ApplierStateResp, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierStateResp{}, errors.New("replication applier-stop is not supported on Coordinators")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if global != nil {
		queryParams["global"] = *global
	}

	// Build URL
	url := c.url(dbName, []string{"applier-stop"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierStateResp      `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, nil)
	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierStateResp, nil
	default:
		return ApplierStateResp{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) GetApplierState(ctx context.Context, dbName string, global *bool) (ApplierStateResp, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierStateResp{}, errors.New("replication applier-stop is not supported on Coordinators")
	}

	// Build query params
	queryParams := map[string]interface{}{}
	if global != nil {
		queryParams["global"] = *global
	}

	// Build URL
	url := c.url(dbName, []string{"applier-state"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierStateResp      `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierStateResp, nil
	default:
		return ApplierStateResp{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) GetReplicationServerId(ctx context.Context, dbName string) (string, error) {

	// Build URL
	url := c.url(dbName, []string{"server-id"}, nil)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ServerId              string `json:"serverId"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ServerId, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) MakeFollower(ctx context.Context, dbName string, opts ApplierOptions) (ApplierStateResp, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ApplierStateResp{}, errors.New("replication make-follower is not supported on Coordinators")
	}

	// Build URL
	url := c.url(dbName, []string{"make-follower"}, nil)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ApplierStateResp      `json:",inline"`
	}
	requestParams, err := formApplierParams(opts)
	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, requestParams)
	if err != nil {
		return ApplierStateResp{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ApplierStateResp, nil
	default:
		return ApplierStateResp{}, response.AsArangoErrorWithCode(code)
	}
}

// RebuildShardRevisionTree triggers a rebuild of the Merkle tree for a specific shard.
// This API must be called directly against a DBServer (not a Coordinator).
func (c *clientReplication) RebuildShardRevisionTree(ctx context.Context, dbName string, shardID ShardID) error {
	// Ensure we are on a DBServer
	role, err := c.client.ServerRole(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	if role != ServerRoleDBServer {
		return fmt.Errorf("rebuild revision tree is only supported on DBServers, got role=%s", role)
	}

	if shardID == "" {
		return RequiredFieldError("shardID")
	}

	// Build URL
	queryParams := map[string]interface{}{
		"collection": shardID,
	}
	url := c.url(dbName, []string{"revisions", "tree"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPost(ctx, c.client.connection, url, &response, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.Code() == http.StatusNoContent {
		return nil
	}
	return response.AsArangoErrorWithCode(resp.Code())
}

func (c *clientReplication) GetShardRevisionTree(ctx context.Context, dbName string, shardID ShardID, batchId string) (json.RawMessage, error) {
	role, err := c.client.ServerRole(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if role != ServerRoleDBServer {
		return nil, fmt.Errorf("get revision tree is only supported on DBServers, got role=%s", role)
	}

	if shardID == "" {
		return nil, RequiredFieldError("shardID")
	}
	if batchId == "" {
		return nil, RequiredFieldError("batchId")
	}

	queryParams := map[string]interface{}{
		"collection": shardID,
		"batchId":    batchId,
	}

	url := c.url(dbName, []string{"revisions", "tree"}, queryParams)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		RevisionTree          json.RawMessage `json:"revisionTree,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.RevisionTree, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) checkRevisionQueryParams(queryParams RevisionQueryParams) (map[string]interface{}, error) {
	params := map[string]interface{}{}
	if queryParams.Collection == "" {
		return nil, RequiredFieldError("collection")
	}
	if queryParams.BatchId == "" {
		return nil, RequiredFieldError("batchId")
	}
	if queryParams.Resume != nil {
		params["resume"] = *queryParams.Resume
	}
	params["collection"] = queryParams.Collection
	params["batchId"] = queryParams.BatchId
	return params, nil
}

func (c *clientReplication) ListDocumentRevisionsInRange(ctx context.Context, dbName string, queryParams RevisionQueryParams, opts [][2]string) ([][2]string, error) {

	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return nil, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return nil, errors.New("replication revisions range is not supported on Coordinators")
	}
	params, err := c.checkRevisionQueryParams(queryParams)
	if err != nil {
		return nil, err
	}
	// Build URL

	url := c.url(dbName, []string{"revisions", "ranges"}, params)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Ranges                [][2]string `json:"ranges,omitempty"`
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Ranges, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) FetchRevisionDocuments(ctx context.Context, dbName string, queryParams RevisionQueryParams, opts []string) ([]map[string]interface{}, error) {

	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return nil, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return nil, errors.New("replication revisions documents is not supported on Coordinators")
	}
	params, err := c.checkRevisionQueryParams(queryParams)
	if err != nil {
		return nil, err
	}

	// Build URL
	url := c.url(dbName, []string{"revisions", "documents"}, params)

	var response []map[string]interface{}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

func (c *clientReplication) formSyncBodyParams(opts ReplicationSyncOptions) (map[string]interface{}, error) {
	params := map[string]interface{}{}
	if opts.Endpoint == "" {
		return nil, RequiredFieldError("endpoint")
	}
	params["endpoint"] = opts.Endpoint
	if opts.Database != nil && *opts.Database != "" {
		params["database"] = *opts.Database
	}
	if opts.Username != "" {
		params["username"] = opts.Username
	}
	if opts.Password != "" {
		params["password"] = opts.Password
	}
	if opts.IncludeSystem != nil {
		params["includeSystem"] = *opts.IncludeSystem
	}
	if opts.Incremental != nil {
		params["incremental"] = *opts.Incremental
	}
	if opts.RestrictType != nil && *opts.RestrictType != "" {
		params["restrictType"] = *opts.RestrictType
	}
	if opts.RestrictCollections != nil && len(*opts.RestrictCollections) > 0 {
		params["restrictCollections"] = *opts.RestrictCollections
	}
	params["initialSyncMaxWaitTime"] = opts.InitialSyncMaxWaitSec
	return params, nil
}

func (c *clientReplication) StartReplicationSync(ctx context.Context, dbName string, opts ReplicationSyncOptions) (ReplicationSyncResult, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return ReplicationSyncResult{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return ReplicationSyncResult{}, errors.New("replication sync is not supported on Coordinators")
	}
	// Form request body params
	body, err := c.formSyncBodyParams(opts)
	if err != nil {
		return ReplicationSyncResult{}, err
	}

	// Build URL
	url := c.url(dbName, []string{"sync"}, nil)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ReplicationSyncResult `json:",inline"`
	}

	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, body)
	if err != nil {
		return ReplicationSyncResult{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ReplicationSyncResult, nil
	default:
		return ReplicationSyncResult{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) GetWALRange(ctx context.Context, dbName string) (WALRangeResponse, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return WALRangeResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return WALRangeResponse{}, errors.New("WAL range is not supported on Coordinators")
	}
	// Build URL
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "wal", "range")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		WALRangeResponse      `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return WALRangeResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.WALRangeResponse, nil
	default:
		return WALRangeResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) GetWALLastTick(ctx context.Context, dbName string) (WALLastTickResponse, error) {
	// Check server role
	serverRole, err := c.client.ServerRole(ctx)

	if err != nil {
		return WALLastTickResponse{}, errors.WithStack(err)
	}
	if serverRole == ServerRoleCoordinator {
		return WALLastTickResponse{}, errors.New("WAL last tick is not supported on Coordinators")
	}
	// Build URL
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "wal", "lastTick")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		WALLastTickResponse   `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return WALLastTickResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.WALLastTickResponse, nil
	default:
		return WALLastTickResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientReplication) formQueryParamsForTail(params *WALTailOptions) map[string]interface{} {
	queryParams := map[string]interface{}{}
	if params == nil {
		return nil
	}
	if params.Global != nil {
		queryParams["global"] = *params.Global
	}
	if params.From != nil {
		queryParams["from"] = *params.From
	}
	if params.To != nil {
		queryParams["to"] = *params.To
	}
	if params.LastScanned != nil {
		queryParams["lastScanned"] = *params.LastScanned
	}
	if params.ChunkSize != nil {
		queryParams["chunkSize"] = *params.ChunkSize
	}
	if params.SyncerId != nil {
		queryParams["syncerId"] = *params.SyncerId
	}
	if params.ServerId != nil {
		queryParams["serverId"] = *params.ServerId
	}
	if params.ClientInfo != nil {
		queryParams["clientInfo"] = *params.ClientInfo
	}
	return queryParams
}

func (c *clientReplication) GetWALTail(ctx context.Context, dbName string, params *WALTailOptions) ([]byte, error) {

	role, err := c.client.ServerRole(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if role == ServerRoleCoordinator {
		return nil, errors.Errorf("replication Tail not supported on role %s", role)
	}

	// Build query params
	queryParams := c.formQueryParamsForTail(params)

	// Build URL
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "wal", "tail")
	req, err := c.client.Connection().NewRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	// Add query params
	for k, v := range queryParams {
		req.AddQuery(k, fmt.Sprintf("%v", v))
	}

	// Use a bytes.Buffer to capture the response
	var buf bytes.Buffer
	resp, err := c.client.Connection().Do(ctx, req, &buf, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return nil, err
	}

	if resp.Code() == http.StatusNoContent {
		return nil, nil
	}

	if resp.Code() != http.StatusOK {
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}

	return buf.Bytes(), nil
}
