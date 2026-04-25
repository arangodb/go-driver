//
// DISCLAIMER
//
// Copyright 2024-2026 ArangoDB GmbH, Cologne, Germany
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
	if err := ValidateEndpointSchemes(endpoint); err != nil {
		panic("connection: invalid endpoint configuration: " + err.Error())
	}
	mods := []Mod[Http2Configuration]{
		WithHTT2PEndpoint(endpoint),
	}
	if IsCleartextEndpoint(endpoint) {
		// h2c: cleartext HTTP/2 for http:// endpoints, regardless of insecureSkipVerify
		mods = append(mods, WithHTTP2Transport(DefaultHTTP2TransportSettings, WithHTTP2Cleartext))
	} else if insecureSkipVerify {
		// HTTPS: TLS with certificate verification skipped (e.g. self-signed certs)
		mods = append(mods, WithHTTP2Transport(DefaultHTTP2TransportSettings, WithHTTP2InsecureSkipVerify))
	} else {
		// HTTPS: standard TLS
		mods = append(mods, WithHTTP2Transport(DefaultHTTP2TransportSettings))
	}
	return New[Http2Configuration](mods...)
}

// WithHTTP2Cleartext configures h2c (HTTP/2 cleartext) transport for plain-HTTP endpoints.
// Use this modifier when connecting to http:// endpoints. ArangoDB-native tcp:// endpoints
// must be normalized with FixupEndpointURLScheme before use.
func WithHTTP2Cleartext(in *http2.Transport) {
	in.AllowHTTP = true
	in.DialTLSContext = func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
		return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, network, addr)
	}
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

// WithHTTP2InsecureSkipVerify configures TLS certificate verification to be skipped for HTTPS
// endpoints (e.g. self-signed certificates). For cleartext endpoints use WithHTTP2Cleartext.
func WithHTTP2InsecureSkipVerify(in *http2.Transport) {
	if in.TLSClientConfig == nil {
		in.TLSClientConfig = &tls.Config{}
	}
	in.TLSClientConfig.InsecureSkipVerify = true

	in.DialTLSContext = func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
		if cfg == nil {
			cfg = in.TLSClientConfig
		}
		var tlsCfg *tls.Config
		if cfg != nil {
			tlsCfg = cfg.Clone()
		} else {
			tlsCfg = &tls.Config{}
		}
		tlsCfg.InsecureSkipVerify = true
		return (&tls.Dialer{
			NetDialer: &net.Dialer{Timeout: 30 * time.Second},
			Config:    tlsCfg,
		}).DialContext(ctx, network, addr)
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
