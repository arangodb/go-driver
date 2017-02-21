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

import (
	"context"
	"path"
)

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
	CreateDatabase(ctx context.Context, name string, options *CreateDatabaseOptions) (Database, error)
}

// ClientConfig contains all settings needed to create a client.
type ClientConfig struct {
	// Connection is the actual server/cluster connection.
	// See http.NewConnection.
	Connection Connection
	// Authentication implements authentication on the server.
	Authentication Authentication
}

// CreateDatabaseOptions contains options that customize the creating of a database.
type CreateDatabaseOptions struct {
	// List of users to initially create for the new database. User information will not be changed for users that already exist.
	// If users is not specified or does not contain any users, a default user root will be created with an empty string password.
	// This ensures that the new database will be accessible after it is created.
	Users []CreateDatabaseUserOptions `json:"users,omitempty"`
}

// CreateDatabaseUserOptions contains options for creating a single user for a database.
type CreateDatabaseUserOptions struct {
	// Loginname of the user to be created
	UserName string `json:"user,omitempty"`
	// The user password as a string. If not specified, it will default to an empty string.
	Password string `json:"passwd,omitempty"`
	// A flag indicating whether the user account should be activated or not. The default value is true. If set to false, the user won't be able to log into the database.
	Active *bool `json:"active,omitempty"`
	// A JSON object with extra user information. The data contained in extra will be stored for the user but not be interpreted further by ArangoDB.
	Extra interface{} `json:"extra,omitempty"`
}

// NewClient creates a new Client based on the given config setting.
func NewClient(config ClientConfig) (Client, error) {
	if config.Connection == nil {
		return nil, WithStack(InvalidArgumentError{Message: "Connection is not set"})
	}
	conn := config.Connection
	if config.Authentication != nil {
		var err error
		conn, err = newAuthenticatedConnection(conn, config.Authentication)
		if err != nil {
			return nil, WithStack(err)
		}
	}
	return &client{
		conn: conn,
	}, nil
}

// client implements the Client interface.
type client struct {
	conn Connection
}

// Database opens a connection to an existing database.
// If no database with given name exists, an NotFoundError is returned.
func (c *client) Database(ctx context.Context, name string) (Database, error) {
	req, err := c.conn.NewRequest("GET", path.Join("_db", name, "_api/database/current"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	db, err := newDatabase(name, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return db, nil
}

// DatabaseExists returns true if a database with given name exists.
func (c *client) DatabaseExists(ctx context.Context, name string) (bool, error) {
	req, err := c.conn.NewRequest("GET", path.Join("_db", name, "_api/database/current"))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

type getDatabaseResponse struct {
	Result []string `json:"result,omitempty"`
}

// Databases returns a list of all databases found by the client.
func (c *client) Databases(ctx context.Context) ([]Database, error) {
	result, err := c.listDatabases(ctx, path.Join("/_db/_system/_api/database"))
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// AccessibleDatabases returns a list of all databases that can be accessed by the authenticated user.
func (c *client) AccessibleDatabases(ctx context.Context) ([]Database, error) {
	result, err := c.listDatabases(ctx, path.Join("/_db/_system/_api/database/user"))
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// listDatabases returns a list of databases using a GET to the given path.
func (c *client) listDatabases(ctx context.Context, path string) ([]Database, error) {
	req, err := c.conn.NewRequest("GET", path)
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getDatabaseResponse
	if err := resp.ParseBody(&data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Database, 0, len(data.Result))
	for _, name := range data.Result {
		db, err := newDatabase(name, c.conn)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, db)
	}
	return result, nil
}

// CreateDatabase creates a new database with given name and opens a connection to it.
// If the a database with given name already exists, a DuplicateError is returned.
func (c *client) CreateDatabase(ctx context.Context, name string, options *CreateDatabaseOptions) (Database, error) {
	input := struct {
		CreateDatabaseOptions
		Name string `json:"name"`
	}{
		Name: name,
	}
	if options != nil {
		input.CreateDatabaseOptions = *options
	}
	req, err := c.conn.NewRequest("POST", path.Join("_db/_system/_api/database"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return nil, WithStack(err)
	}
	db, err := newDatabase(name, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return db, nil
}
