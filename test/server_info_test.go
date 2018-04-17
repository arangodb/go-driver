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

package test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestServerID tests ClientServerInfo.ServerID.
func TestServerID(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	var isCluster bool
	if _, err := c.Cluster(ctx); driver.IsPreconditionFailed(err) {
		isCluster = false
	} else if err != nil {
		t.Fatalf("Health failed: %s", describe(err))
	} else {
		isCluster = true
	}

	if isCluster {
		id, err := c.ServerID(ctx)
		if err != nil {
			t.Fatalf("ServerID failed: %s", describe(err))
		}
		if id == "" {
			t.Error("Expected ID to be non-empty")
		}
	} else {
		if _, err := c.ServerID(ctx); err == nil {
			t.Fatalf("ServerID succeeded, expected error")
		}
	}
}
