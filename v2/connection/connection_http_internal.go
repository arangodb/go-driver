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
//
// Author Adam Janikowski
//

package connection

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/arangodb/go-driver/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	_ "golang.org/x/net/http2"
)

const (
	ContentType = "content-type"
)

func NewHTTP2DialForEndpoint(e Endpoint) func(network, addr string, cfg *tls.Config) (net.Conn, error) {
	if len(e.List()) == 0 {
		return nil
	}

	endpoint := e.List()[0]

	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback to default dial
		return nil
	}

	if strings.ToLower(u.Scheme) == "http" || strings.ToLower(u.Scheme) == "tcp" {
		return func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		}
	}

	return nil
}

func newHttpConnection(t http.RoundTripper, contentType string, endpoint Endpoint) *httpConnection {
	c := &http.Client{}
	c.Transport = t

	if contentType == "" {
		contentType = ApplicationJSON
	}

	return &httpConnection{
		transport:   t,
		client:      c,
		endpoint:    endpoint,
		contentType: contentType,
	}
}

type httpConnection struct {
	client *http.Client

	transport http.RoundTripper

	endpoint       Endpoint
	authentication Authentication
	contentType    string

	streamSender bool
}

func (j httpConnection) GetAuthentication() Authentication {
	return j.authentication
}

func (j *httpConnection) SetAuthentication(a Authentication) error {
	j.authentication = a
	return nil
}

func (j httpConnection) Decoder(content string) Decoder {
	switch content {
	case ApplicationVPack:
		return getVPackDecoder()
	case ApplicationJSON:
		return getJsonDecoder()
	default:
		switch j.contentType {
		case ApplicationVPack:
			return getVPackDecoder()
		case ApplicationJSON:
			return getJsonDecoder()
		default:
			return getJsonDecoder()
		}
	}
}

func (j httpConnection) DoWithReader(ctx context.Context, request Request) (Response, io.ReadCloser, error) {
	req, ok := request.(*httpRequest)
	if !ok {
		return nil, nil, errors.Errorf("unable to parse request into JSON Request")
	}
	return j.do(ctx, req)
}

func (j httpConnection) GetEndpoint() Endpoint {
	return j.endpoint
}

func (j *httpConnection) SetEndpoint(e Endpoint) error {
	j.endpoint = e
	return nil
}

func (j httpConnection) NewRequestWithEndpoint(endpoint string, method string, urls ...string) (Request, error) {
	return j.newRequestWithEndpoint(endpoint, method, urls...)
}

func (j httpConnection) NewRequest(method string, urls ...string) (Request, error) {
	return j.newRequest(method, urls...)
}

func (j httpConnection) newRequest(method string, urls ...string) (*httpRequest, error) {
	return j.newRequestWithEndpoint("", method, urls...)
}

func (j httpConnection) newRequestWithEndpoint(endpoint string, method string, urls ...string) (*httpRequest, error) {
	e, ok := j.endpoint.Get(endpoint)
	if !ok {
		return nil, errors.Errorf("Unable to resolve endpoint for %s", e)
	}
	url, err := url.Parse(e)
	if err != nil {
		return nil, err
	}

	url.Path = path.Join(url.Path, path.Join(urls...))

	r := &httpRequest{
		method:   method,
		url:      url,
		endpoint: endpoint,
	}

	return r, nil
}

func (j httpConnection) Do(ctx context.Context, request Request, output interface{}) (Response, error) {
	req, ok := request.(*httpRequest)
	if !ok {
		return nil, errors.Errorf("unable to parse request into JSON Request")
	}
	return j.doWithOutput(ctx, req, output)
}

func (j httpConnection) doWithOutput(ctx context.Context, request *httpRequest, output interface{}) (*httpResponse, error) {
	resp, body, err := j.do(ctx, request)
	if err != nil {
		return nil, err
	}

	if output != nil {
		defer dropBodyData(body) // In case if there is data drop it all

		if err = j.Decoder(resp.Content()).Decode(body, output); err != nil {
			if err != io.EOF {
				return nil, errors.WithStack(err)
			}
		}
	} else {
		// We still need to read data from request, but we can do this in background and ignore output
		defer dropBodyData(body)
	}

	return resp, nil
}

func (j httpConnection) do(ctx context.Context, req *httpRequest) (*httpResponse, io.ReadCloser, error) {
	id := uuid.New().String()
	log.Debugf("(%s) Sending request to %s/%s", id, req.Method(), req.URL())
	if v, ok := req.GetHeader(ContentType); !ok || v == "" {
		req.AddHeader(ContentType, j.contentType)
	}
	if v, ok := req.GetHeader("Accept"); !ok || v == "" {
		req.AddHeader("Accept", j.contentType)
	}

	if auth := j.authentication; auth != nil {
		if err := auth.RequestModifier(req); err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	var httpReq *http.Request

	if ctx == nil {
		ctx = context.Background()
	}

	if req.Method() == http.MethodPost || req.Method() == http.MethodPut || req.Method() == http.MethodPatch {
		decoder := j.Decoder(j.contentType)
		if !j.streamSender {
			b := bytes.NewBuffer([]byte{})
			if err := decoder.Encode(b, req.body); err != nil {
				return nil, nil, err
			}

			r, err := req.asRequest(ctx, b)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}

			httpReq = r
		} else {
			reader, writer := io.Pipe()
			go func() {
				defer writer.Close()
				if err := decoder.Encode(writer, req.body); err != nil {
					writer.CloseWithError(err)
				}
			}()

			r, err := req.asRequest(ctx, reader)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}

			httpReq = r
		}
	} else {
		r, err := req.asRequest(ctx, nil)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		httpReq = r
	}

	resp, err := j.client.Do(httpReq)
	if err != nil {
		log.Debugf("(%s) Request failed: %s", id, err.Error())
		return nil, nil, errors.WithStack(err)
	}
	log.Debugf("(%s) Response received: %d", id, resp.StatusCode)

	if b := resp.Body; b != nil {
		var body = resp.Body

		return &httpResponse{response: resp, request: req}, body, nil

	}

	return &httpResponse{response: resp, request: req}, nil, nil
}
