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
	"log"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// ExampleNewConnectionAsyncWrapper shows how to create a connection wrapper for async requests
// It lets use async requests on demand
func Main() {
	// Create an HTTP connection to the database
	endpoint := connection.NewRoundRobinEndpoints([]string{"http://localhost:8529"})
	conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

	auth := connection.NewBasicAuth("root", "")
	err := conn.SetAuthentication(auth)
	if err != nil {
		log.Fatalf("Failed to set authentication: %v", err)
	}

	// Create ASYNC wrapper for the connection
	conn = connection.NewConnectionAsyncWrapper(conn)

	// Create a client
	client := arangodb.NewClient(conn)

	// Ask the version of the server
	versionInfo, err := client.Version(context.Background())
	if err != nil {
		log.Fatalf("Failed to get version info: %v", err)
	}
	log.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)

	// Trigger async request
	info, errWithJobID := client.Version(connection.WithAsync(context.Background()))
	if errWithJobID == nil {
		log.Fatalf("err should not be nil. It should be an async job id")
	}
	if info.Version != "" {
		log.Printf("Expected empty version if async request is in progress, got %s", info.Version)
	}

	// Fetch an async job id from the error
	id, isAsyncId := connection.IsAsyncJobInProgress(errWithJobID)
	if !isAsyncId {
		log.Fatalf("Expected async job id, got %v", id)
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
	log.Printf("Async job result: %s", info.Version)
}
