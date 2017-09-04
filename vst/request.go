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
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	driver "github.com/arangodb/go-driver"
	velocypack "github.com/arangodb/go-velocypack"
)

// vstRequest implements driver.Request using Velocystream.
type vstRequest struct {
	method  string
	path    string
	q       url.Values
	hdr     map[string]string
	body    []byte
	written bool
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
	switch len(body) {
	case 0:
		return r, driver.WithStack(fmt.Errorf("Must provide at least 1 body"))
	case 1:
		if data, err := velocypack.Marshal(body[0]); err != nil {
			return r, driver.WithStack(err)
		} else {
			r.body = data
		}
		return r, nil
	default:
		slices := make([]velocypack.Slice, len(body))
		for i, b := range body {
			var err error
			slices[i], err = velocypack.Marshal(b)
			if err != nil {
				return r, driver.WithStack(err)
			}
		}
		merged, err := velocypack.Merge(slices...)
		if err != nil {
			return r, driver.WithStack(err)
		}
		r.body = merged
		return r, nil
	}
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *vstRequest) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	if mergeArray == nil {
		// Simple case; just marshal bodyArray directly.
		if data, err := velocypack.Marshal(bodyArray); err != nil {
			return r, driver.WithStack(err)
		} else {
			r.body = data
		}
		return r, nil
	}

	// Complex case, mergeArray is not nil
	b := velocypack.Builder{}
	// Start array
	if err := b.OpenArray(); err != nil {
		return nil, driver.WithStack(err)
	}

	elementCount := bodyArrayVal.Len()
	for i := 0; i < elementCount; i++ {
		// Marshal body element
		bodySlice, err := velocypack.Marshal(bodyArrayVal.Index(i).Interface())
		if err != nil {
			return nil, driver.WithStack(err)
		}
		var sliceToAdd velocypack.Slice
		if maElem := mergeArray[i]; maElem != nil {
			// Marshal merge array element
			elemSlice, err := velocypack.Marshal(maElem)
			if err != nil {
				return nil, driver.WithStack(err)
			}
			// Merge elemSlice with bodySlice
			sliceToAdd, err = velocypack.Merge(elemSlice, bodySlice)
			if err != nil {
				return nil, driver.WithStack(err)
			}
		} else {
			// Just use bodySlice
			sliceToAdd = bodySlice
		}

		// Add resulting slice
		if err := b.AddValue(velocypack.NewSliceValue(sliceToAdd)); err != nil {
			return nil, driver.WithStack(err)
		}
	}

	// Close array
	if err := b.Close(); err != nil {
		return nil, driver.WithStack(err)
	}

	// Get resulting slice
	arraySlice, err := b.Slice()
	if err != nil {
		return nil, driver.WithStack(err)
	}
	r.body = arraySlice

	return r, nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *vstRequest) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	// Render elements
	buf := &bytes.Buffer{}
	encoder := velocypack.NewEncoder(buf)
	if err := encoder.Encode(bodyArray); err != nil {
		return nil, driver.WithStack(err)
	}
	r.body = buf.Bytes()
	r.SetQuery("type", "list")
	return r, nil
}

// SetHeader sets a single header arguments of the request.
// Any existing header argument with the same key is overwritten.
func (r *vstRequest) SetHeader(key, value string) driver.Request {
	if r.hdr == nil {
		r.hdr = make(map[string]string)
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

// createHTTPRequest creates a golang http.Request based on the configured arguments.
func (r *vstRequest) createMessageParts() ([][]byte, error) {
	r.written = false

	// Build path & database
	path := strings.TrimPrefix(r.path, "/")
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
	for k, v := range r.q {
		if len(v) > 0 {
			b.AddKeyValue(k, velocypack.NewStringValue(v[0]))
		}
	}
	b.Close()      // Parameters
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

	if len(r.body) == 0 {
		return [][]byte{hdr}, nil
	}
	return [][]byte{hdr, r.body}, nil
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
