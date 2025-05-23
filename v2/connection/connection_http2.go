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

import (
	"golang.org/x/net/http2"
)

type Http2Configuration struct {
	Authentication Authentication
	Endpoint       Endpoint

	ContentType string

	ArangoDBConfig ArangoDBConfiguration

	Transport *http2.Transport
}

func (h Http2Configuration) getTransport() *http2.Transport {
	if h.Transport != nil {
		return h.Transport
	}

	return &http2.Transport{AllowHTTP: true}
}

func (h Http2Configuration) GetContentType() string {
	if h.ContentType == "" {
		return ApplicationJSON
	}

	return h.ContentType
}

// NewHttp2Connection
// Warning: Ensure that VST is not enabled to avoid performance issues
func NewHttp2Connection(config Http2Configuration) Connection {
	c := newHttpConnection(config.getTransport(), config.ContentType, config.Endpoint, config.ArangoDBConfig)

	if a := config.Authentication; a != nil {
		c.authentication = a
	}

	c.streamSender = true

	return c
}
