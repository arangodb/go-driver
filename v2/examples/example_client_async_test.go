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
	"log"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

func ExampleNewConnectionAsyncWrapper() {
	// Create an HTTP connection to the database
	conn := connection.NewHttpConnection(exampleJSONHTTPConnectionConfig())

	// Create ASYNC wrapper for the connection
	conn = connection.NewConnectionAsyncWrapper(conn)

	// Create a client
	client := arangodb.NewClient(conn)

	// Ask the version of the server
	versionInfo, err := client.Version(context.Background())
	if err != nil {
		fmt.Printf("Failed to get version info: %v", err)
	} else {
		fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
	}

	// Trigger async request
	info, err := client.Version(connection.WithAsync(context.Background()))
	if err != nil {
		fmt.Printf("this is expected error since we are using async mode and response is not ready yet: %v", err)
	}
	if info.Version != "" {
		fmt.Printf("Expected empty version if async request is in progress, got %s", info.Version)
	}

	// Fetch an async job id from the error
	id, isAsyncId := connection.IsAsyncJobInProgress(err)
	if !isAsyncId {
		fmt.Printf("Expected async job id, got %v", id)
	}

	// Wait for an async result
	time.Sleep(3 * time.Second)

	// List async jobs - there should be one, till the result is fetched
	jobs, err := client.AsyncJobList(context.Background(), arangodb.JobDone, nil)
	if err != nil {
		log.Fatalf("Failed to list async jobs: %v", err)
	}
	if len(jobs) != 1 {
		log.Fatalf("Expected 1 async job, got %d", len(jobs))
	}

	// Fetch an async job result
	info, err = client.Version(connection.WithAsyncID(context.Background(), id))
	if err != nil {
		log.Fatalf("Failed to fetch async job result: %v", err)
	}
}
