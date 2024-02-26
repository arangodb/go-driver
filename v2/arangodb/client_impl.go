//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"github.com/arangodb/go-driver/v2/connection"
)

func NewClient(connection connection.Connection) Client {
	return newClient(connection)
}

func newClient(connection connection.Connection) *client {
	c := &client{
		connection: connection,
	}

	c.clientDatabase = newClientDatabase(c)
	c.clientUser = newClientUser(c)
	c.clientServerInfo = newClientServerInfo(c)
	c.clientAdmin = newClientAdmin(c)
	c.clientAsyncJob = newClientAsyncJob(c)

	c.Requests = NewRequests(connection)

	return c
}

var _ Client = &client{}

type client struct {
	connection connection.Connection

	*clientDatabase
	*clientUser
	*clientServerInfo
	*clientAdmin
	*clientAsyncJob

	Requests
}

func (c *client) Connection() connection.Connection {
	return c.connection
}
