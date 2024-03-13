//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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
	"bytes"
	"compress/gzip"
	"compress/zlib"
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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/log"
)

const (
	ContentType = "content-type"
)

func NewHTTP2DialForEndpoint(e Endpoint) func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
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
		return func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		}
	}

	return nil
}

func newHttpConnection(t http.RoundTripper, contentType string, endpoint Endpoint, config ArangoDBConfiguration) *httpConnection {
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
		config:      config,
	}
}

type httpConnection struct {
	client *http.Client

	transport http.RoundTripper

	endpoint       Endpoint
	authentication Authentication
	contentType    string

	streamSender bool

	config ArangoDBConfiguration
}

func (j *httpConnection) GetAuthentication() Authentication {
	return j.authentication
}

func (j *httpConnection) SetAuthentication(a Authentication) error {
	j.authentication = a
	return nil
}

// Decoder returns the decoder according to the response content type or HTTP connection request content type.
// If the content type is unknown, then it returns default JSON decoder.
func (j *httpConnection) Decoder(contentType string) Decoder {
	// First, try to get decoder by the content type of the response.
	if decoder := getDecoderByContentType(contentType); decoder != nil {
		return decoder
	}

	// Next, try to get decoder by the content type of the HTTP connection.
	if decoder := getDecoderByContentType(j.contentType); decoder != nil {
		return decoder
	}

	// Return the default decoder.
	return getJsonDecoder()
}

func (j *httpConnection) GetEndpoint() Endpoint {
	return j.endpoint
}

func (j *httpConnection) SetEndpoint(e Endpoint) error {
	j.endpoint = e
	return nil
}

func (j *httpConnection) GetConfiguration() ArangoDBConfiguration {
	return j.config
}

func (j *httpConnection) SetConfiguration(config ArangoDBConfiguration) {
	j.config = config
}

func (j *httpConnection) NewRequestWithEndpoint(endpoint string, method string, urlParts ...string) (Request, error) {
	return j.newRequestWithEndpoint(endpoint, method, urlParts...)
}

func (j *httpConnection) NewRequest(method string, urlParts ...string) (Request, error) {
	return j.newRequestWithEndpoint("", method, urlParts...)
}

func (j *httpConnection) newRequestWithEndpoint(endpoint string, method string, urlParts ...string) (*httpRequest, error) {
	urlPath := path.Join(urlParts...)

	e, err := j.endpoint.Get(endpoint, method, urlPath)
	if err != nil {
		return nil, errors.Errorf("Unable to resolve endpoint for %s", endpoint)
	}
	u, err := url.Parse(e)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, urlPath)

	r := &httpRequest{
		method:   method,
		url:      u,
		endpoint: e,
	}

	return r, nil
}

// Do perform HTTP request and returns the response.
// If `output` is provided, then it is populated from response body and the response is automatically freed.
func (j *httpConnection) Do(ctx context.Context, request Request, output interface{}, allowedStatusCodes ...int) (Response, error) {
	resp, body, err := j.Stream(ctx, request)
	if err != nil {
		return resp, err
	}

	// The body should be closed at the end of the function.
	defer func(closer io.ReadCloser) {
		err := dropBodyData(closer)
		if err != nil {
			log.Errorf(err, "error closing body")
		}
	}(body)

	if len(allowedStatusCodes) > 0 {
		found := false
		for _, e := range allowedStatusCodes {
			if resp.Code() == e {
				found = true
				break
			}
		}
		if !found {
			var respStruct shared.Response
			// try parse as ArangoDB error response
			_ = j.Decoder(resp.Content()).Decode(body, &respStruct)
			return resp, respStruct.AsArangoErrorWithCode(resp.Code())
		}
	}

	if output != nil {
		// The output should be stored in the output variable.
		if err = j.Decoder(resp.Content()).Decode(body, output); err != nil {
			if err != io.EOF {
				return resp, errors.WithStack(err)
			}
		}
	}

	return resp, nil
}

// Stream performs HTTP request.
// It returns the response and body reader to read the data from there.
// The caller is responsible for free the response body.
func (j *httpConnection) Stream(ctx context.Context, request Request) (Response, io.ReadCloser, error) {
	req, ok := request.(*httpRequest)
	if !ok {
		return nil, nil, errors.Errorf("unable to parse request into JSON Request")
	}

	return j.stream(ctx, req)
}

// stream performs the HTTP request. It returns HTTP response and body reader to read the data from there.
func (j *httpConnection) stream(ctx context.Context, req *httpRequest) (*httpResponse, io.ReadCloser, error) {
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

	reader := j.bodyReadFunc(j.Decoder(j.contentType), req, j.streamSender)
	r, err := req.asRequest(ctx, reader)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	httpReq = r

	resp, err := j.client.Do(httpReq)
	if err != nil {
		log.Debugf("(%s) Request failed: %s", id, err.Error())
		return nil, nil, errors.WithStack(err)
	}
	log.Debugf("(%s) Response received: %d", id, resp.StatusCode)

	if b := resp.Body; b != nil {
		var resultBody io.ReadCloser

		respEncoding := resp.Header.Get("Content-Encoding")
		switch respEncoding {
		case "gzip":
			resultBody, err = gzip.NewReader(resp.Body)
		case "deflate":
			resultBody, err = zlib.NewReader(resp.Body)
		default:
			resultBody = resp.Body
		}

		return &httpResponse{response: resp, request: req}, resultBody, nil

	}

	return &httpResponse{response: resp, request: req}, nil, nil
}

// getDecoderByContentType returns the decoder according to the content type.
// If contentType is unknown, then nil is returned.
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

func (j *httpConnection) bodyReadFunc(decoder Decoder, req *httpRequest, stream bool) bodyReadFactory {
	if req.body == nil {
		return func() (io.Reader, error) {
			return nil, nil
		}
	}

	if !stream {
		return func() (io.Reader, error) {
			b := bytes.NewBuffer([]byte{})
			compressedWriter, err := newCompression(j.config.Compression).ApplyRequestCompression(req, b)
			if err != nil {
				log.Errorf(err, "error applying compression")
				return nil, err
			}

			if compressedWriter != nil {
				defer func(compressedWriter io.WriteCloser) {
					errCompression := compressedWriter.Close()
					if errCompression != nil {
						log.Error(errCompression, "error closing compressed writer")
						if err == nil {
							err = errCompression
						}
					}
				}(compressedWriter)

				err = decoder.Encode(compressedWriter, req.body)
			} else {
				err = decoder.Encode(b, req.body)
			}

			if err != nil {
				log.Errorf(err, "error encoding body - OBJ: %v", req.body)
				return nil, err
			}
			return b, err
		}
	} else {
		return func() (io.Reader, error) {
			reader, writer := io.Pipe()

			compressedWriter, err := newCompression(j.config.Compression).ApplyRequestCompression(req, writer)
			if err != nil {
				log.Errorf(err, "error applying compression")
				return nil, err
			}

			go func() {
				defer writer.Close()

				var encErr error
				if compressedWriter != nil {
					defer func(compressedWriter io.WriteCloser) {
						errCompression := compressedWriter.Close()
						if errCompression != nil {
							log.Errorf(errCompression, "error closing compressed writer - stream")
							writer.CloseWithError(err)
						}
					}(compressedWriter)

					encErr = decoder.Encode(compressedWriter, req.body)
				} else {
					encErr = decoder.Encode(writer, req.body)
				}

				if encErr != nil {
					log.Errorf(err, "error encoding body stream - OBJ: %v", req.body)
					writer.CloseWithError(err)
				}
			}()
			return reader, nil
		}
	}
}
