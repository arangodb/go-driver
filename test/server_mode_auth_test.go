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

// +build auth

package test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestServerModeAndGrants checks user access grants in combination with
// server mode and WithConfigured.
func TestServerModeAndGrants(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv33p := version.Version.CompareTo("3.3") >= 0
	if !isv33p {
		t.Skip("This test requires version 3.3")
	} else {
		// Get root user
		u, err := c.User(ctx, "root")
		if err != nil {
			t.Fatalf("User('root') failed: %s", describe(err))
		}

		// Initial server mode must be default
		if mode, err := c.ServerMode(ctx); err != nil {
			t.Fatalf("ServerMode failed: %s", describe(err))
		} else if mode != driver.ServerModeDefault {
			t.Errorf("ServerMode returned '%s', but expected '%s'", mode, driver.ServerModeDefault)
		}

		// Create simple collection
		db := ensureDatabase(ctx, c, "_system", nil, t)
		colName := "server_mode_and_grants_test1"
		col := ensureCollection(ctx, db, colName, nil, t)

		// Get database & collection access
		defaultDBAccess, err := u.GetDatabaseAccess(ctx, db)
		if err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		}
		defaultColAccess, err := u.GetCollectionAccess(ctx, col)
		if err != nil {
			t.Fatalf("GetCollectionAccess failed: %s", describe(err))
		}

		// Get database & collection access using WithConfigured
		if grant, err := u.GetDatabaseAccess(driver.WithConfigured(ctx), db); err != nil {
			t.Fatalf("GetDatabaseAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Database access using WithConfigured differs, got '%s', expected '%s'", grant, defaultDBAccess)
		}
		if grant, err := u.GetCollectionAccess(driver.WithConfigured(ctx), col); err != nil {
			t.Fatalf("GetCollectionAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Collection access using WithConfigured differs, got '%s', expected '%s'", grant, defaultColAccess)
		}

		// Change server mode to readonly.
		if err := c.SetServerMode(ctx, driver.ServerModeReadOnly); err != nil {
			t.Fatalf("SetServerMode failed: %s", describe(err))
		}

		// Check server mode, must be readonly
		if mode, err := c.ServerMode(ctx); err != nil {
			t.Fatalf("ServerMode failed: %s", describe(err))
		} else if mode != driver.ServerModeReadOnly {
			t.Errorf("ServerMode returned '%s', but expected '%s'", mode, driver.ServerModeReadOnly)
		}

		// Get database & collection access now (must be readonly)
		if grant, err := u.GetDatabaseAccess(ctx, db); err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		} else if grant != driver.GrantReadOnly {
			t.Errorf("Database access must be readonly, got '%s'", grant)
		}
		if grant, err := u.GetCollectionAccess(ctx, col); err != nil {
			t.Fatalf("GetCollectionAccess failed: %s", describe(err))
		} else if grant != driver.GrantReadOnly {
			t.Errorf("Collection access must be readonly, got '%s'", grant)
		}

		// Get database & collection access using WithConfigured (must be same as before)
		if grant, err := u.GetDatabaseAccess(driver.WithConfigured(ctx), db); err != nil {
			t.Fatalf("GetDatabaseAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Database access using WithConfigured differs, got '%s', expected '%s'", grant, defaultDBAccess)
		}
		if grant, err := u.GetCollectionAccess(driver.WithConfigured(ctx), col); err != nil {
			t.Fatalf("GetCollectionAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Collection access using WithConfigured differs, got '%s', expected '%s'", grant, defaultColAccess)
		}

		// Change server mode back to default.
		if err := c.SetServerMode(ctx, driver.ServerModeDefault); err != nil {
			t.Fatalf("SetServerMode failed: %s", describe(err))
		}

		// Initial server mode must be default
		if mode, err := c.ServerMode(ctx); err != nil {
			t.Fatalf("ServerMode failed: %s", describe(err))
		} else if mode != driver.ServerModeDefault {
			t.Errorf("ServerMode returned '%s', but expected '%s'", mode, driver.ServerModeDefault)
		}

		// Get database & collection access (must now be same as before)
		if grant, err := u.GetDatabaseAccess(ctx, db); err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Database access differs, got '%s', expected '%s'", grant, defaultDBAccess)
		}
		if grant, err := u.GetCollectionAccess(ctx, col); err != nil {
			t.Fatalf("GetCollectionAccess failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Collection access differs, got '%s', expected '%s'", grant, defaultColAccess)
		}

		// Get database & collection access with WithConfigured (must now be same as before)
		if grant, err := u.GetDatabaseAccess(driver.WithConfigured(ctx), db); err != nil {
			t.Fatalf("GetDatabaseAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Database access using WithConfigured differs, got '%s', expected '%s'", grant, defaultDBAccess)
		}
		if grant, err := u.GetCollectionAccess(driver.WithConfigured(ctx), col); err != nil {
			t.Fatalf("GetCollectionAccess(WithConfigured) failed: %s", describe(err))
		} else if grant != defaultDBAccess {
			t.Errorf("Collection access using WithConfigured differs, got '%s', expected '%s'", grant, defaultColAccess)
		}
                col.Remove(ctx)
	}
}
