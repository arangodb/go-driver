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

package driver

import "context"

// User provides access to a single user of a single server / cluster of servers.
type User interface {
	// Name returns the name of the user.
	Name() string

	//  Is this an active user?
	IsActive() bool

	// Is a password change for this user needed?
	IsPasswordChangeNeeded() bool

	// Get extra information about this user that was passed during its creation/update/replacement
	Extra(result interface{}) error

	// Remove removes the user.
	// If the user does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Update updates individual properties of the user.
	// If the user does not exist, a NotFoundError is returned.
	Update(ctx context.Context, options UserOptions) error

	// Replace replaces all properties of the user.
	// If the user does not exist, a NotFoundError is returned.
	Replace(ctx context.Context, options UserOptions) error

	// AccessibleDatabases returns a list of all databases that can be accessed by this user.
	AccessibleDatabases(ctx context.Context) ([]Database, error)

	// GrantDatabaseReadWriteAccess grants this user read/write access to the given database.
	GrantDatabaseReadWriteAccess(ctx context.Context, db Database) error

	// GrantDatabaseReadOnlyAccess grants this user read only access to the given database.
	// This function requires ArangoDB 3.2 and up.
	GrantDatabaseReadOnlyAccess(ctx context.Context, db Database) error

	// RevokeDatabaseAccess revokes this user access to the given database.
	RevokeDatabaseAccess(ctx context.Context, db Database) error

	// GetDatabaseAccess gets the access rights for this user to the given database.
	GetDatabaseAccess(ctx context.Context, db Database) (Grant, error)

	// GrantCollectionReadWriteAccess grants this user read/write access to the given collection.
	// This function requires ArangoDB 3.2 and up.
	GrantCollectionReadWriteAccess(ctx context.Context, col Collection) error

	// GrantCollectionReadOnlyAccess grants this user read only access to the given collection.
	// This function requires ArangoDB 3.2 and up.
	GrantCollectionReadOnlyAccess(ctx context.Context, col Collection) error

	// RevokeCollectionAccess revokes this user access to the given collection.
	// This function requires ArangoDB 3.2 and up.
	RevokeCollectionAccess(ctx context.Context, col Collection) error

	// GetCollectionAccess gets the access rights for this user to the given collection.
	GetCollectionAccess(ctx context.Context, col Collection) (Grant, error)

	// GrantReadWriteAccess grants this user read/write access to the given database.
	//
	// Deprecated: use GrantDatabaseReadWriteAccess instead.
	GrantReadWriteAccess(ctx context.Context, db Database) error

	// RevokeAccess revokes this user access to the given database.
	// Deprecated: use RevokeDatabaseAccess instead.
	RevokeAccess(ctx context.Context, db Database) error
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
)
