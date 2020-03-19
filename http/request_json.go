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
	"github.com/arangodb/go-driver/internal/pkg"
	"io"
	"net/http"

	"net/url"

	"strconv"
	"strings"

	driver "github.com/arangodb/go-driver"
)

// httpRequest implements driver.Request using standard golang http requests.
type httpRequest struct {
	pkg.RequestInternal
}

// createHTTPRequest creates a golang http.Request based on the configured arguments.
func (r *httpRequest) createHTTPRequest(endpoint url.URL) (*http.Request, error) {
	r.Wrote = false
	u := endpoint
	u.Path = ""
	url := u.String()
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	p := r.Path()
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	url = url + p
	if r.Q != nil {
		q := r.Q.Encode()
		if len(q) > 0 {
			url = url + "?" + q
		}
	}

	var bodyReader io.Reader
	body := r.BodyBuilder.GetBody()
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(r.Method(), url, bodyReader)
	if err != nil {
		return nil, driver.WithStack(err)
	}

	if r.Hdr != nil {
		for k, v := range r.Hdr {
			req.Header.Set(k, v)
		}
	}

	if r.VelocyPack {
		req.Header.Set("Accept", "application/x-velocypack")
	}

	if body != nil {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
		req.Header.Set("Content-Type", r.BodyBuilder.GetContentType())
	}
	return req, nil
}
