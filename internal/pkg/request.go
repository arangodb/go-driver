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
// Author Tomasz Mielech
//

package pkg

import (
	"net/http/httptrace"
	"net/url"
	"strings"

	driver "github.com/arangodb/go-driver"
)

// RequestInternal implements only driver.Request using standard golang http requests.
// This Structure can not be imported because it resides in internal directory
// and can be used only by go-driver project.
type RequestInternal struct {
	MethodName  string
	PathName    string
	Q           url.Values
	Hdr         map[string]string
	Wrote       bool
	BodyBuilder driver.BodyBuilder
	VelocyPack  bool
}

// Path returns the Request path
func (r *RequestInternal) Path() string {
	return r.PathName
}

// Method returns the Request method
func (r *RequestInternal) Method() string {
	return r.MethodName
}

// Clone creates a new request containing the same data as this request
func (r *RequestInternal) Clone() driver.Request {
	clone := *r
	clone.Q = url.Values{}
	for k, v := range r.Q {
		for _, x := range v {
			clone.Q.Add(k, x)
		}
	}
	if clone.Hdr != nil {
		clone.Hdr = make(map[string]string)
		for k, v := range r.Hdr {
			clone.Hdr[k] = v
		}
	}

	clone.BodyBuilder = r.BodyBuilder.Clone()
	return &clone
}

// SetQuery sets a single query argument of the request.
// Any existing query argument with the same key is overwritten.
func (r *RequestInternal) SetQuery(key, value string) driver.Request {
	if r.Q == nil {
		r.Q = url.Values{}
	}
	r.Q.Set(key, value)
	return r
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *RequestInternal) SetBody(body ...interface{}) (driver.Request, error) {
	return r, r.BodyBuilder.SetBody(body...)
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *RequestInternal) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
	return r, r.BodyBuilder.SetBodyArray(bodyArray, mergeArray)
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (r *RequestInternal) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {
	err := r.BodyBuilder.SetBodyImportArray(bodyArray)
	if err == nil {
		if r.VelocyPack {
			r.SetQuery("type", "list")
		}
	}

	return r, err
}

// SetHeader sets a single header arguments of the request.
// Any existing header argument with the same key is overwritten.
func (r *RequestInternal) SetHeader(key, value string) driver.Request {
	if r.Hdr == nil {
		r.Hdr = make(map[string]string)
	}

	if strings.EqualFold(key, "Content-Type") {
		switch strings.ToLower(value) {
		case "application/octet-stream":
		case "application/zip":
			r.BodyBuilder = NewBinaryBodyBuilder(strings.ToLower(value))
		}
	}

	r.Hdr[key] = value
	return r
}

// Written returns true as soon as this request has been written completely to the network.
// This does not guarantee that the server has received or processed the request.
func (r *RequestInternal) Written() bool {
	return r.Wrote
}

// WroteRequest implements the WroteRequest function of an httptrace.
// It sets written to true.
func (r *RequestInternal) WroteRequest(httptrace.WroteRequestInfo) {
	r.Wrote = true
}
