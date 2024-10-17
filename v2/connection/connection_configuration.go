//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

func DefaultHTTPConfigurationWrapper(endpoint Endpoint, insecureSkipVerify bool) HttpConfiguration {
	mods := []Mod[HttpConfiguration]{
		WithHTTPEndpoint(endpoint),
	}
	if insecureSkipVerify {
		mods = append(mods, WithHTTPTransport(DefaultHTTPTransportSettings, WithHTTPInsecureSkipVerify))
	} else {
		mods = append(mods, WithHTTPTransport(DefaultHTTPTransportSettings))
	}
	return New[HttpConfiguration](mods...)
}

func DefaultHTTP2ConfigurationWrapper(endpoint Endpoint, insecureSkipVerify bool) Http2Configuration {
	mods := []Mod[Http2Configuration]{
		WithHTT2PEndpoint(endpoint),
	}
	if insecureSkipVerify {
		mods = append(mods, WithHTTP2Transport(DefaultHTTP2TransportSettings, WithHTTP2InsecureSkipVerify))
	} else {
		mods = append(mods, WithHTTP2Transport(DefaultHTTP2TransportSettings))
	}
	return New[Http2Configuration](mods...)
}

type Mod[T any] func(in *T)

func New[T any](mods ...Mod[T]) T {
	var h T
	for _, mod := range mods {
		mod(&h)
	}
	return h
}

func WithHTTPEndpoint(endpoint Endpoint) Mod[HttpConfiguration] {
	return func(in *HttpConfiguration) {
		in.Endpoint = endpoint
	}
}

func WithHTT2PEndpoint(endpoint Endpoint) Mod[Http2Configuration] {
	return func(in *Http2Configuration) {
		in.Endpoint = endpoint
	}
}

func WithHTTPTransport(mods ...Mod[http.Transport]) Mod[HttpConfiguration] {
	return func(in *HttpConfiguration) {
		t := New[http.Transport](mods...)
		in.Transport = &t
	}
}

func WithHTTP2Transport(mods ...Mod[http2.Transport]) Mod[Http2Configuration] {
	return func(in *Http2Configuration) {
		t := New[http2.Transport](mods...)
		in.Transport = &t
	}
}

func WithHTTPInsecureSkipVerify(in *http.Transport) {
	if in.TLSClientConfig == nil {
		in.TLSClientConfig = &tls.Config{}
	}
	in.TLSClientConfig.InsecureSkipVerify = true
}

func WithHTTP2InsecureSkipVerify(in *http2.Transport) {
	if in.TLSClientConfig == nil {
		in.TLSClientConfig = &tls.Config{}
	}
	in.TLSClientConfig.InsecureSkipVerify = true
	in.AllowHTTP = true

	in.DialTLSContext = func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
		// Use net.Dial for plain TCP connection (h2c)
		return net.DialTimeout(network, addr, 30*time.Second)
	}
}

func DefaultHTTPTransportSettings(in *http.Transport) {
	in.Proxy = http.ProxyFromEnvironment
	in.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 90 * time.Second,
	}).DialContext
	in.MaxIdleConns = 100
	in.IdleConnTimeout = 90 * time.Second
	in.TLSHandshakeTimeout = 10 * time.Second
	in.ExpectContinueTimeout = 1 * time.Second

	if in.TLSClientConfig == nil {
		in.TLSClientConfig = &tls.Config{}
	}

	in.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

func DefaultHTTP2TransportSettings(in *http2.Transport) {
	in.IdleConnTimeout = 90 * time.Second

	if in.TLSClientConfig == nil {
		in.TLSClientConfig = &tls.Config{}
	}
	in.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}
}
