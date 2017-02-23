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

// ensureDatabase is a helper to check if a database exists and create it if needed.
// It will fail the test when an error occurs.
func ensureDatabase(ctx context.Context, c driver.Client, name string, options *driver.CreateDatabaseOptions, t *testing.T) driver.Database {
	db, err := c.Database(ctx, name)
	if driver.IsNotFound(err) {
		db, err = c.CreateDatabase(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create database '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open database '%s': %s", name, describe(err))
	}
	return db
}

// TestCreateDatabase creates a database and then checks that it exists.
func TestCreateDatabase(t *testing.T) {
	c := createClientFromEnv(t, true)
	name := "create_test1"
	if _, err := c.CreateDatabase(nil, name, nil); err != nil {
		t.Fatalf("Failed to create database '%s': %s", name, describe(err))
	}
	// Database must exist now
	if found, err := c.DatabaseExists(nil, name); err != nil {
		t.Errorf("DatabaseExists('%s') failed: %s", name, describe(err))
	} else if !found {
		t.Errorf("DatabaseExists('%s') return false, expected true", name)
	}
}
