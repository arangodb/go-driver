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

package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	driver "github.com/arangodb/go-driver"
	velocypack "github.com/arangodb/go-velocypack"
)

// httpVPackRequest implements driver.Request using standard golang http requests.
type httpVPackRequest struct {
	method  string
	path    string
	q       url.Values
	hdr     map[string]string
	body    []byte
	written bool
}

// Path returns the Request path
func (r *httpVPackRequest) Path() string {
	return r.path
}

// Method returns the Request method
func (r *httpVPackRequest) Method() string {
	return r.method
}

// Clone creates a new request containing the same data as this request
func (r *httpVPackRequest) Clone() driver.Request {
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
func (r *httpVPackRequest) SetQuery(key, value string) driver.Request {
	if r.q == nil {
		r.q = url.Values{}
	}
	r.q.Set(key, value)
	return r
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *httpVPackRequest) SetBody(body ...interface{}) (driver.Request, error) {
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
	case 2:
		mo := mergeObject{Object: body[1], Merge: body[0]}
		if data, err := velocypack.Marshal(mo); err != nil {
			return r, driver.WithStack(err)
		} else {
			r.body = data
		}
		return r, nil
	default:
		return r, driver.WithStack(fmt.Errorf("Must provide at most 2 bodies"))
	}

}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *httpVPackRequest) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
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
	elementCount := bodyArrayVal.Len()
	mergeObjects := make([]mergeObject, elementCount)
	for i := 0; i < elementCount; i++ {
		mergeObjects[i] = mergeObject{
			Object: bodyArrayVal.Index(i).Interface(),
			Merge:  mergeArray[i],
		}
	}
	// Now marshal merged array
	if data, err := velocypack.Marshal(mergeObjects); err != nil {
		return r, driver.WithStack(err)
	} else {
		r.body = data
	}
	return r, nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *httpVPackRequest) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {
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
func (r *httpVPackRequest) SetHeader(key, value string) driver.Request {
	if r.hdr == nil {
		r.hdr = make(map[string]string)
	}
	r.hdr[key] = value
	return r
}

// Written returns true as soon as this request has been written completely to the network.
// This does not guarantee that the server has received or processed the request.
func (r *httpVPackRequest) Written() bool {
	return r.written
}

// WroteRequest implements the WroteRequest function of an httptrace.
// It sets written to true.
func (r *httpVPackRequest) WroteRequest(httptrace.WroteRequestInfo) {
	r.written = true
}

// createHTTPRequest creates a golang http.Request based on the configured arguments.
func (r *httpVPackRequest) createHTTPRequest(endpoint url.URL) (*http.Request, error) {
	r.written = false
	u := endpoint
	u.Path = ""
	url := u.String()
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	p := r.path
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	url = url + p
	if r.q != nil {
		q := r.q.Encode()
		if len(q) > 0 {
			url = url + "?" + q
		}
	}
	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req, err := http.NewRequest(r.method, url, body)
	if err != nil {
		return nil, driver.WithStack(err)
	}

	if r.hdr != nil {
		for k, v := range r.hdr {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("Accept", "application/x-velocypack")
	//req.Header.Set("Accept", "application/json")
	if r.body != nil {
		req.Header.Set("Content-Length", strconv.Itoa(len(r.body)))
		req.Header.Set("Content-Type", "application/x-velocypack")
	}
	return req, nil
}
