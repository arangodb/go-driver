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
	method      string
	path        string
	q           url.Values
	hdr         map[string]string
	written     bool
	bodyBuilder driver.BodyBuilder
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

	clone.bodyBuilder = r.bodyBuilder.Clone()
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
	return r, r.bodyBuilder.SetBody(body...)
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *httpVPackRequest) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
	return r, r.bodyBuilder.SetBodyArray(bodyArray, mergeArray)
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *httpVPackRequest) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {

	err := r.bodyBuilder.SetBodyImportArray(bodyArray)
	if err != nil {
		r.SetQuery("type", "list")
	}

	return r, err
}

// SetHeader sets a single header arguments of the request.
// Any existing header argument with the same key is overwritten.
func (r *httpVPackRequest) SetHeader(key, value string) driver.Request {
	if r.hdr == nil {
		r.hdr = make(map[string]string)
	}

	if strings.EqualFold(key, "Content-Type") {
		switch strings.ToLower(value) {
		case "application/octet-stream":
		case "application/zip":
			r.bodyBuilder = NewBinaryBodyBuilder(strings.ToLower(value))
		}
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

	var bodyReader io.Reader
	body := r.bodyBuilder.GetBody()
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(r.method, url, bodyReader)
	if err != nil {
		return nil, driver.WithStack(err)
	}

	if r.hdr != nil {
		for k, v := range r.hdr {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("Accept", "application/x-velocypack")
	if body != nil {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
		req.Header.Set("Content-Type", r.bodyBuilder.GetContentType())
	}
	return req, nil
}

type velocyPackBody struct {
	body []byte
}

func NewVelocyPackBodyBuilder() *velocyPackBody {
	return &velocyPackBody{}
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBody(body ...interface{}) error {

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
	case 2:
		mo := mergeObject{Object: body[1], Merge: body[0]}
		if data, err := velocypack.Marshal(mo); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	default:
		return driver.WithStack(fmt.Errorf("Must provide at most 2 bodies"))
	}

	return nil
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) error {

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
		return driver.WithStack(err)
	} else {
		b.body = data
	}
	return nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *velocyPackBody) SetBodyImportArray(bodyArray interface{}) error {
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

func (b *velocyPackBody) GetBody() []byte {
	return b.body
}

func (b *velocyPackBody) GetContentType() string {
	return "application/x-velocypack"
}

func (b *velocyPackBody) Clone() driver.BodyBuilder {
	return &velocyPackBody{
		body: b.GetBody(),
	}
}
