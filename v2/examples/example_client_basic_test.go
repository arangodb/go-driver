//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

package examples

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// ExampleNewClient shows how to create the simple client with a single endpoint
func ExampleNewClient() {
	// Create an HTTP connection to the database
	endpoint := connection.NewRoundRobinEndpoints([]string{"http://localhost:8529"})
	conn := connection.NewHttpConnection(exampleJSONHTTPConnectionConfig(endpoint))

	// Create a client
	client := arangodb.NewClient(conn)

	// Ask the version of the server
	versionInfo, err := client.Version(context.Background())
	if err != nil {
		fmt.Printf("Failed to get version info: %v", err)
	} else {
		fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
	}
}

func exampleJSONHTTPConnectionConfig(endpoint connection.Endpoint) connection.HttpConfiguration {
	return connection.HttpConfiguration{
		Endpoint:    endpoint,
		ContentType: connection.ApplicationJSON,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 90 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
