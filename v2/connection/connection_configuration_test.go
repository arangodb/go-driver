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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IsCleartextEndpoint(t *testing.T) {
	tests := map[string]struct {
		urls []string
		want bool
	}{
		"all http":                 {[]string{"http://a:8529", "http://b:8529"}, true},
		"all tcp (not normalized)": {[]string{"tcp://a:8529"}, false}, // must be normalized via FixupEndpointURLScheme first
		"all https":                {[]string{"https://a:8529", "https://b:8529"}, false},
		"mixed http and https":     {[]string{"http://a:8529", "https://b:8529"}, false},
		"empty list":               {[]string{}, false},
		"single https":             {[]string{"https://a:8529"}, false},
		"single http":              {[]string{"http://a:8529"}, true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ep := NewRoundRobinEndpoints(tc.urls)
			assert.Equal(t, tc.want, IsCleartextEndpoint(ep))
		})
	}
}

func Test_DefaultHTTP2ConfigurationWrapper_TransportSelection(t *testing.T) {
	t.Run("http endpoint uses h2c transport", func(t *testing.T) {
		ep := NewRoundRobinEndpoints([]string{"http://localhost:8529"})
		cfg := DefaultHTTP2ConfigurationWrapper(ep, false)

		require.NotNil(t, cfg.Transport)
		assert.True(t, cfg.Transport.AllowHTTP, "h2c requires AllowHTTP=true")
		assert.NotNil(t, cfg.Transport.DialTLSContext, "h2c requires a custom DialTLSContext for plain TCP")
	})

	t.Run("http endpoint with insecureSkipVerify still uses h2c transport", func(t *testing.T) {
		ep := NewRoundRobinEndpoints([]string{"http://localhost:8529"})
		cfg := DefaultHTTP2ConfigurationWrapper(ep, true)

		require.NotNil(t, cfg.Transport)
		assert.True(t, cfg.Transport.AllowHTTP, "h2c requires AllowHTTP=true")
		assert.NotNil(t, cfg.Transport.DialTLSContext, "h2c requires a custom DialTLSContext for plain TCP")
		assert.False(t, cfg.Transport.TLSClientConfig.InsecureSkipVerify,
			"InsecureSkipVerify should not be set on h2c transport")
	})

	t.Run("https endpoint with insecureSkipVerify=false uses standard TLS", func(t *testing.T) {
		ep := NewRoundRobinEndpoints([]string{"https://localhost:8529"})
		cfg := DefaultHTTP2ConfigurationWrapper(ep, false)

		require.NotNil(t, cfg.Transport)
		assert.False(t, cfg.Transport.AllowHTTP)
		assert.Nil(t, cfg.Transport.DialTLSContext, "standard TLS should not override DialTLSContext")
		require.NotNil(t, cfg.Transport.TLSClientConfig)
		assert.False(t, cfg.Transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("https endpoint with insecureSkipVerify=true skips TLS verification", func(t *testing.T) {
		ep := NewRoundRobinEndpoints([]string{"https://localhost:8529"})
		cfg := DefaultHTTP2ConfigurationWrapper(ep, true)

		require.NotNil(t, cfg.Transport)
		assert.False(t, cfg.Transport.AllowHTTP)
		assert.NotNil(t, cfg.Transport.DialTLSContext, "insecure TLS requires a custom DialTLSContext")
		require.NotNil(t, cfg.Transport.TLSClientConfig)
		assert.True(t, cfg.Transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("mixed-scheme endpoint panics", func(t *testing.T) {
		ep := NewRoundRobinEndpoints([]string{"http://a:8529", "https://b:8529"})
		assert.Panics(t, func() {
			DefaultHTTP2ConfigurationWrapper(ep, false)
		})
	})
}

func Test_ValidateEndpointSchemes(t *testing.T) {
	tests := map[string]struct {
		urls    []string
		wantErr bool
	}{
		"all http":              {[]string{"http://a:8529", "http://b:8529"}, false},
		"all https":             {[]string{"https://a:8529", "https://b:8529"}, false},
		"empty list":            {[]string{}, false},
		"single http":           {[]string{"http://a:8529"}, false},
		"single https":          {[]string{"https://a:8529"}, false},
		"mixed http and https":  {[]string{"http://a:8529", "https://b:8529"}, true},
		"mixed https then http": {[]string{"https://a:8529", "http://b:8529"}, true},
		"tcp (unnormalized)":    {[]string{"tcp://a:8529"}, true},
		"ssl (unnormalized)":    {[]string{"ssl://a:8529"}, true},
		"unknown scheme":        {[]string{"ftp://a:8529"}, true},
		"typo in scheme":        {[]string{"htps://a:8529"}, true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ep := NewRoundRobinEndpoints(tc.urls)
			err := ValidateEndpointSchemes(ep)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
