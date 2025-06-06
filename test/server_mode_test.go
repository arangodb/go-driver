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

package test

import (
	"context"
	"os"
	"strings"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestServerMode creates a database and checks the various server modes.
func TestServerMode(t *testing.T) {
	c := createClient(t, nil)
	ctx := context.Background()

	if strings.HasPrefix(os.Getenv("TEST_AUTHENTICATION"), "super:") {
		t.Skip("Skipping read only test because of superuser access.")
	}
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv33p := version.Version.CompareTo("3.3") >= 0
	if !isv33p {
		t.Skip("This test requires version 3.3")
	} else {
		// Create simple collection
		db := ensureDatabase(ctx, c, "_system", nil, t)
		colName := "server_mode_test1"
		col := ensureCollection(ctx, db, colName, nil, t)

		// Initial server mode must be default
		if mode, err := c.ServerMode(ctx); err != nil {
			t.Fatalf("ServerMode failed: %s", describe(err))
		} else if mode != driver.ServerModeDefault {
			t.Errorf("ServerMode returned '%s', but expected '%s'", mode, driver.ServerModeDefault)
		}

		// Change server mode to readonly.
		if err := c.SetServerMode(ctx, driver.ServerModeReadOnly); err != nil {
			t.Fatalf("SetServerMode failed: %s", describe(err))
		}

		// Try to drop collection now (it must fail)
		if err := col.Remove(ctx); !driver.IsForbidden(err) {
			t.Fatalf("Collection remove should have return ForbiddenError, got error %s", describe(err))
		}

		// Check server mode, must be readonly
		if mode, err := c.ServerMode(ctx); err != nil {
			t.Fatalf("ServerMode failed: %s", describe(err))
		} else if mode != driver.ServerModeReadOnly {
			t.Errorf("ServerMode returned '%s', but expected '%s'", mode, driver.ServerModeReadOnly)
		}

		// Change server mode back to default.
		if err := c.SetServerMode(ctx, driver.ServerModeDefault); err != nil {
			t.Fatalf("SetServerMode failed: %s", describe(err))
		}

		// Try to drop collection now (it must succeed)
		if err := col.Remove(ctx); err != nil {
			t.Fatalf("Collection remove failed: %s", describe(err))
		}
		err = db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}
}
