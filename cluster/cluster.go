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
	"net"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
)

// ConnectionConfig provides all configuration options for a cluster connection.
type ConnectionConfig struct {
	// DefaultTimeout is the timeout used by requests that have no timeout set in the given context.
	DefaultTimeout time.Duration
}

// NewConnection creates a new cluster connection to a cluster of servers.
// The given connections are existing connections to each of the servers.
func NewConnection(config ConnectionConfig, servers ...driver.Connection) (driver.Connection, error) {
	if len(servers) == 0 {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "Must provide at least 1 server"})
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = defaultTimeout
	}
	return &clusterConnection{
		servers:        servers,
		defaultTimeout: config.DefaultTimeout,
	}, nil
}

const (
	defaultTimeout = time.Minute
)

type clusterConnection struct {
	servers        []driver.Connection
	current        int
	mutex          sync.RWMutex
	defaultTimeout time.Duration
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
		cancel()
		if err == nil {
			// We're done
			return resp, nil
		}
		// No success yet
		cause := driver.Cause(err)
		if cause == context.Canceled {
			// Request was cancelled, we return directly.
			return nil, driver.WithStack(err)
		}
		if cause == context.DeadlineExceeded {
			// Server context timeout, failover to a new server
		} else {
			// Some other error has occurred, check for network errors
			if _, ok := cause.(net.Error); ok {
				// Error is a network error, failover to new server
			} else {
				// A connection error has occurred, return the error.
				return nil, driver.WithStack(err)
			}
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
