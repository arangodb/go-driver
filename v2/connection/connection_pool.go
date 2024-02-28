//
// DISCLAIMER
//
// Copyright 2021-2024 ArangoDB GmbH, Cologne, Germany
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

package connection

import (
	"context"
	"io"
	"sync"
)

func NewPool(connections int, factory Factory) (Connection, error) {
	var c []Connection
	for i := 0; i < connections; i++ {
		if n, err := factory(); err != nil {
			return nil, err
		} else {
			c = append(c, n)
		}
	}

	return &connectionPool{
		factory:     factory,
		connections: c,
	}, nil
}

type connectionPool struct {
	lock sync.Mutex

	factory     Factory
	connections []Connection

	id int
}

func (c *connectionPool) Stream(ctx context.Context, request Request) (Response, io.ReadCloser, error) {
	return c.connection().Stream(ctx, request)
}

func (c *connectionPool) NewRequest(method string, urls ...string) (Request, error) {
	return c.connections[0].NewRequest(method, urls...)
}

func (c *connectionPool) NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error) {
	return c.connections[0].NewRequestWithEndpoint(endpoint, method, urls...)
}

func (c *connectionPool) Do(ctx context.Context, request Request, output interface{}, allowedStatusCodes ...int) (Response, error) {
	return c.connection().Do(ctx, request, output, allowedStatusCodes...)
}

func (c *connectionPool) GetEndpoint() Endpoint {
	return c.connections[0].GetEndpoint()
}

func (c *connectionPool) SetEndpoint(e Endpoint) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, c := range c.connections {
		base := c.GetEndpoint()
		if err := c.SetEndpoint(e); err != nil {
			c.SetEndpoint(base)
			return err
		}
	}

	return nil
}

func (c *connectionPool) GetConfiguration() ArangoDBConfiguration {
	return c.connections[0].GetConfiguration()
}

func (c *connectionPool) SetConfiguration(config ArangoDBConfiguration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, c := range c.connections {
		c.SetConfiguration(config)
	}
}

func (c *connectionPool) GetAuthentication() Authentication {
	return c.connections[0].GetAuthentication()
}

func (c *connectionPool) SetAuthentication(a Authentication) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, c := range c.connections {
		base := c.GetAuthentication()
		if err := c.SetAuthentication(a); err != nil {
			c.SetAuthentication(base)
			return err
		}
	}

	return nil
}

func (c *connectionPool) Decoder(contentType string) Decoder {
	return c.connections[0].Decoder(contentType)
}

func (c *connectionPool) connection() Connection {
	c.lock.Lock()
	defer c.lock.Unlock()

	id := c.id
	c.id++
	if c.id >= len(c.connections) {
		c.id = 0
	}

	return c.connections[id]
}
