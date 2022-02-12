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
	"encoding/json"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// ensureUser is a helper to check if a user exists and create it if needed.
// It will fail the test when an error occurs.
func ensureUser(ctx context.Context, c driver.Client, name string, options *driver.UserOptions, t *testing.T) driver.User {
	u, err := c.User(ctx, name)
	if driver.IsNotFound(err) {
		u, err = c.CreateUser(ctx, name, options)
		if err != nil {
			t.Fatalf("Failed to create user '%s': %s", name, describe(err))
		}
	} else if err != nil {
		t.Fatalf("Failed to open user '%s': %s", name, describe(err))
	}
	return u
}

// TODO velocity test
// TestCreateUser creates a user and then checks that it exists.
func TestCreateUser(t *testing.T) {
	c := createClientFromEnv(t, true)

	tests := map[string]*driver.UserOptions{
		"jan1":   nil,
		"george": &driver.UserOptions{Password: "foo", Active: boolRef(false)},
		"candy":  &driver.UserOptions{Password: "ARANGODB_DEFAULT_ROOT_PASSWORD", Active: boolRef(true)},
		"joe":    &driver.UserOptions{Extra: map[string]interface{}{"key": "value", "x": 5}},
		// Some strange names
		"ewout/foo": nil,
		"admin@api": nil,
		"測試用例":      nil,
		"測試用例@foo":  nil,
		"_":         nil,
		//" ":         nil, // No longer valid in 3.2
		"/": nil,
	}

	for name, options := range tests {
		if _, err := c.CreateUser(nil, name, options); err != nil {
			t.Fatalf("Failed to create user '%s': %s", name, describe(err))
		}
		// User must exist now
		if found, err := c.UserExists(nil, name); err != nil {
			t.Errorf("UserExists('%s') failed: %s", name, describe(err))
		} else if !found {
			t.Errorf("UserExists('%s') return false, expected true", name)
		}

		// Must be able to open user
		if u, err := c.User(nil, name); err != nil {
			t.Errorf("Failed to open user '%s': %s", name, describe(err))
		} else {
			if u.Name() != name {
				t.Errorf("Invalid name, expected '%s', got '%s'", name, u.Name())
			}
			if options != nil {
				if options.Active != nil {
					if u.IsActive() != *options.Active {
						t.Errorf("Invalid active, expected '%v', got '%v'", *options.Active, u.IsActive())
					}
				}
				var extra map[string]interface{}
				if err := u.Extra(&extra); err != nil {
					t.Errorf("Expected success, got %s", describe(err))
				} else {
					if options.Extra == nil {
						if len(extra) != 0 {
							t.Errorf("Invalid extra, expected 'nil', got '%+v'", extra)
						}
					} else {
						expected, _ := json.Marshal(options.Extra)
						got, _ := json.Marshal(extra)
						if string(expected) != string(got) {
							t.Errorf("Invalid extra, expected '%s', got '%s'", string(expected), string(got))
						}
					}
				}
			}
			if u.IsPasswordChangeNeeded() != false {
				t.Errorf("Invalid passwordChangeNeeded, expected 'false', got '%v'", u.IsPasswordChangeNeeded())
			}
		}

		// Create again (must fail)
		if _, err := c.CreateUser(nil, name, options); !driver.IsConflict(err) {
			t.Fatalf("Expected ConflictError, got %s", describe(err))
		}
	}

	// Fetch all users
	users, err := c.Users(nil)
	if err != nil {
		t.Fatalf("Failed to fetch users: %s", describe(err))
	}
	for userName := range tests {
		foundUser := false
		for _, u := range users {
			if u.Name() == userName {
				foundUser = true
				break
			}
		}
		if !foundUser {
			t.Errorf("Cannot find user '%s'", userName)
		}
	}

	// Now remove the users
	for userName := range tests {
		u, err := c.User(nil, userName)
		if err != nil {
			t.Errorf("Expected success, got %s", describe(err))
		} else {
			if err := u.Remove(context.Background()); err != nil {
				t.Errorf("Failed to remove user '%s': %s", userName, describe(err))
			}

			// User must no longer exist
			if found, err := c.UserExists(nil, userName); err != nil {
				t.Errorf("Expected success, got %s", describe(err))
			} else if found {
				t.Errorf("Expected user '%s' to be NOT found, but it was found", userName)
			}
		}
	}
}

// TestUpdateUser creates a user and performs various updates.
func TestUpdateUser(t *testing.T) {
	c := createClientFromEnv(t, true)
	u := ensureUser(nil, c, "update_user", nil, t)

	if err := u.Update(context.TODO(), driver.UserOptions{}); err != nil {
		t.Errorf("Cannot update user with empty options: %s", describe(err))
	}

	if u.IsActive() != true {
		t.Errorf("Expected IsActive to be true, got false")
	}
	if err := u.Update(context.TODO(), driver.UserOptions{
		Active: boolRef(false),
	}); err != nil {
		t.Errorf("Cannot update user with Active in options: %s", describe(err))
	}
	if u.IsActive() != false {
		t.Errorf("Expected IsActive to be false, got true")
	}

	if err := u.Update(context.TODO(), driver.UserOptions{
		Active: boolRef(true),
	}); err != nil {
		t.Errorf("Cannot update user with Active in options: %s", describe(err))
	}
	if u.IsActive() != true {
		t.Errorf("Expected IsActive to be true, got false")
	}

	book := Book{Title: "Testing is fun"}
	if err := u.Update(context.TODO(), driver.UserOptions{
		Extra: book,
	}); err != nil {
		t.Errorf("Cannot update user with Extra in options: %s", describe(err))
	}
	var readBook Book
	if err := u.Extra(&readBook); err != nil {
		t.Errorf("Failed to read extra: %s", describe(err))
	} else if !reflect.DeepEqual(book, readBook) {
		t.Errorf("Extra differs; expected '%+v', got '%+v'", book, readBook)
	}
}

// TestReplaceUser creates a user and performs various replacements.
func TestReplaceUser(t *testing.T) {
	c := createClientFromEnv(t, true)
	u := ensureUser(nil, c, "replace_user", nil, t)

	if err := u.Replace(context.TODO(), driver.UserOptions{}); err != nil {
		t.Errorf("Cannot replace user with empty options: %s", describe(err))
	}

	if u.IsActive() != true {
		t.Errorf("Expected IsActive to be true, got false")
	}
	if err := u.Replace(context.TODO(), driver.UserOptions{
		Active: boolRef(false),
	}); err != nil {
		t.Errorf("Cannot replace user with Active in options: %s", describe(err))
	}
	if u.IsActive() != false {
		t.Errorf("Expected IsActive to be false, got true")
	}

	book := Book{Title: "Testing is fun"}
	if err := u.Replace(context.TODO(), driver.UserOptions{
		Extra: book,
	}); err != nil {
		t.Errorf("Cannot replace user with Extra in options: %s", describe(err))
	}
	var readBook Book
	if err := u.Extra(&readBook); err != nil {
		t.Errorf("Failed to read extra: %s", describe(err))
	} else if !reflect.DeepEqual(book, readBook) {
		t.Errorf("Extra differs; expected '%+v', got '%+v'", book, readBook)
	}
	if u.IsActive() != true {
		t.Errorf("Expected IsActive to be true, got false")
	}
}
