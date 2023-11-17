//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

//go:build !auth

package examples

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/arangodb/go-driver"
	drviverHttp "github.com/arangodb/go-driver/http"
)

// ExampleNewClient shows how to create the simple client with a single endpoint
// By default, the client will use the http.DefaultTransport configuration
func ExampleNewClient() {
	// Create an HTTP connection to the database
	conn, err := drviverHttp.NewConnection(drviverHttp.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	// Ask the version of the server
	versionInfo, err := c.Version(nil)
	if err != nil {
		log.Fatalf("Failed to get version info: %v", err)
	}
	fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
}

// ExampleNewConnection shows how to create the client with custom connection configuration
// If there is more than one endpoint, the client will pick the first one that works and use it till it fails.
// Then it will try the next one.
func ExampleNewConnection() {
	// Create an HTTP connection to the database
	conn, err := drviverHttp.NewConnection(drviverHttp.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529", "http://localhost:8539"},
		Transport: NewConnectionTransport(),
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	// Ask the version of the server
	versionInfo, err := c.Version(nil)
	if err != nil {
		log.Fatalf("Failed to get version info: %v", err)
	}
	fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
}

// NewConnectionTransport creates a new http.RoundTripper (values are copied from http.DefaultTransport)
func NewConnectionTransport() http.RoundTripper {
	return &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
