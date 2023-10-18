//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

package vst

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	velocypack "github.com/arangodb/go-velocypack"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

// vstRequest implements driver.Request using Velocystream.
type vstRequest struct {
	method      string
	path        string
	q           url.Values
	hdr         map[string]string
	written     bool
	bodyBuilder driver.BodyBuilder
}

// Path returns the Request path
func (r *vstRequest) Path() string {
	return r.path
}

// Method returns the Request method
func (r *vstRequest) Method() string {
	return r.method
}

// Clone creates a new request containing the same data as this request
func (r *vstRequest) Clone() driver.Request {
	clone := *r
	clone.q = url.Values{}
	for k, v := range r.q {
		for _, x := range v {
			clone.q.Add(k, x)
		}
	}
	if clone.hdr != nil {
		clone.hdr = make(map[string]string)
		for k, v := range r.hdr {
			clone.hdr[k] = v
		}
	}

	clone.bodyBuilder = r.bodyBuilder.Clone()
	return &clone
}

// SetQuery sets a single query argument of the request.
// Any existing query argument with the same key is overwritten.
func (r *vstRequest) SetQuery(key, value string) driver.Request {
	if r.q == nil {
		r.q = url.Values{}
	}
	r.q.Set(key, value)
	return r
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
// When multiple bodies are given, they are merged, with fields in the first document prevailing.
func (r *vstRequest) SetBody(body ...interface{}) (driver.Request, error) {
	return r, r.bodyBuilder.SetBody(body...)
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *vstRequest) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
	return r, r.bodyBuilder.SetBodyArray(bodyArray, mergeArray)
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *vstRequest) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {
	err := r.bodyBuilder.SetBodyImportArray(bodyArray)
	if err == nil {
		r.SetQuery("type", "list")
	}
	return r, err
}

// SetHeader sets a single header arguments of the request.
// Any existing header argument with the same key is overwritten.
func (r *vstRequest) SetHeader(key, value string) driver.Request {
	if r.hdr == nil {
		r.hdr = make(map[string]string)
	}

	if strings.EqualFold(key, "Content-Type") {
		switch strings.ToLower(value) {
		case "application/octet-stream":
		case "application/zip":
			r.bodyBuilder = http.NewBinaryBodyBuilder(strings.ToLower(value))
		}
	}

	r.hdr[key] = value
	return r
}

// Written returns true as soon as this request has been written completely to the network.
// This does not guarantee that the server has received or processed the request.
func (r *vstRequest) Written() bool {
	return r.written
}

// WroteRequest sets written to true.
func (r *vstRequest) WroteRequest() {
	r.written = true
}

// createMessageParts creates a golang http.Request based on the configured arguments.
func (r *vstRequest) createMessageParts() ([][]byte, error) {
	r.written = false

	// Build path & database
	path := strings.TrimPrefix(r.path, "/")
	databaseValue := velocypack.NewStringValue("_system")
	if strings.HasPrefix(path, "_db/") {
		path = path[4:] // Remove '_db/'
		parts := strings.SplitN(path, "/", 2)

		// ensure database name is not URL-encoded
		dbName, err := url.QueryUnescape(parts[0])
		if err != nil {
			return nil, driver.WithStack(err)
		}
		databaseValue = velocypack.NewStringValue(dbName)

		if len(parts) == 1 {
			path = ""
		} else {
			path = parts[1]
		}
	}
	path = "/" + path

	// Create header
	var b velocypack.Builder
	b.OpenArray()

	// member 0: numeric version of the velocypack protocol. Must always be 1 at the moment.
	b.AddValue(velocypack.NewIntValue(1))

	// member 1: numeric representation of the VST request type. Must always be 1 at the moment.
	b.AddValue(velocypack.NewIntValue(1))

	// member 2: string with the database name - this must be the normalized database name, but not URL-encoded in any way!
	b.AddValue(databaseValue) // Database name

	// member 3: numeric representation of the request type (GET, POST, PUT etc.)
	b.AddValue(velocypack.NewIntValue(r.requestType()))

	// member 4: string with a relative request path, starting at /
	// There is no need for this path to contain the database name, as the database name is already transferred in member 2.
	// There is also no need for the path to contain request parameters (e.g. key=value), as they should be transferred in member 5.
	b.AddValue(velocypack.NewStringValue(path))

	// member 5:  object with request parameters (e.g. { "foo": "bar", "baz": "qux" }
	b.OpenObject()
	for k, v := range r.q {
		if len(v) > 0 {
			b.AddKeyValue(k, velocypack.NewStringValue(v[0]))
		}
	}
	b.Close()

	// member 6: object with “HTTP” headers (e.g. { "x-arango-async" : "store" }
	b.OpenObject() // Meta
	for k, v := range r.hdr {
		b.AddKeyValue(k, velocypack.NewStringValue(v))
	}
	b.Close() // Meta
	b.Close() // Header

	hdr, err := b.Bytes()
	if err != nil {
		return nil, driver.WithStack(err)
	}

	if len(r.bodyBuilder.GetBody()) == 0 {
		return [][]byte{hdr}, nil
	}
	return [][]byte{hdr, r.bodyBuilder.GetBody()}, nil
}

// requestType converts method to request type.
func (r *vstRequest) requestType() int64 {
	switch r.method {
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
		panic(fmt.Errorf("Unknown method '%s'", r.method))
	}
}

type vstBody struct {
	body []byte
}

func NewVstBodyBuilder() *vstBody {
	return &vstBody{}
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBody(body ...interface{}) error {
	switch len(body) {
	case 0:
		return driver.WithStack(fmt.Errorf("Must provide at least 1 body"))
	case 1:
		if data, err := velocypack.Marshal(body[0]); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	default:
		slices := make([]velocypack.Slice, len(body))
		for i, b := range body {
			var err error
			slices[i], err = velocypack.Marshal(b)
			if err != nil {
				return driver.WithStack(err)
			}
		}
		merged, err := velocypack.Merge(slices...)
		if err != nil {
			return driver.WithStack(err)
		}
		b.body = merged
		return nil
	}
	return nil
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	if mergeArray == nil {
		// Simple case; just marshal bodyArray directly.
		if data, err := velocypack.Marshal(bodyArray); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	}

	// Complex case, mergeArray is not nil
	builder := velocypack.Builder{}
	// Start array
	if err := builder.OpenArray(); err != nil {
		return driver.WithStack(err)
	}

	elementCount := bodyArrayVal.Len()
	for i := 0; i < elementCount; i++ {
		// Marshal body element
		bodySlice, err := velocypack.Marshal(bodyArrayVal.Index(i).Interface())
		if err != nil {
			return driver.WithStack(err)
		}
		var sliceToAdd velocypack.Slice
		if maElem := mergeArray[i]; maElem != nil {
			// Marshal merge array element
			elemSlice, err := velocypack.Marshal(maElem)
			if err != nil {
				return driver.WithStack(err)
			}
			// Merge elemSlice with bodySlice
			sliceToAdd, err = velocypack.Merge(elemSlice, bodySlice)
			if err != nil {
				return driver.WithStack(err)
			}
		} else {
			// Just use bodySlice
			sliceToAdd = bodySlice
		}

		// Add resulting slice
		if err := builder.AddValue(velocypack.NewSliceValue(sliceToAdd)); err != nil {
			return driver.WithStack(err)
		}
	}

	// Close array
	if err := builder.Close(); err != nil {
		return driver.WithStack(err)
	}

	// Get resulting slice
	arraySlice, err := builder.Slice()
	if err != nil {
		return driver.WithStack(err)
	}

	b.body = arraySlice
	return nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBodyImportArray(bodyArray interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	// Render elements
	buf := &bytes.Buffer{}
	encoder := velocypack.NewEncoder(buf)
	if err := encoder.Encode(bodyArray); err != nil {
		return driver.WithStack(err)
	}

	b.body = buf.Bytes()
	return nil
}

func (b *vstBody) GetBody() []byte {
	return b.body
}

func (b *vstBody) GetContentType() string {
	return ""
}

func (b *vstBody) Clone() driver.BodyBuilder {
	return &vstBody{
		body: b.GetBody(),
	}
}
