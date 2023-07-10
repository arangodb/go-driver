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

//go:build !auth

package driver_test

import (
	"context"
	"fmt"
	"log"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/util/connection/wrappers/async"
)

func ExampleNewClientWithAsyncMode() {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}

	// Create a client with optional async mode
	c, err := driver.NewClientWithAsyncMode(driver.ClientConfig{Connection: conn}, true)

	// Trigger async request
	info, err := c.Version(driver.WithAsync(context.Background()))
	if err != nil {
		fmt.Printf("this is expected error since we are using async mode and response is not ready yet: %v", err)
	}
	if info.Version != "" {
		log.Fatalf("Expected empty version if async request is in progress, got %s", info.Version)
	}

	// Fetch async job id
	id, isAsyncId := async.IsAsyncJobInProgress(err)
	if !isAsyncId {
		log.Fatalf("Expected async job id, got %v", id)
	}

	// Wait for an async result
	time.Sleep(3 * time.Second)

	// List async jobs - there should be one, till the result is fetched
	jobs, err := c.AsyncJob().List(context.Background(), driver.JobDone, nil)
	if err != nil {
		log.Fatalf("Failed to list async jobs: %v", err)
	}
	if len(jobs) != 1 {
		log.Fatalf("Expected 1 async job, got %d", len(jobs))
	}

	// Fetch an async job result
	info, err = c.Version(driver.WithAsyncId(context.Background(), id))
	if err != nil {
		log.Fatalf("Failed to fetch async job result: %v", err)
	}
}
