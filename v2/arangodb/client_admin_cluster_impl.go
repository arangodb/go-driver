//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func (c *clientAdmin) Health(ctx context.Context) (ClusterHealth, error) {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "health")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ClusterHealth         `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return ClusterHealth{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ClusterHealth, nil
	default:
		return ClusterHealth{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) DatabaseInventory(ctx context.Context, dbName string) (DatabaseInventory, error) {
	urlEndpoint := connection.NewUrl("_db", url.PathEscape(dbName), "_api", "replication", "clusterInventory")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		DatabaseInventory     `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return DatabaseInventory{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.DatabaseInventory, nil
	default:
		return DatabaseInventory{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) MoveShard(ctx context.Context, col Collection, shard ShardID, fromServer, toServer ServerID) (string, error) {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "moveShard")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		JobID                 string `json:"id"`
	}

	body := struct {
		Database   string   `json:"database"`
		Collection string   `json:"collection"`
		Shard      ShardID  `json:"shard"`
		FromServer ServerID `json:"fromServer"`
		ToServer   ServerID `json:"toServer"`
	}{
		Database:   col.Database().Name(),
		Collection: col.Name(),
		Shard:      shard,
		FromServer: fromServer,
		ToServer:   toServer,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted:
		return response.JobID, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) CleanOutServer(ctx context.Context, serverID ServerID) (string, error) {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "cleanOutServer")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		JobID                 string `json:"id"`
	}

	body := struct {
		Server ServerID `json:"server"`
	}{
		Server: serverID,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return response.JobID, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) ResignServer(ctx context.Context, serverID ServerID) (string, error) {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "resignLeadership")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		JobID                 string `json:"id"`
	}

	body := struct {
		Server ServerID `json:"server"`
	}{
		Server: serverID,
	}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return response.JobID, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) NumberOfServers(ctx context.Context) (NumberOfServersResponse, error) {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "numberOfServers")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                NumberOfServersResponse `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return NumberOfServersResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return NumberOfServersResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) IsCleanedOut(ctx context.Context, serverID ServerID) (bool, error) {
	r, err := c.NumberOfServers(ctx)
	if err != nil {
		return false, errors.WithStack(err)
	}

	for _, id := range r.CleanedServerIDs {
		if id == serverID {
			return true, nil
		}
	}
	return false, nil
}

func (c *clientAdmin) RemoveServer(ctx context.Context, serverID ServerID) error {
	urlEndpoint := connection.NewUrl("_admin", "cluster", "removeServer")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, serverID)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

// ClusterStatistics retrieves statistical information from a specific DBServer
// in an ArangoDB cluster. The statistics include system, client, HTTP, and server
// metrics such as CPU usage, memory, connections, requests, and transaction details.
func (c *clientAdmin) ClusterStatistics(ctx context.Context, dbServer string) (ClusterStatisticsResponse, error) {
	if dbServer == "" {
		return ClusterStatisticsResponse{}, RequiredFieldError("dbServer")
	}
	// Form URL
	urlEndpoint := connection.NewUrl("_admin", "cluster", "statistics")

	var response struct {
		shared.ResponseStruct     `json:",inline"`
		ClusterStatisticsResponse `json:",inline"`
	}

	//Adding request params
	var mod []connection.RequestModifier
	mod = append(mod, connection.WithQuery("DBserver", dbServer))
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response, mod...)
	if err != nil {
		return ClusterStatisticsResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ClusterStatisticsResponse, nil
	default:
		return ClusterStatisticsResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// ClusterEndpoints returns the endpoints of a cluster.
func (c *clientAdmin) ClusterEndpoints(ctx context.Context) (ClusterEndpointsResponse, error) {
	url := connection.NewUrl("_api", "cluster", "endpoints")

	var response struct {
		shared.ResponseStruct    `json:",inline"`
		ClusterEndpointsResponse `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ClusterEndpointsResponse{}, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ClusterEndpointsResponse, nil
	default:
		return ClusterEndpointsResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// GetClusterMaintenance retrieves the maintenance status of a given DB-Server.
// It checks whether the specified DB-Server is in maintenance mode and,
// if so, until what date and time (in ISO 8601 format) the maintenance will last.
func (c *clientAdmin) GetClusterMaintenance(ctx context.Context, dbServer string) (ClusterMaintenanceResponse, error) {
	if dbServer == "" {
		return ClusterMaintenanceResponse{}, RequiredFieldError("dbServer")
	}

	urlEndpoint := connection.NewUrl("_admin", "cluster", "maintenance", dbServer)

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Result                ClusterMaintenanceResponse `json:"result"`
	}

	// Perform GET request
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return ClusterMaintenanceResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return ClusterMaintenanceResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// SetDBServerMaintenance sets the maintenance mode for a specific DB-Server.
// This endpoint affects only the given DB-Server. When in maintenance mode,
// the server is excluded from supervision actions such as shard distribution
// or failover. This is typically used during planned restarts or upgrades.
func (c *clientAdmin) SetDBServerMaintenance(ctx context.Context, dbServer string, opts *ClusterMaintenanceOpts) error {
	if dbServer == "" {
		return RequiredFieldError("dbServer")
	}

	if opts == nil {
		return RequiredFieldError("opts")
	}
	if opts.Mode == "" {
		return RequiredFieldError("mode")
	}

	// Build request body with optional timeout
	body := ClusterMaintenanceOpts{
		Mode: opts.Mode,
	}
	if opts.Timeout != nil {
		body.Timeout = opts.Timeout
	}

	urlEndpoint := connection.NewUrl("_admin", "cluster", "maintenance", dbServer)

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	// Perform PUT request
	resp, err := connection.CallPut(ctx, c.client.connection, urlEndpoint, &response, body)
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

// SetClusterMaintenance sets the cluster-wide supervision maintenance mode.
// This endpoint affects the supervision (Agency) component of the cluster.
// While enabled, automatic failovers, shard movements, and repair jobs
// are suspended. The mode can be:
//
//   - "on":   Enable maintenance mode for the default 60 minutes.
//   - "off":  Disable maintenance mode immediately.
//   - "<number>":  Enable maintenance mode for <number> seconds.
//
// Be aware that no automatic failovers of any kind will take place while
// the maintenance mode is enabled. The supervision will reactivate itself
// automatically after the duration expires.
func (c *clientAdmin) SetClusterMaintenance(ctx context.Context, mode string) error {

	if mode == "" {
		return RequiredFieldError("mode")
	}

	urlEndpoint := connection.NewUrl("_admin", "cluster", "maintenance")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	// Perform PUT request
	resp, err := connection.CallPut(ctx, c.client.connection, urlEndpoint, &response, mode)
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
