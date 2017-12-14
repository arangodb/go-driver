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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestClusterHealth tests the Cluster.Health method.
func TestClusterHealth(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	cl, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else {
		h, err := cl.Health(ctx)
		if err != nil {
			t.Fatalf("Health failed: %s", describe(err))
		}
		if h.ID == "" {
			t.Error("Expected cluster ID to be non-empty")
		}
		agents := 0
		dbservers := 0
		coordinators := 0
		for _, sh := range h.Health {
			switch sh.Role {
			case driver.ServerRoleAgent:
				agents++
			case driver.ServerRoleDBServer:
				dbservers++
			case driver.ServerRoleCoordinator:
				coordinators++
			}
		}
		if agents == 0 {
			t.Error("Expected at least 1 agent")
		}
		if dbservers == 0 {
			t.Error("Expected at least 1 dbserver")
		}
		if coordinators == 0 {
			t.Error("Expected at least 1 coordinator")
		}
	}
}
