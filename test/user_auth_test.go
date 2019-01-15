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
	"time"

	driver "github.com/arangodb/go-driver"
)

// TestUpdateUserPasswordMyself creates a user and tries to update the password of the authenticated user.
func TestUpdateUserPasswordMyself(t *testing.T) {
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
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
	ensureSynchronizedEndpoints(authClient, "", t)

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
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
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
	ensureSynchronizedEndpoints(authClient, "", t)

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
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
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
	if isv32p {
		// Read back access
		if grant, err := u.GetDatabaseAccess(nil, db); err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		} else if grant != driver.GrantReadWrite {
			t.Errorf("Database access invalid, expected 'rw', got '%s'", grant)
		}
	}

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication("grant_user1", "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	ensureSynchronizedEndpoints(authClient, "grant_user_test", t)
	authDb := waitForDatabaseAccess(authClient, "grant_user_test", t)

	// Try to create a collection in the db
	if _, err := authDb.CreateCollection(nil, "some_collection", nil); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}

	// Now revoke access
	if err := u.SetDatabaseAccess(nil, db, driver.GrantNone); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	if isv32p {
		// Read back access
		if grant, err := u.GetDatabaseAccess(nil, db); err != nil {
			t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
		} else if grant != driver.GrantNone {
			t.Errorf("Database access invalid, expected 'none', got '%s'", grant)
		}
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

// TestGrantUserDefaultDatabase creates a user & database and granting the user access to the "default" database.
func TestGrantUserDefaultDatabase(t *testing.T) {
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv32p := version.Version.CompareTo("3.2") >= 0
	if !isv32p {
		t.Skipf("This test requires 3.2 or higher, got %s", version.Version)
	}

	u := ensureUser(nil, c, "grant_user_def", &driver.UserOptions{Password: "foo"}, t)
	db := ensureDatabase(nil, c, "grant_user_def_test", nil, t)
	// Grant read/write access to default database
	if err := u.SetDatabaseAccess(nil, nil, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	// Read back default database access
	if grant, err := u.GetDatabaseAccess(nil, nil); err != nil {
		t.Fatalf("GetDatabaseAccess failed: %s", describe(err))
	} else if grant != driver.GrantReadWrite {
		t.Errorf("Collection access invalid, expected 'rw', got '%s'", grant)
	}

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication(u.Name(), "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	ensureSynchronizedEndpoints(authClient, "grant_user_def_test", t)

	// Try to create a collection in the db, should succeed
	authDb := waitForDatabaseAccess(authClient, "grant_user_def_test", t)

	authCol, err := authDb.CreateCollection(nil, "books_def_db", nil)
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Remove explicit grant for db
	if err := u.RemoveDatabaseAccess(nil, db); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Remove explicit grant for col
	if err := u.RemoveCollectionAccess(nil, authCol); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Grant read-only access to default database
	if err := u.SetDatabaseAccess(nil, nil, driver.GrantReadOnly); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}

	// wait for change to propagate
	{
		deadline := time.Now().Add(time.Minute)
		for {
			// Try to create document in collection, should fail because there are no collection grants for this user and/or collection.
			if _, err := authCol.CreateDocument(nil, Book{Title: "I cannot write"}); err == nil {
				if time.Now().Before(deadline) {
					t.Logf("Expected failure, got %s, trying again...", describe(err))
					time.Sleep(time.Second * 2)
					continue
				}
				t.Errorf("Expected failure, got %s", describe(err))
			}

			// Try to create collection, should fail
			if _, err := authDb.CreateCollection(nil, "books_def_ro_db", nil); err == nil {
				t.Errorf("Expected failure, got %s", describe(err))
			}
			break
		}
	}

	// Grant no access to default database
	if err := u.SetDatabaseAccess(nil, nil, driver.GrantNone); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}

	// wait for change to propagate
	{
		deadline := time.Now().Add(time.Minute)
		for {
			// Try to create collection, should fail
			if _, err := authDb.CreateCollection(nil, "books_def_none_db", nil); err == nil {
				if time.Now().Before(deadline) {
					t.Logf("Expected failure, got %s, trying again...", describe(err))
					time.Sleep(time.Second * 2)
					continue
				}
				t.Errorf("Expected failure, got %s", describe(err))
			}
			break
		}
	}

	// Remove default database access, should fallback to "no-access" then
	if err := u.RemoveDatabaseAccess(nil, nil); err != nil {
		t.Fatalf("RemoveDatabaseAccess failed: %s", describe(err))
	}
	// wait for change to propagate
	{
		deadline := time.Now().Add(time.Minute)
		for {
			// Try to create collection, should fail
			if _, err := authDb.CreateCollection(nil, "books_def_star_db", nil); err == nil {
				if time.Now().Before(deadline) {
					t.Logf("Expected failure, got %s, trying again...", describe(err))
					time.Sleep(time.Second * 2)
					continue
				}
				t.Errorf("Expected failure, got %s", describe(err))
			}
			break
		}
	}
}

// TestGrantUserCollection creates a user & database & collection and granting the user access to the collection.
func TestGrantUserCollection(t *testing.T) {
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
	c := createClientFromEnv(t, true)
	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	// 3.3.4 changes behaviour to better support LDAP
	isv32p := version.Version.CompareTo("3.2") >= 0
	isv334 := version.Version.CompareTo("3.3.4") >= 0
	if !isv32p {
		t.Skipf("This test requires 3.2 or higher, got %s", version.Version)
	}

	u := ensureUser(nil, c, "grant_user_col", &driver.UserOptions{Password: "foo"}, t)
	db := ensureDatabase(nil, c, "grant_user_col_test", nil, t)
	// Grant read/write access to database
	if err := u.SetDatabaseAccess(nil, db, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	col := ensureCollection(nil, db, "grant_col_test", nil, t)
	// Grant read/write access to collection
	if err := u.SetCollectionAccess(nil, col, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetCollectionAccess failed: %s", describe(err))
	}

	// wait for change to propagate
	{
		deadline := time.Now().Add(time.Minute)
		for {
			// Read back collection access
			if grant, err := u.GetCollectionAccess(nil, col); err == nil {
				if grant == driver.GrantReadWrite {
					break
				}
				if time.Now().Before(deadline) {
					t.Logf("Expected failure, got %s, trying again...", describe(err))
					time.Sleep(time.Second * 2)
					continue
				}
				t.Errorf("Collection access invalid, expected 'rw', got '%s'", grant)
			} else {
				t.Fatalf("GetCollectionAccess failed: %s", describe(err))
			}
		}
	}

	authClient, err := driver.NewClient(driver.ClientConfig{
		Connection:     createConnectionFromEnv(t),
		Authentication: driver.BasicAuthentication("grant_user_col", "foo"),
	})
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	ensureSynchronizedEndpoints(authClient, "grant_user_col_test", t)
	authDb := waitForDatabaseAccess(authClient, "grant_user_col_test", t)

	// Try to create a document in the col
	authCol, err := authDb.Collection(nil, col.Name())
	if err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}
	meta1, err := authCol.CreateDocument(nil, Book{Title: "I can write"})
	if err != nil {
		t.Errorf("CreateDocument failed: %s", describe(err))
	}

	// Now set collection access to Read-only
	if err := u.SetCollectionAccess(nil, col, driver.GrantReadOnly); err != nil {
		t.Fatalf("SetCollectionAccess failed: %s", describe(err))
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != driver.GrantReadOnly {
		t.Errorf("Collection access invalid, expected 'ro', got '%s'", grant)
	}
	// Try to create another document, should fail
	if _, err := authCol.CreateDocument(nil, Book{Title: "I should not be able to write"}); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got: %s", describe(err))
	}
	// Try to read back first document, should succeed
	var doc Book
	if _, err := authCol.ReadDocument(nil, meta1.Key, &doc); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}

	// Now set collection access to None
	if err := u.SetCollectionAccess(nil, col, driver.GrantNone); err != nil {
		t.Fatalf("SetCollectionAccess failed: %s", describe(err))
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != driver.GrantNone {
		t.Errorf("Collection access invalid, expected 'none', got '%s'", grant)
	}
	// Try to create another document, should fail
	if _, err := authCol.CreateDocument(nil, Book{Title: "I should not be able to write"}); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got: %s", describe(err))
	}
	// Try to read back first document, should fail
	if _, err := authCol.ReadDocument(nil, meta1.Key, &doc); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got %s", describe(err))
	}
	// Now remove explicit collection access
	if err := u.RemoveCollectionAccess(nil, col); err != nil {
		t.Fatalf("RemoveCollectionAccess failed: %s", describe(err))
	}
	expected := driver.GrantNone
	if isv334 {
		expected = driver.GrantReadWrite
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != expected {
		t.Errorf("Collection access invalid, expected '%s', got '%s'", expected, grant)
	}
	// Grant read-only access to database
	if err := u.SetDatabaseAccess(nil, db, driver.GrantReadOnly); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	expected = driver.GrantNone
	if isv334 {
		expected = driver.GrantReadOnly
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != expected {
		t.Errorf("Collection access invalid, expected '%s', got '%s'", expected, grant)
	}
	// Try to create another document, should fail
	if _, err := authCol.CreateDocument(nil, Book{Title: "I should not be able to write"}); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got: %s", describe(err))
	}
	// Grant no access to collection
	if err := u.SetCollectionAccess(nil, col, driver.GrantNone); err != nil {
		t.Fatalf("SetDatabaseAccess failed: %s", describe(err))
	}
	// Try to read back first document, should fail
	if _, err := authCol.ReadDocument(nil, meta1.Key, &doc); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got %s", describe(err))
	}

	// Set default collection access to read-only
	if err := u.SetCollectionAccess(nil, db, driver.GrantReadOnly); err != nil {
		t.Fatalf("SetCollectionAccess failed: %s", describe(err))
	}
	if err := u.RemoveCollectionAccess(nil, col); err != nil {
		t.Fatalf("RemoveCollectionAccess failed: %s", describe(err))
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != driver.GrantReadOnly {
		t.Errorf("Collection access invalid, expected 'ro', got '%s'", grant)
	}
	// Try to create another document, should fail
	if _, err := authCol.CreateDocument(nil, Book{Title: "I should not be able to write"}); !driver.IsForbidden(err) {
		t.Errorf("Expected failure, got: %s", describe(err))
	}
	// Try to read back first document, should succeed
	if _, err := authCol.ReadDocument(nil, meta1.Key, &doc); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}

	// Set default collection access to read-write
	if err := u.SetCollectionAccess(nil, db, driver.GrantReadWrite); err != nil {
		t.Fatalf("SetCollectionAccess failed: %s", describe(err))
	}
	// Read back collection access
	if grant, err := u.GetCollectionAccess(nil, col); err != nil {
		t.Fatalf("GetCollectionAccess failed: %s", describe(err))
	} else if grant != driver.GrantReadWrite {
		t.Errorf("Collection access invalid, expected 'rw', got '%s'", grant)
	}
	// Try to create another document, should succeed
	if _, err := authCol.CreateDocument(nil, Book{Title: "I should again be able to write"}); err != nil {
		t.Errorf("Expected success, got: %s", describe(err))
	}
	// Try to read back first document, should succeed
	if _, err := authCol.ReadDocument(nil, meta1.Key, &doc); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUserAccessibleDatabases creates a user & databases and checks the list of accessible databases.
func TestUserAccessibleDatabases(t *testing.T) {
	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
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

func waitForDatabaseAccess(authClient driver.Client, dbname string, t *testing.T) driver.Database {
	deadline := time.Now().Add(time.Minute)
	for {
		// Try to select the database
		authDb, err := authClient.Database(nil, dbname)
		if err == nil {
			return authDb
		}
		if time.Now().Before(deadline) {
			t.Logf("Expected success, got %s, trying again...", describe(err))
			time.Sleep(time.Second * 2)
			continue
		}
		t.Fatalf("Failed to select database, got %s", describe(err))
		return nil
	}
}

func ensureSynchronizedEndpoints(authClient driver.Client, dbname string, t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	if err := waitUntilEndpointSynchronized(ctx, authClient, dbname, t); err != nil {
		t.Fatalf("Failed to synchronize endpoint: %s", describe(err))
	}
}
