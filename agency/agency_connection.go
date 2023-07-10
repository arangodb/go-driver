//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

package agency

import (
	"context"
	"fmt"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

const (
	minAgencyTimeout = time.Second * 2

	keyRawResponse driver.ContextKey = "arangodb-rawResponse"
	keyResponse    driver.ContextKey = "arangodb-response"
)

type agencyConnection struct {
	mutex       sync.RWMutex
	config      http.ConnectionConfig
	connections []driver.Connection
	auth        driver.Authentication
}

// NewAgencyConnection creates an agency connection for agents at the given endpoints.
// This type of connection differs from normal HTTP/VST connection in the way
// requests are executed.
// This type of connection makes use of the fact that only 1 agent will respond
// to requests at a time. All other agents will respond with an "I'm not the leader" error.
// A request will be send to all agents at the same time.
// The result of the first agent to respond with a normal response is used.
func NewAgencyConnection(config http.ConnectionConfig) (driver.Connection, error) {
	c := &agencyConnection{
		config: config,
	}
	if err := c.UpdateEndpoints(config.Endpoints); err != nil {
		return nil, driver.WithStack(err)
	}
	return c, nil
}

// NewRequest creates a new request with given method and path.
func (c *agencyConnection) NewRequest(method, path string) (driver.Request, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.connections) == 0 {
		return nil, driver.WithStack(fmt.Errorf("no connections"))
	}
	r, err := c.connections[0].NewRequest(method, path)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return r, nil
}

// Do performs a given request, returning its response.
// In case of a termporary failure, the request is retried until
// the deadline is exceeded.
func (c *agencyConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Second * 30)
	}
	timeout := time.Until(deadline)
	if timeout < minAgencyTimeout {
		timeout = minAgencyTimeout
	}
	attempt := 1
	delay := agencyConnectionFailureBackoff(0)
	for {
		lctx, cancel := context.WithTimeout(ctx, timeout/3)
		resp, isPerm, err := c.doOnce(lctx, req)
		cancel()
		if err == nil {
			// Success
			return resp, nil
		} else if isPerm {
			// Permanent error
			return nil, driver.WithStack(err)
		}
		// Is deadline exceeded?
		if time.Now().After(deadline) {
			return nil, driver.WithStack(fmt.Errorf("All %d attemps resulted in temporary failure", attempt))
		}
		// Just retry
		attempt++
		delay = agencyConnectionFailureBackoff(delay)
		// Wait a bit so we don't hammer the agency
		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			// Context canceled
			return nil, driver.WithStack(ctx.Err())
		}
	}
}

// Do performs a given request once, returning its response.
// Returns: Response, isPermanentError, Error
func (c *agencyConnection) doOnce(ctx context.Context, req driver.Request) (driver.Response, bool, error) {
	c.mutex.RLock()
	connections := c.connections
	c.mutex.RUnlock()

	if len(c.connections) == 0 {
		return nil, true, driver.WithStack(fmt.Errorf("no connections"))
	}

	parallelRequests := true
	if ctx != nil {
		if v := ctx.Value(keyResponse); v != nil {
			parallelRequests = false
		}
		if v := ctx.Value(keyRawResponse); v != nil {
			parallelRequests = false
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	results := make(chan driver.Response, len(connections))
	errors := make(chan error, len(connections))
	wg := sync.WaitGroup{}
	for _, epConn := range connections {
		wg.Add(1)
		go func(epConn driver.Connection) {
			defer wg.Done()
			epReq := req.Clone()
			result, err := epConn.Do(ctx, epReq)
			if err == nil {
				if err = isSuccess(result); err == nil {
					// Success
					results <- result
					// Cancel all other requests
					cancel()
					return
				}
			}
			// Check error
			if statusCode, ok := isArangoError(err); ok {
				// We have a status code, check it
				if statusCode >= 400 && statusCode < 500 && statusCode != 408 {
					// Permanent error, return it
					errors <- driver.WithStack(err)
					// Cancel all other requests
					cancel()
					return
				}
			}
			// No permanent error. Are we the only endpoint?
			if len(connections) == 1 {
				errors <- driver.WithStack(err)
			}
			// No permanent error, try next agent
		}(epConn)
		if !parallelRequests {
			// Parallel requests not allowed so we should wait till routine finishes
			wg.Wait()
		}
	}

	if parallelRequests {
		// Wait for go routines to be finished
		wg.Wait()
	}

	cancel()
	close(results)
	close(errors)
	if result, ok := <-results; ok {
		// Return first result
		return result, false, nil
	}
	if err, ok := <-errors; ok {
		// Return first error
		return nil, true, driver.WithStack(err)
	}
	return nil, false, driver.WithStack(fmt.Errorf("All %d servers responded with temporary failure", len(connections)))
}

func isSuccess(resp driver.Response) error {
	if resp == nil {
		return driver.WithStack(fmt.Errorf("Response is nil"))
	}
	statusCode := resp.StatusCode()
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return driver.ArangoError{
		HasError: true,
		Code:     statusCode,
	}
}

// isArangoError checks if the given error is (or is caused by) an ArangoError.
// If so it returned the Code and true, otherwise it returns 0, false.
func isArangoError(err error) (int, bool) {
	if aerr, ok := driver.Cause(err).(driver.ArangoError); ok {
		return aerr.Code, true
	}
	return 0, false
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *agencyConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.connections) == 0 {
		return driver.WithStack(fmt.Errorf("no connections"))
	}
	if err := c.connections[0].Unmarshal(data, result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Endpoints returns the endpoints used by this connection.
func (c *agencyConnection) Endpoints() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var result []string
	for _, x := range c.connections {
		result = append(result, x.Endpoints()...)
	}
	return result
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *agencyConnection) UpdateEndpoints(endpoints []string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	newConnections := make([]driver.Connection, len(endpoints))
	for i, ep := range endpoints {
		config := c.config
		config.Endpoints = []string{ep}
		config.DontFollowRedirect = true
		httpConn, err := http.NewConnection(config)
		if err != nil {
			return driver.WithStack(err)
		}
		if c.auth != nil {
			httpConn, err = httpConn.SetAuthentication(c.auth)
			if err != nil {
				return driver.WithStack(err)
			}
		}
		newConnections[i] = httpConn
	}
	c.connections = newConnections
	return nil
}

// Configure the authentication used for this connection.
func (c *agencyConnection) SetAuthentication(auth driver.Authentication) (driver.Connection, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	newConnections := make([]driver.Connection, len(c.connections))
	for i, x := range c.connections {
		xAuth, err := x.SetAuthentication(auth)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		newConnections[i] = xAuth
	}
	c.connections = newConnections
	c.auth = auth
	return c, nil
}

// Protocols returns all protocols used by this connection.
func (c *agencyConnection) Protocols() driver.ProtocolSet {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := driver.ProtocolSet{}
	for _, x := range c.connections {
		for _, p := range x.Protocols() {
			if !result.Contains(p) {
				result = append(result, p)
			}
		}
	}
	return result
}

// agencyConnectionFailureBackoff returns a backoff delay for cases where all
// agents responded with a non-fatal error.
func agencyConnectionFailureBackoff(lastDelay time.Duration) time.Duration {
	return increaseDelay(lastDelay, 1.5, time.Millisecond, time.Second*2)
}

// increaseDelay returns an delay, increased from an old delay with a given
// factor, limited to given min & max.
func increaseDelay(oldDelay time.Duration, factor float64, min, max time.Duration) time.Duration {
	delay := time.Duration(float64(oldDelay) * factor)
	if delay < min {
		delay = min
	}
	if delay > max {
		delay = max
	}
	return delay
}
