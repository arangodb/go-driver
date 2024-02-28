//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

import "net/http"

type HttpConfiguration struct {
	Authentication Authentication
	Endpoint       Endpoint

	ContentType string

	ArangoDBConfig ArangoDBConfiguration

	Transport http.RoundTripper
}

func (h HttpConfiguration) getTransport() http.RoundTripper {
	if h.Transport != nil {
		return h.Transport
	}

	return &http.Transport{
		MaxIdleConns: 100,
	}
}

func (h HttpConfiguration) GetContentType() string {
	if h.ContentType == "" {
		return ApplicationJSON
	}

	return h.ContentType
}

func NewHttpConnection(config HttpConfiguration) Connection {
	c := newHttpConnection(config.getTransport(), config.ContentType, config.Endpoint, config.ArangoDBConfig)

	if a := config.Authentication; a != nil {
		c.authentication = a
	}

	c.streamSender = false

	return c
}
