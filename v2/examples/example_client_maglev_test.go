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
	"fmt"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// ExampleNewMaglevHashEndpoints shows how to create a new MaglevHashEndpoints
// It lets use different endpoints for different databases by using a RequestDBNameValueExtractor function.
// E.g.:
// - all requests to _db/<db-name-1> will use endpoint 1
// - all requests to _db/<db-name-2> will use endpoint 2
func ExampleNewMaglevHashEndpoints() {
	// Create an HTTP connection to the database
	endpoints, err := connection.NewMaglevHashEndpoints(
		[]string{"https://a:8529", "https://a:8539", "https://b:8529"},
		connection.RequestDBNameValueExtractor,
	)

	conn := connection.NewHttpConnection(exampleJSONHTTPConnectionConfig(endpoints))

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
