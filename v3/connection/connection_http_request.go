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
	"context"
	"io"
	"net/http"
	"net/url"
)

var _ Request = &httpRequest{}

type httpRequest struct {
	method string

	url *url.URL

	endpoint string

	body interface{}

	headers map[string]string
}

func (j *httpRequest) GetHeader(key string) (string, bool) {
	k, ok := j.headers[key]
	return k, ok
}

func (j *httpRequest) GetQuery(key string) (string, bool) {
	q := j.url.Query().Get(key)
	return q, q != ""
}

func (j *httpRequest) Endpoint() string {
	return j.endpoint
}

func (j *httpRequest) SetFragment(s string) {
	j.url.Fragment = s
}

func (j *httpRequest) AddQuery(key, value string) {
	q := j.url.Query()

	q.Add(key, value)

	j.url.RawQuery = q.Encode()
}

func (j *httpRequest) AddHeader(key, value string) {
	if j.headers == nil {
		j.headers = map[string]string{}
	}

	j.headers[key] = value
}

func (j *httpRequest) SetBody(i interface{}) error {
	if i == nil {
		return nil
	}

	j.body = i

	return nil
}

func (j *httpRequest) Method() string {
	return j.method
}

func (j *httpRequest) URL() string {
	// return unescaped string since it is escaped again in Connection.Do()
	u, _ := url.QueryUnescape(j.url.String())
	return u
}

func (j *httpRequest) asRequest(ctx context.Context, bodyReader bodyReadFactory) (*http.Request, error) {
	body, err := bodyReader()
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, j.Method(), j.URL(), body)
	if err != nil {
		return nil, err
	}

	r.GetBody = func() (io.ReadCloser, error) {
		if body, err := bodyReader(); err != nil {
			return nil, err
		} else if c, ok := body.(io.ReadCloser); ok {
			return c, nil
		} else {
			return io.NopCloser(body), nil
		}
	}

	for key, value := range j.headers {
		r.Header.Add(key, value)
	}

	return r, nil
}
