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

// Client provides access to a single arangodb database server, or an entire cluster of arangodb servers.
type Client interface {
	// Database opens a connection to an existing database.
	// If no database with given name exists, an NotFoundError is returned.
	Database(ctx context.Context, name string) (Database, error)

	// DatabaseExists returns true if a database with given name exists.
	DatabaseExists(ctx context.Context, name string) (bool, error)

	// Databases returns a list of all databases found by the client.
	Databases(ctx context.Context) ([]Database, error)

	// AccessibleDatabases returns a list of all databases that can be accessed by the authenticated user.
	AccessibleDatabases(ctx context.Context) ([]Database, error)

	// CreateDatabase creates a new database with given name and opens a connection to it.
	// If the a database with given name already exists, a DuplicateError is returned.
	CreateDatabase(ctx context.Context, name string) (Database, error)
}

// ClientConfig contains all settings needed to create a client.
type ClientConfig struct {
	// Endpoints holds 1 or more URL's used to connect to the database.
	// In case of a connection to an ArangoDB cluster, you must provide the URL's of all coordinators.
	Endpoints []string
}
