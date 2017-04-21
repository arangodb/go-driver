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
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	driver "github.com/arangodb/go-driver"
)

// connection is a single socket connection to a server.
type connection struct {
	lastMessageID uint64
	maxChunkSize  uint32
	msgStore      messageStore
	conn          net.Conn
	writeMutex    sync.Mutex
	closing       bool
	lastActivity  time.Time
}

const (
	defaultMaxChunkSize = 30000
)

var (
	vstProtocolHeader = []byte("VST/1.0\r\n\r\n")
)

// dial opens a new connection to the server on the given address.
func dial(addr string, tlsConfig *tls.Config) (*connection, error) {
	var conn net.Conn
	var err error
	if tlsConfig != nil {
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, driver.WithStack(err)
	}

	// Configure connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetNoDelay(true)
	}

	// Send protocol header
	if _, err := conn.Write(vstProtocolHeader); err != nil {
		return nil, driver.WithStack(err)
	}

	// prepare connection
	c := &connection{
		maxChunkSize: defaultMaxChunkSize,
		conn:         conn,
	}
	c.updateLastActivity()

	// Start reading responses
	go c.readChunkLoop()

	return c, nil
}

// Close the connection to the server
func (c *connection) Close() error {
	if !c.closing {
		c.closing = true
		if err := c.conn.Close(); err != nil {
			return driver.WithStack(err)
		}
		c.msgStore.ForEach(func(m *Message) {
			if m.response != nil {
				close(m.response)
				m.response = nil
			}
		})
	}
	return nil
}

// IsClosed returns true when the connection is closed, false otherwise.
func (c *connection) IsClosed() bool {
	return c.closing
}

// Send sends a message (consisting of given parts) to the server and returns
// a channel on which the response will be delivered.
// When the connection is closed before a response was received, the returned
// channel will be closed.
func (c *connection) Send(ctx context.Context, messageParts ...[]byte) (<-chan Message, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	msgID := atomic.AddUint64(&c.lastMessageID, 1)
	chunks, err := buildChunks(msgID, c.maxChunkSize, messageParts...)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	// Prepare for receiving a response
	m := c.msgStore.Add(msgID)

	//panic(fmt.Sprintf("chunks: %d, messageParts: %d, first: %s", len(chunks), len(messageParts), hex.EncodeToString(messageParts[0])))

	// Send all chunks
	sendErrors := make(chan error)
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Time{}
	}
	go func() {
		defer close(sendErrors)
		for _, chunk := range chunks {
			if err := c.sendChunk(deadline, chunk); err != nil {
				// Cancel response
				c.msgStore.Remove(msgID)
				// Return error
				sendErrors <- driver.WithStack(err)
				return
			}
		}
	}()

	// Wait for sending to be ready, or context to be cancelled.
	select {
	case err := <-sendErrors:
		if err != nil {
			return nil, driver.WithStack(err)
		}
		return m.response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendChunk sends a single chunk to the server.
func (c *connection) sendChunk(deadline time.Time, chunk chunk) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	c.conn.SetWriteDeadline(deadline)
	_, err := chunk.WriteTo(c.conn)
	c.updateLastActivity()
	if err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// readChunkLoop reads chunks from the connection until it is closed.
func (c *connection) readChunkLoop() {
	for {
		if c.closing {
			// Closing, we're done
			return
		}
		chunk, err := readChunk(c.conn)
		c.updateLastActivity()
		if err != nil {
			if !c.closing {
				// Handle error
				if err == io.EOF {
					// Connection closed
					c.Close()
				} else {
					fmt.Printf("readChunkLoop error: %#v\n", err)
				}
			}
		} else {
			// Process chunk
			go c.processChunk(chunk)
		}
	}
}

// processChunk adds the given chunk to its message and notifies the listener
// when the message is complete.
func (c *connection) processChunk(chunk chunk) {
	m := c.msgStore.Get(chunk.MessageID)
	if m == nil {
		// Unexpected chunk, ignore it
		return
	}

	// Add chunk to message
	m.addChunk(chunk)

	// Try to assembly
	if m.assemble() {
		// Message is complete
		// Remove message from store
		c.msgStore.Remove(m.ID)

		//fmt.Println("Chunk: " + hex.EncodeToString(chunk.Data) + "\nMessage: " + hex.EncodeToString(m.Data))

		// Notify listener
		if m.response != nil {
			m.response <- *m
			close(m.response)
		}
	}
}

// updateLastActivity sets the lastActivity field to the current time.
func (c *connection) updateLastActivity() {
	c.lastActivity = time.Now()
}

// IsIdle returns true when the last activity was more than the given timeout ago.
func (c *connection) IsIdle(idleTimeout time.Duration) bool {
	return time.Since(c.lastActivity) > idleTimeout
}
