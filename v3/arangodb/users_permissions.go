//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import "context"

type UserPermissions interface {
	// AccessibleDatabases returns a list of all databases that can be accessed (read/write or read-only) by this user.
	AccessibleDatabases(ctx context.Context) (map[string]Grant, error)

	// AccessibleDatabasesFull return the full set of access levels for all databases and all collections.
	AccessibleDatabasesFull(ctx context.Context) (map[string]DatabasePermissions, error)

	// GetDatabaseAccess fetch the database access level for a specific database
	GetDatabaseAccess(ctx context.Context, db string) (Grant, error)

	// GetCollectionAccess returns the collection access level for a specific collection
	GetCollectionAccess(ctx context.Context, db, col string) (Grant, error)

	// SetDatabaseAccess sets the access this user has to the given database.
	// Pass a `nil` database to set the default access this user has to any new database.
	// You need the Administrate server access level
	SetDatabaseAccess(ctx context.Context, db string, access Grant) error

	// SetCollectionAccess sets the access this user has to a collection.
	// You need the Administrate server access level
	SetCollectionAccess(ctx context.Context, db, col string, access Grant) error

	// RemoveDatabaseAccess removes the access this user has to the given database.
	// As a consequence, the default database access level is used.
	// If there is no defined default database access level, it defaults to No access.
	// You need a write permissions (Administrate access level) for the '_system' database
	RemoveDatabaseAccess(ctx context.Context, db string) error

	// RemoveCollectionAccess removes the access this user has to a collection.
	// As a consequence, the default collection access level is used.
	// If there is no defined default collection access level, it defaults to No access.
	RemoveCollectionAccess(ctx context.Context, db, col string) error
}

type DatabasePermissions struct {
	// Permission access type for the database.
	Permission Grant `json:"permission"`

	// Collections contain a map with collection names as object keys and the associated privileges for them.
	Collections map[string]Grant `json:"collections"`
}

// Grant specifies access rights for an object
type Grant string

const (
	// GrantReadWrite indicates read/write access to an object
	GrantReadWrite Grant = "rw"

	// GrantReadOnly indicates read-only access to an object
	GrantReadOnly Grant = "ro"

	// GrantNone indicates no access to an object
	GrantNone Grant = "none"

	// GrantUndefined indicates undefined access to an object (read-only operation)
	GrantUndefined Grant = "undefined"
)
