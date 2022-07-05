//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
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

	"github.com/google/uuid"
	"github.com/pkg/errors"
	_ "golang.org/x/net/http2"

	"github.com/arangodb/go-driver/v2/log"
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

// Decoder returns the decoder according to the response content type or HTTP connection request content type.
// If the content type is unknown then it returns default JSON decoder.
func (j httpConnection) Decoder(contentType string) Decoder {
	// First try to get decoder by the content type of the response.
	if decoder := getDecoderByContentType(contentType); decoder != nil {
		return decoder
	}

	// Next try to get decoder by the content type of the HTTP connection.
	if decoder := getDecoderByContentType(j.contentType); decoder != nil {
		return decoder
	}

	// Return the default decoder.
	return getJsonDecoder()
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

// Do performs HTTP request and returns the response.
// If `output` is provided then it is populated from response body and the response is automatically freed.
func (j httpConnection) Do(ctx context.Context, request Request, output interface{}) (Response, error) {
	resp, body, err := j.Stream(ctx, request)
	if err != nil {
		return nil, err
	}

	// The body should be closed at the end of the function.
	defer dropBodyData(body)

	if output != nil {
		// The output should be stored in the output variable.
		if err = j.Decoder(resp.Content()).Decode(body, output); err != nil {
			if err != io.EOF {
				return nil, errors.WithStack(err)
			}
		}
	}

	return resp, nil
}

// Stream performs HTTP request.
// It returns the response and body reader to read the data from there.
// The caller is responsible to free the response body.
func (j httpConnection) Stream(ctx context.Context, request Request) (Response, io.ReadCloser, error) {
	req, ok := request.(*httpRequest)
	if !ok {
		return nil, nil, errors.Errorf("unable to parse request into JSON Request")
	}

	return j.stream(ctx, req)
}

// stream performs the HTTP request.
// It returns HTTP response and body reader to read the data from there.
func (j httpConnection) stream(ctx context.Context, req *httpRequest) (*httpResponse, io.ReadCloser, error) {
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
		reader := j.bodyReadFunc(decoder, req.body, j.streamSender)
		r, err := req.asRequest(ctx, reader)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		httpReq = r
	} else {
		r, err := req.asRequest(ctx, func() (io.Reader, error) {
			return nil, nil
		})
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

// getDecoderByContentType returns the decoder according to the content type.
// If content type is unknown then nil is returned.
func getDecoderByContentType(contentType string) Decoder {
	switch contentType {
	case ApplicationVPack:
		return getVPackDecoder()
	case ApplicationJSON:
		return getJsonDecoder()
	case PlainText, ApplicationOctetStream, ApplicationZip:
		return getBytesDecoder()
	default:
		return nil
	}
}

type bodyReadFactory func() (io.Reader, error)

func (j httpConnection) bodyReadFunc(decoder Decoder, obj interface{}, stream bool) bodyReadFactory {
	if !stream {
		return func() (io.Reader, error) {
			b := bytes.NewBuffer([]byte{})
			if err := decoder.Encode(b, obj); err != nil {
				return nil, err
			}

			return b, nil
		}
	} else {
		return func() (io.Reader, error) {
			reader, writer := io.Pipe()
			go func() {
				defer writer.Close()
				if err := decoder.Encode(writer, obj); err != nil {
					writer.CloseWithError(err)
				}
			}()
			return reader, nil
		}
	}
}
