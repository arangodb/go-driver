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

package cluster

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
)

// ConnectionConfig provides all configuration options for a cluster connection.
type ConnectionConfig struct {
	// DefaultTimeout is the timeout used by requests that have no timeout set in the given context.
	DefaultTimeout time.Duration
}

// ServerConnectionBuilder specifies a function called by the cluster connection when it
// needs to create an underlying connection to a specific endpoint.
type ServerConnectionBuilder func(endpoint string) (driver.Connection, error)

// NewConnection creates a new cluster connection to a cluster of servers.
// The given connections are existing connections to each of the servers.
func NewConnection(config ConnectionConfig, connectionBuilder ServerConnectionBuilder, endpoints []string) (driver.Connection, error) {
	if connectionBuilder == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "Must a connection builder"})
	}
	if len(endpoints) == 0 {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "Must provide at least 1 endpoint"})
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = defaultTimeout
	}
	cConn := &clusterConnection{
		connectionBuilder: connectionBuilder,
		defaultTimeout:    config.DefaultTimeout,
	}
	// Initialize endpoints
	if err := cConn.UpdateEndpoints(endpoints); err != nil {
		return nil, driver.WithStack(err)
	}
	return cConn, nil
}

const (
	defaultTimeout = time.Minute
)

type clusterConnection struct {
	connectionBuilder ServerConnectionBuilder
	servers           []driver.Connection
	endpoints         []string
	current           int
	mutex             sync.RWMutex
	defaultTimeout    time.Duration
}

// NewRequest creates a new request with given method and path.
func (c *clusterConnection) NewRequest(method, path string) (driver.Request, error) {
	// It is assumed that all servers used the same protocol.
	return c.servers[0].NewRequest(method, path)
}

// Do performs a given request, returning its response.
func (c *clusterConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Timeout management.
	// We take the given timeout and divide it in 3 so we allow for other servers
	// to give it a try if an earlier server fails.
	deadline, hasDeadline := ctx.Deadline()
	var timeout time.Duration
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	} else {
		timeout = c.defaultTimeout
	}

	attempt := 1
	s := c.getCurrentServer()
	for {
		serverCtx, cancel := context.WithTimeout(ctx, timeout/3)
		resp, err := s.Do(serverCtx, req)
		if driver.Cause(err) == context.Canceled {
			// Request was cancelled, we return directly.
			cancel()
			return nil, driver.WithStack(err)
		} else if driver.Cause(err) == context.DeadlineExceeded {
			// Server context timeout, failover to a new server
			cancel()
			// Will continue after this
		} else if err == nil {
			// We're done
			cancel()
			return resp, nil
		} else {
			// A connection error has occurred, return the error.
			cancel()
			return nil, driver.WithStack(err)
		}

		// Failed, try next server
		attempt++
		if attempt > len(c.servers) {
			// We've tried all servers. Giving up.
			return nil, driver.WithStack(err)
		}
		s = c.getNextServer()
	}
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *clusterConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	if err := c.servers[0].Unmarshal(data, result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *clusterConnection) UpdateEndpoints(endpoints []string) error {
	if len(endpoints) == 0 {
		return driver.WithStack(driver.InvalidArgumentError{Message: "Must provide at least 1 endpoint"})
	}
	sort.Strings(endpoints)
	if strings.Join(endpoints, ",") == strings.Join(c.endpoints, ",") {
		// No changes
		return nil
	}

	// Create new connections
	servers := make([]driver.Connection, 0, len(endpoints))
	for _, ep := range endpoints {
		conn, err := c.connectionBuilder(ep)
		if err != nil {
			return driver.WithStack(err)
		}
		servers = append(servers, conn)
	}

	// Swap connections
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.servers = servers
	c.endpoints = endpoints
	c.current = 0

	return nil
}

// Endpoints returns the endpoints used by this connection.
func (c *clusterConnection) Endpoints() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var endpoints []string
	for _, s := range c.servers {
		endpoints = append(endpoints, s.Endpoints()...)
	}

	return endpoints
}

// getCurrentServer returns the currently used server.
func (c *clusterConnection) getCurrentServer() driver.Connection {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.servers[c.current]
}

// getNextServer changes the currently used server and returns the new server.
func (c *clusterConnection) getNextServer() driver.Connection {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.current = (c.current + 1) % len(c.servers)
	return c.servers[c.current]
}
