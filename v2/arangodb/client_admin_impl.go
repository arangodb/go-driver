//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

type clientAdmin struct {
	client *client
}

func newClientAdmin(client *client) *clientAdmin {
	return &clientAdmin{
		client: client,
	}
}

var _ ClientAdmin = &clientAdmin{}

// Health returns the cluster configuration & health.
// It works in cluster or active fail-over mode.
func (c clientAdmin) Health(ctx context.Context) (ClusterHealth, error) {
	url := connection.NewUrl("_admin", "cluster", "health")

	var response struct {
		shared.ResponseStruct `json:",inline"`
		ClusterHealth         `json:",inline"` // TODo inline ?
	}

	resp, err := connection.CallGet(ctx, c.client.connection, url, &response)
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
