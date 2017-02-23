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

package main

import (
	"flag"
	"fmt"
	"log"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

var (
	endpoint string
)

func init() {
	flag.StringVar(&endpoint, "endpoint", "http://localhost:8529", "URL used to connect to the database")
}

func main() {
	flag.Parse()
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{endpoint},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	versionInfo, err := c.Version(nil)
	if err != nil {
		log.Fatalf("Failed to get version info: %v", err)
	}
	fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
}
