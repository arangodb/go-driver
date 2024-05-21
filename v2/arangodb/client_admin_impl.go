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
