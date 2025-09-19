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
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

// ServerStatus describes the health status of a server
type ServerStatus string

const (
	// ServerStatusGood indicates server is in good state
	ServerStatusGood ServerStatus = "GOOD"

	// ServerStatusBad indicates server has missed 1 heartbeat
	ServerStatusBad ServerStatus = "BAD"

	// ServerStatusFailed indicates server has been declared failed by the supervision, this happens after about 15s being bad.
	ServerStatusFailed ServerStatus = "FAILED"
)

// ServerSyncStatus describes the servers sync status
type ServerSyncStatus string

const (
	ServerSyncStatusUnknown   ServerSyncStatus = "UNKNOWN"
	ServerSyncStatusUndefined ServerSyncStatus = "UNDEFINED"
	ServerSyncStatusStartup   ServerSyncStatus = "STARTUP"
	ServerSyncStatusStopping  ServerSyncStatus = "STOPPING"
	ServerSyncStatusStopped   ServerSyncStatus = "STOPPED"
	ServerSyncStatusServing   ServerSyncStatus = "SERVING"
	ServerSyncStatusShutdown  ServerSyncStatus = "SHUTDOWN"
)

// ClusterHealth contains health information for all servers in a cluster.
type ClusterHealth struct {
	// Unique identifier of the entire cluster.
	// This ID is created when the cluster was first created.
	ID string `json:"ClusterId"`

	// Health per server
	Health map[ServerID]ServerHealth `json:"Health"`
}

// ServerHealth contains health information of a single server in a cluster.
type ServerHealth struct {
	Endpoint            string           `json:"Endpoint"`
	LastHeartbeatAcked  time.Time        `json:"LastHeartbeatAcked"`
	LastHeartbeatSent   time.Time        `json:"LastHeartbeatSent"`
	LastHeartbeatStatus string           `json:"LastHeartbeatStatus"`
	Role                ServerRole       `json:"Role"`
	ShortName           string           `json:"ShortName"`
	Status              ServerStatus     `json:"Status"`
	CanBeDeleted        bool             `json:"CanBeDeleted"`
	HostID              string           `json:"Host,omitempty"`
	Version             Version          `json:"Version,omitempty"`
	Engine              EngineType       `json:"Engine,omitempty"`
	SyncStatus          ServerSyncStatus `json:"SyncStatus,omitempty"`

	// Only for Coordinators
	AdvertisedEndpoint *string `json:"AdvertisedEndpoint,omitempty"`

	// Only for Agents
	Leader  *string `json:"Leader,omitempty"`
	Leading *bool   `json:"Leading,omitempty"`
}

type ServerMode string

const (
	// ServerModeDefault is the normal mode of the database in which read and write requests
	// are allowed.
	ServerModeDefault ServerMode = "default"
	// ServerModeReadOnly is the mode in which all modifications to th database are blocked.
	// Behavior is the same as user that has read-only access to all databases & collections.
	ServerModeReadOnly ServerMode = "readonly"
)

type clientAdmin struct {
	client *client
}

func newClientAdmin(client *client) *clientAdmin {
	return &clientAdmin{
		client: client,
	}
}

var _ ClientAdmin = &clientAdmin{}

func (c *clientAdmin) ServerMode(ctx context.Context) (ServerMode, error) {
	url := connection.NewUrl("_admin", "server", "mode")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Mode                  ServerMode `json:"mode,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Mode, nil
	default:
		return "", response.AsArangoErrorWithCode(code)
	}
}

func (c *clientAdmin) SetServerMode(ctx context.Context, mode ServerMode) error {
	url := connection.NewUrl("_admin", "server", "mode")

	reqBody := struct {
		Mode ServerMode `json:"mode"`
	}{
		Mode: mode,
	}

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}
	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, reqBody)
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

func (c *clientAdmin) CheckAvailability(ctx context.Context, serverEndpoint string) error {
	url := connection.NewUrl("_admin", "server", "availability")

	req, err := c.client.Connection().NewRequestWithEndpoint(utils.FixupEndpointURLScheme(serverEndpoint), http.MethodGet, url)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = c.client.Connection().Do(ctx, req, nil, http.StatusOK)
	return errors.WithStack(err)
}

// GetSystemTime returns the current system time as a Unix timestamp with microsecond precision
func (c *clientAdmin) GetSystemTime(ctx context.Context, dbName string) (float64, error) {
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_admin", "time")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		Time                  float64 `json:"time,omitempty"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Time, nil
	default:
		return 0, response.AsArangoErrorWithCode(code)
	}
}

// GetServerStatus returns status information about the server
func (c *clientAdmin) GetServerStatus(ctx context.Context, dbName string) (ServerStatusResponse, error) {
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_admin", "status")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ServerStatusResponse  `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return ServerStatusResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.ServerStatusResponse, nil
	default:
		return ServerStatusResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// GetDeploymentSupportInfo retrieves deployment information for support purposes.
func (c *clientAdmin) GetDeploymentSupportInfo(ctx context.Context) (SupportInfoResponse, error) {
	url := connection.NewUrl("_db", "_system", "_admin", "support-info")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		SupportInfoResponse   `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return SupportInfoResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.SupportInfoResponse, nil
	default:
		return SupportInfoResponse{}, response.AsArangoErrorWithCode(code)
	}
}

// GetStartupConfiguration returns the effective configuration of the queried arangod instance.
func (c *clientAdmin) GetStartupConfiguration(ctx context.Context) (map[string]interface{}, error) {
	url := connection.NewUrl("_db", "_system", "_admin", "options")

	var response map[string]interface{}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

// GetStartupConfigurationDescription fetches the available startup configuration
// options of the queried arangod instance.
func (c *clientAdmin) GetStartupConfigurationDescription(ctx context.Context) (map[string]interface{}, error) {
	url := connection.NewUrl("_db", "_system", "_admin", "options-description")

	var response map[string]interface{}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

// ReloadRoutingTable reloads the routing information from the _routing system collection.
func (c *clientAdmin) ReloadRoutingTable(ctx context.Context, dbName string) error {
	urlEndpoint := connection.NewUrl("_db", url.PathEscape(dbName), "_admin", "routing", "reload")

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, nil, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusNoContent:
		return nil
	default:
		return (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

// ExecuteAdminScript executes JavaScript code on the server.
// Note: Requires ArangoDB to be started with --javascript.allow-admin-execute enabled.
func (c *clientAdmin) ExecuteAdminScript(ctx context.Context, dbName string, script *string) (interface{}, error) {
	url := connection.NewUrl("_db", url.PathEscape(dbName), "_admin", "execute")

	req, err := c.client.Connection().NewRequest("POST", url)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if script == nil {
		return nil, RequiredFieldError("script")
	}
	if err := req.SetBody(*script); err != nil {
		return nil, errors.WithStack(err)
	}
	var response interface{}
	resp, err := c.client.Connection().Do(ctx, req, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}

// CompactDatabases can be used to reclaim disk space after substantial data deletions have taken place,
// by compacting the entire database system data.
// The endpoint requires superuser access.
func (c *clientAdmin) CompactDatabases(ctx context.Context, opts *CompactOpts) (map[string]interface{}, error) {
	url := connection.NewUrl("_admin", "compact")

	// In client_admin_impl.go, consider this cleaner approach:
	var modifyRequest []connection.RequestModifier

	// Always add both parameters with appropriate defaults
	changeLevel := false
	compactBottomMost := false

	if opts != nil {
		if opts.ChangeLevel != nil {
			changeLevel = *opts.ChangeLevel
		}
		if opts.CompactBottomMostLevel != nil {
			compactBottomMost = *opts.CompactBottomMostLevel
		}
	}

	modifyRequest = append(modifyRequest,
		connection.WithQuery("changeLevel", strconv.FormatBool(changeLevel)),
		connection.WithQuery("compactBottomMostLevel", strconv.FormatBool(compactBottomMost)))

	var response map[string]interface{}
	resp, err := connection.CallPut(ctx, c.client.connection, url, &response, nil, modifyRequest...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response, nil
	default:
		return nil, (&shared.ResponseStruct{}).AsArangoErrorWithCode(code)
	}
}
