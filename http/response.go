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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	driver "github.com/arangodb/go-driver"
)

// httpResponse implements driver.Response for standard golang http responses.
type httpResponse struct {
	resp *http.Response
}

// StatusCode returns an HTTP compatible status code of the response.
func (r *httpResponse) StatusCode() int {
	return r.resp.StatusCode
}

// CheckStatus checks if the status of the response equals to one of the given status codes.
// If so, nil is returned.
// If not, an attempt is made to parse an error response in the body and an error is returned.
func (r *httpResponse) CheckStatus(validStatusCodes ...int) error {
	for _, x := range validStatusCodes {
		if x == r.resp.StatusCode {
			// Found valid status code
			return nil
		}
	}
	// Invalid status code, try to parse arango error response.
	var aerr driver.ArangoError
	if err := r.ParseBody(&aerr); err == nil {
		// Found correct arango error.
		return aerr
	}

	// We do not have a valid error code, so we can only create one based on the HTTP status code.
	return driver.ArangoError{
		HasError:     true,
		Code:         r.resp.StatusCode,
		ErrorMessage: fmt.Sprintf("Unexpected status code %d", r.resp.StatusCode),
	}
}

// Body returns a reader for accessing the content of the response.
// Clients have to close this body.
func (r *httpResponse) Body() io.ReadCloser {
	return r.resp.Body
}

// ParseBody performs protocol specific unmarshalling of the response data into the given result.
func (r *httpResponse) ParseBody(result interface{}) error {
	body := r.resp.Body
	defer body.Close()
	if err := json.NewDecoder(body).Decode(result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}
