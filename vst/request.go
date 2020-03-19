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

package vst

import (
	"fmt"
	"github.com/arangodb/go-driver/internal/pkg"
	"strings"

	driver "github.com/arangodb/go-driver"

	velocypack "github.com/arangodb/go-velocypack"
)

// vstRequest implements driver.Request using Velocystream.
type vstRequest struct {
	pkg.RequestInternal
}

// createMessageParts creates a golang http.Request based on the configured arguments.
func (r *vstRequest) createMessageParts() ([][]byte, error) {
	r.Wrote = false

	// Build path & database
	path := strings.TrimPrefix(r.Path(), "/")
	databaseValue := velocypack.NewStringValue("_system")
	if strings.HasPrefix(path, "_db/") {
		path = path[4:] // Remove '_db/'
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 1 {
			databaseValue = velocypack.NewStringValue(parts[0])
			path = ""
		} else {
			databaseValue = velocypack.NewStringValue(parts[0])
			path = parts[1]
		}
	}
	path = "/" + path

	// Create header
	var b velocypack.Builder
	b.OpenArray()
	b.AddValue(velocypack.NewIntValue(1))               // Version
	b.AddValue(velocypack.NewIntValue(1))               // Type (1=Req)
	b.AddValue(databaseValue)                           // Database name
	b.AddValue(velocypack.NewIntValue(r.requestType())) // Request type
	b.AddValue(velocypack.NewStringValue(path))         // Request
	b.OpenObject()                                      // Parameters
	for k, v := range r.Q {
		if len(v) > 0 {
			b.AddKeyValue(k, velocypack.NewStringValue(v[0]))
		}
	}
	b.Close()      // Parameters
	b.OpenObject() // Meta
	for k, v := range r.Hdr {
		b.AddKeyValue(k, velocypack.NewStringValue(v))
	}
	b.Close() // Meta
	b.Close() // Header

	hdr, err := b.Bytes()
	if err != nil {
		return nil, driver.WithStack(err)
	}

	if len(r.BodyBuilder.GetBody()) == 0 {
		return [][]byte{hdr}, nil
	}
	return [][]byte{hdr, r.BodyBuilder.GetBody()}, nil
}

// requestType converts method to request type.
func (r *vstRequest) requestType() int64 {
	switch r.Method() {
	case "DELETE":
		return 0
	case "GET":
		return 1
	case "POST":
		return 2
	case "PUT":
		return 3
	case "HEAD":
		return 4
	case "PATCH":
		return 5
	case "OPTIONS":
		return 6
	default:
		panic(fmt.Errorf("Unknown method '%s'", r.Method()))
	}
}
