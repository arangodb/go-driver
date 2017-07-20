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

// TestUpdateUserPasswordMyself creates a user and tries to update the password of the authenticated user.
func TestUpdateUserPasswordMyself(t *testing.T) {
	var conn driver.Connection
	c := createClientFromEnv(t, true, &conn)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	isVST1_0 := conn.Protocols().Contains(driver.ProtocolVST1_0)
	ensureUser(nil, c, "user@TestUpdateUserPasswordMyself", &driver.UserOptions{Password: "foo"}, t)

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication("user@TestUpdateUserPasswordMyself", "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	if isVST1_0 && !isv32p {
		t.Skip("Cannot update my own password using VST in 3.1")
	} else {
		u, err := authClient.User(nil, "user@TestUpdateUserPasswordMyself")
		if err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
		if err := u.Update(context.TODO(), driver.UserOptions{Password: "something"}); err != nil {
			t.Errorf("Expected success, got %s", describe(err))
		}
	}
}

// TestUpdateUserPasswordOtherUser creates a user and tries to update the password of another user.
func TestUpdateUserPasswordOtherUser(t *testing.T) {
	var conn driver.Connection
	c := createClientFromEnv(t, true, &conn)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	isVST1_0 := conn.Protocols().Contains(driver.ProtocolVST1_0)
	u1 := ensureUser(nil, c, "user1", &driver.UserOptions{Password: "foo"}, t)
	ensureUser(nil, c, "user2", nil, t)
	systemDb, err := c.Database(nil, "_system")
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication("user1", "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	if isVST1_0 && !isv32p {
		t.Skip("Cannot update other password using VST in 3.1")
	} else {
		// Right now user1 has no right to access user2
		if _, err := authClient.User(nil, "user2"); !driver.IsForbidden(err) {
			t.Fatalf("Expected ForbiddenError, got %s", describe(err))
		}

		// Grant user1 access to _system db, then it should be able to access user2
		if err := u1.SetDatabaseAccess(nil, systemDb, driver.GrantReadWrite); err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}

		// Now change the password of another user.
		// With user1 having rights for _system, this must succeed now
		u2, err := authClient.User(nil, "user2")
		if err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
		if err := u2.Update(context.TODO(), driver.UserOptions{Password: "something"}); err != nil {
			t.Errorf("Expected success, got %s", describe(err))
		}
	}
}

// TestGrantUserDatabase creates a user & database and granting the user access to the database.
func TestGrantUserDatabase(t *testing.T) {
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	u := ensureUser(nil, c, "grant_user1", &driver.UserOptions{Password: "foo"}, t)
	db := ensureDatabase(nil, c, "grant_user_test", nil, t)

	// Grant read/write access
	if err := u.SetDatabaseAccess(nil, db, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	// Read back access
	if grant, err := u.GetDatabaseAccess(nil, db); err != nil {
		t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
	} else if grant != driver.GrantReadWrite {
		t.Errorf("Database access invalid, expected 'rw', got '%s'", grant)
	}

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication("grant_user1", "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Try to create a collection in the db
	authDb, err := authClient.Database(nil, "grant_user_test")
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	if _, err := authDb.CreateCollection(nil, "some_collection", nil); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}

	// Now revoke access
	if err := u.SetDatabaseAccess(nil, db, driver.GrantNone); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	// Read back access
	if grant, err := u.GetDatabaseAccess(nil, db); err != nil {
		t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
	} else if grant != driver.GrantNone {
		t.Errorf("Database access invalid, expected 'none', got '%s'", grant)
	}

	// Try to access the db, should fail now
	if _, err := authClient.Database(nil, "grant_user_test"); !driver.IsUnauthorized(err) {
		t.Errorf("Expected UnauthorizedError, got %s %#v", describe(err), err)
	}

	if isv32p {
		// Now grant read-only access
		if err := u.SetDatabaseAccess(nil, db, driver.GrantReadOnly); err != nil {
			t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
		}
		// Read back access
		if grant, err := u.GetDatabaseAccess(nil, db); err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		} else if grant != driver.GrantReadOnly {
			t.Errorf("Database access invalid, expected 'ro', got '%s'", grant)
		}
		// Try to access the db, should succeed
		if _, err := authClient.Database(nil, "grant_user_test"); err != nil {
			t.Errorf("Expected success, got %s", describe(err))
		}
		// Try to create another collection, should fail
		if _, err := authDb.CreateCollection(nil, "some_other_collection", nil); !driver.IsForbidden(err) {
			t.Errorf("Expected UnauthorizedError, got %s %#v", describe(err), err)
		}
	} else {
		t.Logf("SetDatabaseAccess(ReadOnly) is not supported on versions below 3.2 (got version %s)", version.Version)
	}
}

// TestUserAccessibleDatabases creates a user & databases and checks the list of accessible databases.
func TestUserAccessibleDatabases(t *testing.T) {
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	u := ensureUser(nil, c, "accessible_db_user1", nil, t)
	db1 := ensureDatabase(nil, c, "accessible_db1", nil, t)
	db2 := ensureDatabase(nil, c, "accessible_db2", nil, t)

	contains := func(list []driver.Database, name string) bool {
		for _, db := range list {
			if db.Name() == name {
				return true
			}
		}
		return false
	}

	expectListContains := func(listName string, list []driver.Database, name ...string) {
		for _, n := range name {
			if !contains(list, n) {
				t.Errorf("Expected list '%s' to contain '%s', it did not", listName, n)
			}
		}
	}

	expectListNotContains := func(listName string, list []driver.Database, name ...string) {
		for _, n := range name {
			if contains(list, n) {
				t.Errorf("Expected list '%s' to not contain '%s', it did", listName, n)
			}
		}
	}

	// Nothing allowed yet
	list, err := u.AccessibleDatabases(nil)
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	expectListContains("expect-none", list)
	expectListNotContains("expect-none", list, db1.Name(), db2.Name())

	// Allow db1
	if err := u.SetDatabaseAccess(nil, db1, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}

	list, err = u.AccessibleDatabases(nil)
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	expectListContains("expect-db1", list, db1.Name())
	expectListNotContains("expect-db1", list, db2.Name())

	// allow db2, revoke db1
	if err := u.SetDatabaseAccess(nil, db2, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetDatabaseAccess(RW) failed: %s", describe(err))
	}
	if err := u.SetDatabaseAccess(nil, db1, driver.GrantNone); err != nil {
		t.Fatalf("SetDatabaseAccess(None) failed: %s", describe(err))
	}

	if isv32p {
		list, err = u.AccessibleDatabases(nil)
		if err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
		expectListContains("expect-db2", list, db2.Name())
		expectListNotContains("expect-db2", list, db1.Name())

		// revoke db2
		if err := u.SetDatabaseAccess(nil, db2, driver.GrantNone); err != nil {
			t.Fatalf("SetDatabaseAccess(None) failed: %s", describe(err))
		}

		list, err = u.AccessibleDatabases(nil)
		if err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
		expectListContains("expect-none2", list)
		expectListNotContains("expect-none2", list, db1.Name(), db2.Name())

		// grant read-only access to db1, db2
		if err := u.SetDatabaseAccess(nil, db1, driver.GrantReadOnly); err != nil {
			t.Fatalf("SetDatabaseAccess(RO) failed: %s", describe(err))
		}
		if err := u.SetDatabaseAccess(nil, db2, driver.GrantReadOnly); err != nil {
			t.Fatalf("SetDatabaseAccess(RO) failed: %s", describe(err))
		}

		list, err = u.AccessibleDatabases(nil)
		if err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
		expectListContains("expect-db1-db2", list, db1.Name(), db2.Name())
		expectListNotContains("expect-db1-db2", list)

	} else {
		t.Logf("Last part of test fails on version < 3.2 (got version %s)", version.Version)
	}
}
