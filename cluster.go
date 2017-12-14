//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"time"
)

// Cluster provides access to cluster wide specific operations.
// To use this interface, an ArangoDB cluster is required.
type Cluster interface {
	// Get the cluster configuration & health
	Health(ctx context.Context) (ClusterHealth, error)

	// MoveShard moves a single shard of the given collection from server `fromServer` to
	// server `toServer`.
	MoveShard(ctx context.Context, col Collection, shard int, fromServer, toServer ServerID) error
}

// ServerID identifies an arangod server in a cluster.
type ServerID string

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
	Endpoint            string       `json:"Endpoint"`
	LastHeartbeatAcked  time.Time    `json:"LastHeartbeatAcked"`
	LastHeartbeatSent   time.Time    `json:"LastHeartbeatSent"`
	LastHeartbeatStatus string       `json:"LastHeartbeatStatus"`
	Role                ServerRole   `json:"Role"`
	ShortName           string       `json:"ShortName"`
	Status              ServerStatus `json:"Status"`
	CanBeDeleted        bool         `json:"CanBeDeleted"`
	HostID              string       `json:"Host,omitempty"`
}

// ServerRole is the role of an arangod server
type ServerRole string

const (
	ServerRoleDBServer    ServerRole = "DBServer"
	ServerRoleCoordinator ServerRole = "Coordinator"
	ServerRoleAgent       ServerRole = "Agent"
)

// ServerStatus describes the health status of a server
type ServerStatus string

const (
	ServerStatusGood ServerStatus = "GOOD"
)
