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
	"net/url"
	"path"
)

// newUser creates a new User implementation.
func newUser(data userData, conn Connection) (User, error) {
	if data.Name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "data.Name is empty"})
	}
	if conn == nil {
		return nil, WithStack(InvalidArgumentError{Message: "conn is nil"})
	}
	return &user{
		data: data,
		conn: conn,
	}, nil
}

type user struct {
	data userData
	conn Connection
}

type userData struct {
	Name           string     `json:"user,omitempty"`
	Active         bool       `json:"active,omitempty"`
	Extra          *RawObject `json:"extra,omitempty"`
	ChangePassword bool       `json:"changePassword,omitempty"`
}

// relPath creates the relative path to this index (`_api/user/<name>`)
func (u *user) relPath() string {
	escapedName := url.QueryEscape(u.data.Name)
	return path.Join("_api", "user", escapedName)
}

// Name returns the name of the user.
func (u *user) Name() string {
	return u.data.Name
}

//  Is this an active user?
func (u *user) IsActive() bool {
	return u.data.Active
}

// Is a password change for this user needed?
func (u *user) IsPasswordChangeNeeded() bool {
	return u.data.ChangePassword
}

// Get extra information about this user that was passed during its creation/update/replacement
func (u *user) Extra(result interface{}) error {
	if u.data.Extra == nil {
		return nil
	}
	if err := u.conn.Unmarshal(*u.data.Extra, result); err != nil {
		return WithStack(err)
	}
	return nil
}

// Remove removes the entire user.
// If the user does not exist, a NotFoundError is returned.
func (u *user) Remove(ctx context.Context) error {
	req, err := u.conn.NewRequest("DELETE", u.relPath())
	if err != nil {
		return WithStack(err)
	}
	resp, err := u.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(202); err != nil {
		return WithStack(err)
	}
	return nil
}

// Update updates individual properties of the user.
// If the user does not exist, a NotFoundError is returned.
func (u *user) Update(ctx context.Context, options UserOptions) error {
	req, err := u.conn.NewRequest("PATCH", u.relPath())
	if err != nil {
		return WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return WithStack(err)
	}
	resp, err := u.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	var data userData
	if err := resp.ParseBody("", &data); err != nil {
		return WithStack(err)
	}
	u.data = data
	return nil
}

// Replace replaces all properties of the user.
// If the user does not exist, a NotFoundError is returned.
func (u *user) Replace(ctx context.Context, options UserOptions) error {
	req, err := u.conn.NewRequest("PUT", u.relPath())
	if err != nil {
		return WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return WithStack(err)
	}
	resp, err := u.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	var data userData
	if err := resp.ParseBody("", &data); err != nil {
		return WithStack(err)
	}
	u.data = data
	return nil
}

type userAccessibleDatabasesResponse struct {
	Result map[string]string `json:"result"`
}

// AccessibleDatabases returns a list of all databases that can be accessed by this user.
func (u *user) AccessibleDatabases(ctx context.Context) ([]Database, error) {
	req, err := u.conn.NewRequest("GET", path.Join(u.relPath(), "database"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := u.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data userAccessibleDatabasesResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Database, 0, len(data.Result))
	for name := range data.Result {
		db, err := newDatabase(name, u.conn)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, db)
	}
	return result, nil
}

// GrantReadWriteAccess grants this user read/write access to the given database.
func (u *user) GrantReadWriteAccess(ctx context.Context, db Database) error {
	if err := u.grant(ctx, db, "rw"); err != nil {
		return WithStack(err)
	}
	return nil
}

// RevokeAccess revokes this user access to the given database.
func (u *user) RevokeAccess(ctx context.Context, db Database) error {
	if err := u.grant(ctx, db, "none"); err != nil {
		return WithStack(err)
	}
	return nil
}

// grant grants or revokes access to a database for this user.
func (u *user) grant(ctx context.Context, db Database, access string) error {
	escapedDbName := url.QueryEscape(db.Name())
	req, err := u.conn.NewRequest("PUT", path.Join(u.relPath(), "database", escapedDbName))
	if err != nil {
		return WithStack(err)
	}
	input := struct {
		Grant string `json:"grant"`
	}{
		Grant: access,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	resp, err := u.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}
