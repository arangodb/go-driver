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
	"strings"
	"sync"

	velocypack "github.com/arangodb/go-velocypack"

	driver "github.com/arangodb/go-driver"
)

// vstResponse implements driver.Response for Velocystream responses.
type vstResponse struct {
	endpoint     string
	Version      int
	Type         int
	ResponseCode int
	meta         velocypack.Slice
	metaMutex    sync.Mutex
	metaMap      map[string]string
	slice        velocypack.Slice
	bodyArray    []driver.Response
}

// newResponse builds a vstResponse from given message.
func newResponse(msgData []byte, endpoint string, rawResponse *[]byte) (*vstResponse, error) {
	// Decode header
	hdr := velocypack.Slice(msgData)
	if err := hdr.AssertType(velocypack.Array); err != nil {
		return nil, driver.WithStack(err)
	}
	//panic("hdr: " + hex.EncodeToString(hdr))
	var hdrLen velocypack.ValueLength
	if l, err := hdr.Length(); err != nil {
		return nil, driver.WithStack(err)
	} else if l < 3 {
		return nil, driver.WithStack(fmt.Errorf("Expected a header of 3 elements, got %d", l))
	} else {
		hdrLen = l
	}

	resp := &vstResponse{
		endpoint: endpoint,
	}
	// Decode version
	if elem, err := hdr.At(0); err != nil {
		return nil, driver.WithStack(err)
	} else if version, err := elem.GetInt(); err != nil {
		return nil, driver.WithStack(err)
	} else {
		resp.Version = int(version)
	}
	// Decode type
	if elem, err := hdr.At(1); err != nil {
		return nil, driver.WithStack(err)
	} else if tp, err := elem.GetInt(); err != nil {
		return nil, driver.WithStack(err)
	} else {
		resp.Type = int(tp)
	}
	// Decode responseCode
	if elem, err := hdr.At(2); err != nil {
		return nil, driver.WithStack(err)
	} else if code, err := elem.GetInt(); err != nil {
		return nil, driver.WithStack(err)
	} else {
		resp.ResponseCode = int(code)
	}
	// Decode meta
	if hdrLen >= 4 {
		if elem, err := hdr.At(3); err != nil {
			return nil, driver.WithStack(err)
		} else if !elem.IsObject() {
			return nil, driver.WithStack(fmt.Errorf("Expected meta field to be of type Object, got %s", elem.Type()))
		} else {
			resp.meta = elem
		}
	}

	// Fetch body directly after hdr
	if body, err := hdr.Next(); err != nil {
		return nil, driver.WithStack(err)
	} else {
		resp.slice = body
		if rawResponse != nil {
			*rawResponse = body
		}
	}
	//fmt.Printf("got response: code=%d, body=%s\n", resp.ResponseCode, hex.EncodeToString(resp.slice))
	return resp, nil
}

// StatusCode returns an HTTP compatible status code of the response.
func (r *vstResponse) StatusCode() int {
	return r.ResponseCode
}

// Endpoint returns the endpoint that handled the request.
func (r *vstResponse) Endpoint() string {
	return r.endpoint
}

// CheckStatus checks if the status of the response equals to one of the given status codes.
// If so, nil is returned.
// If not, an attempt is made to parse an error response in the body and an error is returned.
func (r *vstResponse) CheckStatus(validStatusCodes ...int) error {
	for _, x := range validStatusCodes {
		if x == r.ResponseCode {
			// Found valid status code
			return nil
		}
	}
	// Invalid status code, try to parse arango error response.
	var aerr driver.ArangoError
	if err := r.ParseBody("", &aerr); err == nil {
		// Found correct arango error.
		return aerr
	}

	// We do not have a valid error code, so we can only create one based on the HTTP status code.
	return driver.ArangoError{
		HasError:     true,
		Code:         r.ResponseCode,
		ErrorMessage: fmt.Sprintf("Unexpected status code %d", r.ResponseCode),
	}
}

// Header returns the value of a response header with given key.
// If no such header is found, an empty string is returned.
func (r *vstResponse) Header(key string) string {
	r.metaMutex.Lock()
	defer r.metaMutex.Unlock()

	if r.meta != nil {
		if r.metaMap == nil {
			// Read all headers
			metaMap := make(map[string]string)
			keyCount, err := r.meta.Length()
			if err != nil {
				return ""
			}
			for k := velocypack.ValueLength(0); k < keyCount; k++ {
				key, err := r.meta.KeyAt(k)
				if err != nil {
					continue
				}
				value, err := r.meta.ValueAt(k)
				if err != nil {
					continue
				}
				keyStr, err := key.GetString()
				if err != nil {
					continue
				}
				valueStr, err := value.GetString()
				if err != nil {
					continue
				}
				metaMap[strings.ToLower(keyStr)] = valueStr
			}
			r.metaMap = metaMap
		}
		key = strings.ToLower(key)
		if value, found := r.metaMap[key]; found {
			return value
		}
	}
	return ""
}

// ParseBody performs protocol specific unmarshalling of the response data into the given result.
// If the given field is non-empty, the contents of that field will be parsed into the given result.
func (r *vstResponse) ParseBody(field string, result interface{}) error {
	slice := r.slice
	if field != "" {
		var err error
		slice, err = slice.Get(field)
		if err != nil {
			return driver.WithStack(err)
		}
		if slice.IsNone() {
			// Field not found
			return nil
		}
	}
	if result != nil {
		if err := velocypack.Unmarshal(slice, result); err != nil {
			return driver.WithStack(err)
		}
	}
	return nil
}

// ParseArrayBody performs protocol specific unmarshalling of the response array data into individual response objects.
// This can only be used for requests that return an array of objects.
func (r *vstResponse) ParseArrayBody() ([]driver.Response, error) {
	if r.bodyArray == nil {
		slice := r.slice
		l, err := slice.Length()
		if err != nil {
			return nil, driver.WithStack(err)
		}

		bodyArray := make([]driver.Response, 0, l)
		it, err := velocypack.NewArrayIterator(slice)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		for it.IsValid() {
			v, err := it.Value()
			if err != nil {
				return nil, driver.WithStack(err)
			}
			bodyArray = append(bodyArray, &vstResponseElement{slice: v})
			it.Next()
		}
		r.bodyArray = bodyArray
	}

	return r.bodyArray, nil
}
