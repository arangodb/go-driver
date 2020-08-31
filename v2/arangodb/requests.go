//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package arangodb

import (
	"context"

	"github.com/arangodb/go-driver/v2/connection"
)

type Requests interface {
	Get(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error)
	Post(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error)
	Put(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error)
	Delete(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error)
	Head(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error)
	Patch(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error)
}

func NewRequests(connection connection.Connection, urlParts ...string) Requests {
	return &requests{
		connection: connection,
		prefix:     urlParts,
	}
}

var _ Requests = &requests{}

type requests struct {
	connection connection.Connection

	prefix []string
}

func (r requests) Patch(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallPatch(ctx, r.connection, r.path(urlParts...), output, input)
}

func (r requests) path(urlParts ...string) string {
	n := make([]string, len(r.prefix)+len(urlParts))
	for id, s := range r.prefix {
		n[id] = s
	}
	for id, s := range urlParts {
		n[id+len(r.prefix)] = s
	}
	return connection.NewUrl(n...)
}

func (r requests) Get(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallGet(ctx, r.connection, r.path(urlParts...), output)
}

func (r requests) Post(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallPost(ctx, r.connection, r.path(urlParts...), output, input)
}

func (r requests) Put(ctx context.Context, output, input interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallPut(ctx, r.connection, r.path(urlParts...), output, input)
}

func (r requests) Delete(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallDelete(ctx, r.connection, r.path(urlParts...), output)
}

func (r requests) Head(ctx context.Context, output interface{}, urlParts ...string) (connection.Response, error) {
	return connection.CallHead(ctx, r.connection, r.path(urlParts...), output)
}
