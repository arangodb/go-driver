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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/arangodb/go-driver/pkg/arangodb"
	"github.com/arangodb/go-driver/pkg/connection"
	"github.com/arangodb/go-driver/pkg/log"
	"github.com/pkg/errors"
)

const (
	ContentType     = "content-type"
	ApplicationJSON = "application/json"
)

func NewJSONConnection(config Configuration) (connection.Connection, error) {
	c := &http.Client{}
	c.Transport = &http.Transport{
		TLSClientConfig: config.TLS,
	}

	return &jsonConnection{
		config: config,
		client: c,
	}, nil
}

type jsonConnection struct {
	client *http.Client

	config Configuration
}

func (j jsonConnection) DoWithArray(ctx context.Context, request connection.Request) (connection.Response, connection.Array, error) {
	resp, body, err := j.do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	arr, err := newArray(body)
	if err != nil {
		return nil, nil, err
	}

	return resp, arr, nil
}

func (j jsonConnection) DoWithReader(ctx context.Context, request connection.Request) (connection.Response, io.ReadCloser, error) {
	return j.do(ctx, request)
}

func (j jsonConnection) Endpoint() string {
	if len(j.config.Endpoints) == 0 {
		return ""
	}

	return j.config.Endpoints[rand.Intn(len(j.config.Endpoints))]
}

func (j jsonConnection) NewRequestWithEndpoint(endpoint string, method string, urls ...string) (connection.Request, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	url.Path = path.Join(url.Path, path.Join(urls...))

	r := &jsonRequest{
		method:   method,
		url:      url,
		endpoint: endpoint,
	}

	if auth := j.config.Authentication; auth != nil {
		if err = auth.RequestModifier(r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (j jsonConnection) NewRequest(method string, urls ...string) (connection.Request, error) {
	return j.NewRequestWithEndpoint(j.Endpoint(), method, urls...)
}

func (j jsonConnection) Do(ctx context.Context, request connection.Request) (connection.Response, error) {
	return j.DoWithOutput(ctx, request, nil)
}

func (j jsonConnection) DoWithOutput(ctx context.Context, request connection.Request, output interface{}) (connection.Response, error) {
	if j.config.TraceRequestData {
		return j.doWithOutputTrace(ctx, request, output)
	}

	return j.doWithOutput(ctx, request, output)
}

func (j jsonConnection) doWithOutputTrace(ctx context.Context, request connection.Request, output interface{}) (connection.Response, error) {
	var data []byte
	t := time.Now()

	defer func() {
		if len(data) > 0 {
			log.Tracef("Request %s %s took %s, received %d bytes: %s", request.Method(), request.URL(), time.Since(t), len(data), string(data))
		} else {
			log.Tracef("Request %s %s took %s", request.Method(), request.URL(), time.Since(t))
		}
	}()

	var z interface{}
	var resp connection.Response
	var err error

	if request.Method() != http.MethodHead {
		resp, err = j.doWithOutput(ctx, request, &z)
		if err != nil {
			return nil, err
		}
	} else {
		resp, err = j.doWithOutput(ctx, request, nil)
		if err != nil {
			return nil, err
		}
	}

	data, err = json.Marshal(z)
	if err != nil {
		return nil, err
	}

	if output != nil {
		if err = json.Unmarshal(data, output); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (j jsonConnection) doWithOutput(ctx context.Context, request connection.Request, output interface{}) (connection.Response, error) {
	t := time.Now()

	if j.config.TraceRequestTime {
		defer func() {
			log.Tracef("Request %s %s took %s", request.Method(), request.URL(), time.Since(t))
		}()
	}

	resp, body, err := j.do(ctx, request)
	if err != nil {
		return nil, err
	}

	if output != nil {
		defer dropBodyData(body) // In case if there is data drop it all
		if err = json.NewDecoder(body).Decode(output); err != nil {
			return nil, err
		}
	} else {
		// We still need to read data from request, but we can do this in background and ignore output
		go dropBodyData(body)
	}

	return resp, nil
}

func (j jsonConnection) do(ctx context.Context, request connection.Request) (connection.Response, io.ReadCloser, error) {
	if j.config.TraceRequestTime {
		t := time.Now()
		defer func() {
			log.Tracef("Request %s %s took %s", request.Method(), request.URL(), time.Since(t))
		}()
	}
	request.AddHeader(ContentType, ApplicationJSON)

	req, ok := request.(*jsonRequest)
	if !ok {
		return nil, nil, errors.Errorf("unable to parse request into JSON Request")
	}

	var httpReq *http.Request

	if request.Method() == http.MethodPost || request.Method() == http.MethodPut {
		data, err := json.Marshal(req.body)

		r, err := req.asRequest(ctx, bytes.NewBuffer(data))
		if err != nil {
			return nil, nil, err
		}
		httpReq = r
	} else {
		r, err := req.asRequest(ctx, nil)
		if err != nil {
			return nil, nil, err
		}
		httpReq = r
	}

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}

	if b := resp.Body; b != nil {
		var body = resp.Body

		if isFailCode(resp.StatusCode) {
			defer dropBodyData(resp.Body) // In case if there is data drop it all

			// In case of failure try to get response as json or raw
			switch h := strings.Split(resp.Header.Get(ContentType), ";")[0]; h {
			case ApplicationJSON:
				var errStr arangodb.Response
				if err := json.NewDecoder(body).Decode(&errStr); err != nil {
					return nil, nil, err
				}
				return nil, nil, connection.NewErrorf(resp.StatusCode, "Error: %d, %s", errStr.GetErrorNum(), errStr.GetErrorMessage())
			default:
				data, err := ioutil.ReadAll(body)
				if err != nil {
					return nil, nil, err
				}

				return nil, nil, connection.NewErrorf(resp.StatusCode, "Content: %s, %s", h, string(data))
			}
		} else {
			return &jsonResponse{response: resp, request: req}, body, nil
		}
	}

	return &jsonResponse{response: resp, request: req}, nil, nil
}

func (j jsonConnection) Authentication(auth connection.Authentication) error {
	panic("implement me")
}

func isFailCode(code int) bool {
	if code >= 400 && code < 500 {
		return true
	}

	return false
}

func deferCloser(closer io.ReadCloser) error {
	return closer.Close()
}

func dropBodyData(closer io.ReadCloser) error {
	defer deferCloser(closer)

	b := make([]byte, 1024)
	for {
		_, err := closer.Read(b)
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}
	}
}
