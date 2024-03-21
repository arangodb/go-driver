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
