//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package http

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/pkg/errors"
)

type jsonRequest struct {
	method string

	url *url.URL

	endpoint string

	body interface{}

	headers map[string]string
}

func (j *jsonRequest) Endpoint() string {
	return j.endpoint
}

func (j *jsonRequest) SetFragment(s string) {
	j.url.Fragment = s
}

func (j *jsonRequest) AddQuery(key, value string) {
	q := j.url.Query()

	q.Add(key, value)

	j.url.RawQuery = q.Encode()
}

func (j *jsonRequest) AddHeader(key, value string) {
	if j.headers == nil {
		j.headers = map[string]string{}
	}

	j.headers[key] = value
}

func (j *jsonRequest) SetBody(i interface{}) error {
	if i == nil {
		return nil
	}

	if reflect.TypeOf(i).Kind() != reflect.Ptr && reflect.TypeOf(i).Kind() != reflect.Slice && reflect.TypeOf(i).Kind() != reflect.Array {
		return errors.Errorf("body needs to be pointer")
	}

	j.body = i

	return nil
}

func (j jsonRequest) Method() string {
	return j.method
}

func (j *jsonRequest) URL() string {
	return j.url.String()
}

func (j *jsonRequest) asRequest(ctx context.Context, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, j.Method(), j.URL(), body)
	if err != nil {
		return nil, err
	}

	for key, value := range j.headers {
		r.Header.Add(key, value)
	}

	return r, nil
}
