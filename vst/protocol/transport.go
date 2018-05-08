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

package protocol

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"
	"time"

	driver "github.com/arangodb/go-driver"
)

const (
	DefaultIdleConnTimeout = time.Minute
	DefaultConnLimit       = 3
)

// TransportConfig contains configuration options for Transport.
type TransportConfig struct {
	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration

	// ConnLimit is the upper limit to the number of connections to a single server.
	// Due to the nature of the VST protocol, this value does not have to be high.
	// The default is 3 (DefaultConnLimit).
	ConnLimit int

	// Version specifies the version of the Velocystream protocol
	Version Version
}

// Transport manages client-server connections using the VST protocol to a specific host.
type Transport struct {
	TransportConfig

	hostAddr            string
	tlsConfig           *tls.Config
	connMutex           sync.Mutex
	connections         []*Connection
	onConnectionCreated func(context.Context, *Connection) error
}

// NewTransport creates a new Transport for given address & tls settings.
func NewTransport(hostAddr string, tlsConfig *tls.Config, config TransportConfig) *Transport {
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = DefaultIdleConnTimeout
	}
	if config.ConnLimit == 0 {
		config.ConnLimit = DefaultConnLimit
	}
	return &Transport{
		TransportConfig: config,
		hostAddr:        hostAddr,
		tlsConfig:       tlsConfig,
	}
}

// Send sends a message (consisting of given parts) to the server and returns
// a channel on which the response will be delivered.
// When the connection is closed before a response was received, the returned
// channel will be closed.
func (c *Transport) Send(ctx context.Context, messageParts ...[]byte) (<-chan Message, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	result, err := conn.Send(ctx, messageParts...)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return result, nil
}

// CloseIdleConnections closes all connections which are closed or have been idle for more than the configured idle timeout.
func (c *Transport) CloseIdleConnections() (closed, remaining int) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	for i := 0; i < len(c.connections); {
		conn := c.connections[i]
		if conn.IsClosed() || conn.IsIdle(c.IdleConnTimeout) {
			// Remove connection from list
			c.connections = append(c.connections[:i], c.connections[i+1:]...)
			// Close connection
			go conn.Close()
			closed++
		} else {
			i++
		}
	}

	remaining = len(c.connections)
	return closed, remaining
}

// CloseAllConnections closes all connections.
func (c *Transport) CloseAllConnections() {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	for _, conn := range c.connections {
		// Close connection
		go conn.Close()
	}
}

// SetOnConnectionCreated stores a callback function that is called every time a new connection has been created.
func (c *Transport) SetOnConnectionCreated(handler func(context.Context, *Connection) error) {
	c.onConnectionCreated = handler
}

// getConnection returns the first available connection, or when no such connection is available,
// is created a new connection.
func (c *Transport) getConnection(ctx context.Context) (*Connection, error) {
	conn := c.getAvailableConnection()
	if conn != nil {
		return conn, nil
	}

	// No connections available, make a new one
	conn, err := c.createConnection()
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, driver.WithStack(err)
	}

	// Invoke callback
	if cb := c.onConnectionCreated; cb != nil {
		if err := cb(ctx, conn); err != nil {
			conn.Close()
			return nil, driver.WithStack(err)
		}
	}

	// Mark the connection as ready
	atomic.StoreInt32(&conn.configured, 1)

	return conn, nil
}

// getAvailableConnection returns the first available connection.
// If no such connection is available, nil is returned.
func (c *Transport) getAvailableConnection() *Connection {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	// Select the connection with the least amount of traffic
	var bestConn *Connection
	bestConnLoad := 0
	activeConnCount := 0
	for _, conn := range c.connections {
		if !conn.IsClosed() {
			activeConnCount++
			if conn.IsConfigured() {
				connLoad := conn.load()
				if bestConn == nil || connLoad < bestConnLoad {
					bestConn = conn
					bestConnLoad = connLoad
				}
			}
		}
	}

	if bestConn == nil {
		// No connections available
		return nil
	}

	// Is load is >0 AND the number of connections is below the limit, create a new one
	if bestConnLoad > 0 && activeConnCount < c.ConnLimit {
		return nil
	}

	// Use the best connection found
	bestConn.updateLastActivity()
	return bestConn
}

// createConnection creates a new connection.
func (c *Transport) createConnection() (*Connection, error) {
	conn, err := dial(c.Version, c.hostAddr, c.tlsConfig)
	if err != nil {
		return nil, driver.WithStack(err)
	}

	// Record connection
	c.connMutex.Lock()
	c.connections = append(c.connections, conn)
	startCleanup := len(c.connections) == 1
	c.connMutex.Unlock()

	if startCleanup {
		// TODO enable cleanup
		go c.cleanup()
	}

	return conn, nil
}

// cleanup keeps removing idle connections
func (c *Transport) cleanup() {
	for {
		time.Sleep(c.IdleConnTimeout / 10)
		remaining, _ := c.CloseIdleConnections()
		if remaining == 0 {
			return
		}
	}
}
