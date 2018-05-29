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

// Connection is a single socket connection to a server.
type Connection struct {
	version       Version
	lastMessageID uint64
	maxChunkSize  uint32
	msgStore      messageStore
	conn          net.Conn
	writeMutex    sync.Mutex
	closing       int32
	lastActivity  time.Time
	configured    int32 // Set to 1 after the configuration callback has finished without errors.
}

const (
	defaultMaxChunkSize = 30000
	maxRecentErrors     = 64
)

var (
	vstProtocolHeader1_0 = []byte("VST/1.0\r\n\r\n")
	vstProtocolHeader1_1 = []byte("VST/1.1\r\n\r\n")
)

// dial opens a new connection to the server on the given address.
func dial(version Version, addr string, tlsConfig *tls.Config) (*Connection, error) {
	var conn net.Conn
	var err error
	if tlsConfig != nil {
		tlsConfigCopy := *tlsConfig
		tlsConfigCopy.MaxVersion = tls.VersionTLS10
		conn, err = tls.Dial("tcp", addr, &tlsConfigCopy)
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
	switch version {
	case Version1_0:
		if _, err := conn.Write(vstProtocolHeader1_0); err != nil {
			return nil, driver.WithStack(err)
		}
	case Version1_1:
		if _, err := conn.Write(vstProtocolHeader1_1); err != nil {
			return nil, driver.WithStack(err)
		}
	default:
		return nil, driver.WithStack(fmt.Errorf("Unknown protocol version %d", int(version)))
	}

	// prepare connection
	c := &Connection{
		version:      version,
		maxChunkSize: defaultMaxChunkSize,
		conn:         conn,
	}
	c.updateLastActivity()

	// Start reading responses
	go c.readChunkLoop()

	return c, nil
}

// load returns an indication of the amount of work this connection has.
// 0 means no work at all, >0 means some work.
func (c *Connection) load() int {
	return c.msgStore.Size()
}

// Close the connection to the server
func (c *Connection) Close() error {
	if atomic.CompareAndSwapInt32(&c.closing, 0, 1) {
		if err := c.conn.Close(); err != nil {
			return driver.WithStack(err)
		}
		c.msgStore.ForEach(func(m *Message) {
			m.closeResponseChan()
		})
	}
	return nil
}

// IsClosed returns true when the connection is closed, false otherwise.
func (c *Connection) IsClosed() bool {
	return atomic.LoadInt32(&c.closing) == 1
}

// IsConfigured returns true when the configuration callback has finished on this connection, without errors.
func (c *Connection) IsConfigured() bool {
	return atomic.LoadInt32(&c.configured) == 1
}

// Send sends a message (consisting of given parts) to the server and returns
// a channel on which the response will be delivered.
// When the connection is closed before a response was received, the returned
// channel will be closed.
func (c *Connection) Send(ctx context.Context, messageParts ...[]byte) (<-chan Message, error) {
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
	responseChan := m.responseChan

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
		return responseChan, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendChunk sends a single chunk to the server.
func (c *Connection) sendChunk(deadline time.Time, chunk chunk) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	c.conn.SetWriteDeadline(deadline)
	var err error
	switch c.version {
	case Version1_0:
		_, err = chunk.WriteToVST1_0(c.conn)
	case Version1_1:
		_, err = chunk.WriteToVST1_1(c.conn)
	default:
		err = driver.WithStack(fmt.Errorf("Unknown protocol version %d", int(c.version)))
	}
	c.updateLastActivity()
	if err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// readChunkLoop reads chunks from the connection until it is closed.
func (c *Connection) readChunkLoop() {
	recentErrors := 0
	goodChunks := 0
	for {
		if c.IsClosed() {
			// Closing, we're done
			return
		}
		var chunk chunk
		var err error
		switch c.version {
		case Version1_0:
			chunk, err = readChunkVST1_0(c.conn)
		case Version1_1:
			chunk, err = readChunkVST1_1(c.conn)
		default:
			err = driver.WithStack(fmt.Errorf("Unknown protocol version %d", int(c.version)))
		}
		c.updateLastActivity()
		if err != nil {
			if !c.IsClosed() {
				// Handle error
				if err == io.EOF {
					// Connection closed
					c.Close()
				} else {
					recentErrors++
					fmt.Printf("readChunkLoop error: %#v (goodChunks=%d)\n", err, goodChunks)
					if recentErrors > maxRecentErrors {
						// When we get to many errors in a row, close this connection
						c.Close()
					} else {
						// Backoff a bit, so we allow things to settle.
						time.Sleep(time.Millisecond * time.Duration(recentErrors*5))
					}
				}
			}
		} else {
			// Process chunk
			recentErrors = 0
			goodChunks++
			go c.processChunk(chunk)
		}
	}
}

// processChunk adds the given chunk to its message and notifies the listener
// when the message is complete.
func (c *Connection) processChunk(chunk chunk) {
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
		m.notifyListener()
	}
}

// updateLastActivity sets the lastActivity field to the current time.
func (c *Connection) updateLastActivity() {
	c.lastActivity = time.Now()
}

// IsIdle returns true when the last activity was more than the given timeout ago.
func (c *Connection) IsIdle(idleTimeout time.Duration) bool {
	return time.Since(c.lastActivity) > idleTimeout
}
