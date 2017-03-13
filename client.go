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

// Client provides access to a single arangodb database server, or an entire cluster of arangodb servers.
type Client interface {
	// Version returns version information from the connected database server.
	// Use WithDetails to configure a context that will include additional details in the return VersionInfo.
	Version(ctx context.Context) (VersionInfo, error)

	// SynchronizeEndpoints fetches all endpoints from an ArangoDB cluster and updates the
	// connection to use those endpoints.
	// When this client is connected to a single server, nothing happens.
	// When this client is connected to a cluster of servers, the connection will be updated to reflect
	// the layout of the cluster.
	SynchronizeEndpoints(ctx context.Context) error

	// Database functions
	ClientDatabases

	// User functions
	ClientUsers
}

// ClientConfig contains all settings needed to create a client.
type ClientConfig struct {
	// Connection is the actual server/cluster connection.
	// See http.NewConnection.
	Connection Connection
	// Authentication implements authentication on the server.
	Authentication Authentication
	// SynchronizeEndpointsInterval is the interval between automatisch synchronization of endpoints.
	// If this value is 0, no automatic synchronization is performed.
	// If this value is > 0, automatic synchronization is started on a go routine.
	SynchronizeEndpointsInterval time.Duration
}

// VersionInfo describes the version of a database server.
type VersionInfo struct {
	// This will always contain "arango"
	Server string `json:"server,omitempty"`
	//  The server version string. The string has the format "major.minor.sub".
	// Major and minor will be numeric, and sub may contain a number or a textual version.
	Version Version `json:"version,omitempty"`
	// Type of license of the server
	License string `json:"license,omitempty"`
	// Optional additional details. This is returned only if the context is configured using WithDetails.
	Details map[string]interface{} `json:"details,omitempty"`
}
