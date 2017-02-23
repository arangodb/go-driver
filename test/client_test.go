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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

// describe returns a string description of the given error.
func describe(err error) string {
	if err == nil {
		return "nil"
	}
	cause := driver.Cause(err)
	c, _ := json.Marshal(cause)
	if cause.Error() != err.Error() {
		return fmt.Sprintf("%v caused by %v", err, string(c))
	} else {
		return fmt.Sprintf("%v", string(c))
	}
}

// getEndpointsFromEnv returns the endpoints specified in the TEST_ENDPOINTS
// environment variable.
func getEndpointsFromEnv(t *testing.T) []string {
	eps := strings.Split(os.Getenv("TEST_ENDPOINTS"), ",")
	if len(eps) == 0 {
		t.Fatal("No endpoints found in environment variable TEST_ENDPOINTS")
	}
	return eps
}

// createAuthenticationFromEnv initializes an authentication specified in the TEST_AUTHENTICATION
// environment variable.
func createAuthenticationFromEnv(t *testing.T) driver.Authentication {
	authSpec := os.Getenv("TEST_AUTHENTICATION")
	if authSpec == "" {
		return nil
	}
	parts := strings.Split(authSpec, ":")
	switch parts[0] {
	case "basic":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for basic authentication")
		}
		return driver.BasicAuthentication(parts[1], parts[2])
	case "jwt":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for jwt authentication")
		}
		return driver.JWTAuthentication(parts[1], parts[2])
	default:
		t.Fatalf("Unknown authentication: '%s'", parts[0])
		return nil
	}
}

// createConnectionFromEnv initializes a Connection from information specified in environment variables.
func createConnectionFromEnv(t *testing.T) driver.Connection {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: getEndpointsFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new http connection: %s", describe(err))
	}
	return conn
}

// createClientFromEnv initializes a Client from information specified in environment variables.
func createClientFromEnv(t *testing.T, waitUntilReady bool) driver.Client {
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: createAuthenticationFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new client: %s", describe(err))
	}
	if waitUntilReady {
		timeout := time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if up := waitUntilServerAvailable(ctx, c); !up {
			t.Fatalf("Connection is not available in %s", timeout)
		}
	}
	return c
}

// waitUntilServerAvailable keeps waiting until the server/cluster that the client is addressing is available.
func waitUntilServerAvailable(ctx context.Context, c driver.Client) bool {
	instanceUp := make(chan bool)
	go func() {
		for {
			verCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if _, err := c.Version(verCtx); err == nil {
				cancel()
				instanceUp <- true
				return
			} else {
				cancel()
				time.Sleep(time.Second)
			}
		}
	}()
	select {
	case up := <-instanceUp:
		return up
	case <-ctx.Done():
		return false
	}
}

// TestCreateClientHttpConnection creates an HTTP connection to the environment specified
// endpoints and creates a client for that.
func TestCreateClientHttpConnection(t *testing.T) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: getEndpointsFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new http connection: %s", describe(err))
	}
	_, err = driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: createAuthenticationFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new client: %s", describe(err))
	}
}
