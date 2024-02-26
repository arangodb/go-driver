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

import (
	"context"
)

type ClientUsers interface {
	// User opens a connection to an existing user.
	// If no user with given name exists, an NotFoundError is returned.
	User(ctx context.Context, name string) (User, error)

	// UserExists returns true if a user with a given name exists.
	UserExists(ctx context.Context, name string) (bool, error)

	// Users return a list of all users found by the client.
	Users(ctx context.Context) ([]User, error)

	// CreateUser creates a new user with a given name and opens a connection to it.
	// If a user with a given name already exists, a Conflict error is returned.
	CreateUser(ctx context.Context, name string, options *UserOptions) (User, error)

	// ReplaceUser Replaces the data of an existing user.
	ReplaceUser(ctx context.Context, name string, options *UserOptions) (User, error)

	// UpdateUser Partially modifies the data of an existing user
	UpdateUser(ctx context.Context, name string, options *UserOptions) (User, error)

	// RemoveUser removes an existing user.
	RemoveUser(ctx context.Context, name string) error
}

// UserOptions contains options for creating a new user, updating or replacing a user.
type UserOptions struct {
	// The user password as a string. If not specified, it will default to an empty string.
	Password string `json:"passwd,omitempty"`

	// An optional flag that specifies whether the user is active. If not specified, this will default to true.
	Active *bool `json:"active,omitempty"`

	// A JSON object with extra user information.
	// The data contained in extra will be stored for the user but not be interpreted further by ArangoDB.
	Extra interface{} `json:"extra,omitempty"`
}

// User provides access to a single user of a single server / cluster of servers.
type User interface {
	// Name returns the name of the user.
	Name() string

	// IsActive returns whether the user is active.
	IsActive() bool

	// Extra information about this user that was passed during its creation/update/replacement
	Extra(result interface{}) error

	UserPermissions
}
