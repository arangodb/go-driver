//go:build toxiproxy

//
// DISCLAIMER
//
// Copyright 2026 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// toxiproxyConnectionFactory builds a driver connection routed through Toxiproxy.
type toxiproxyConnectionFactory func(testing.TB) connection.Connection

// newToxiproxyClient creates a driver client and waits until ArangoDB is reachable through the proxy.
func newToxiproxyClient(t testing.TB, connFactory toxiproxyConnectionFactory) arangodb.Client {
	timeout := 1 * time.Minute
	if isK8S() {
		timeout = 3 * time.Minute
	}
	return waitForConnectionTimeout(t, arangodb.NewClient(connFactory(t)), timeout)
}

// runToxiproxyWithHTTPProtocols runs the given test body for both HTTP/1 and HTTP/2 connections.
func runToxiproxyWithHTTPProtocols(t *testing.T, run func(t *testing.T, connFactory toxiproxyConnectionFactory)) {
	requireToxiproxyAvailable(t)

	probe := newToxiproxyClient(t, connectionJsonHttp)
	version, err := probe.Version(context.Background())
	require.NoError(t, err)

	t.Run("HTTP/1", func(t *testing.T) {
		run(t, connectionJsonHttp)
	})

	t.Run("HTTP/2", func(t *testing.T) {
		if version.Version.CompareTo("3.7.1") < 1 {
			t.Skip("HTTP/2 requires ArangoDB 3.7.1 or newer")
		}
		run(t, connectionJsonHttp2)
	})
}

// isConnectionError reports whether err indicates a transport-level connection failure.
// These are not HTTP status codes (401, 503, etc.) — they come from the TCP/HTTP stack
// when the connection is reset, closed, or otherwise broken before a response is received.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	if pkgerrors.As(err, &opErr) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isConnectionError(urlErr.Unwrap())
	}
	if pkgerrors.As(err, &urlErr) {
		return isConnectionError(urlErr.Unwrap())
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "unexpected eof") || // HTTP/2 often surfaces abrupt close as EOF, not RST
		strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "client connection lost") ||
		strings.Contains(msg, "transport connection broken")
}

// isResetOrEOFError reports the documented outcomes for RST-by-peer simulation.
// HTTP/1 typically surfaces "connection reset by peer"; HTTP/2 often surfaces "unexpected EOF".
func isResetOrEOFError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "unexpected eof") {
		return true
	}

	// Accept wrapped net.OpError from the driver stack as well.
	var opErr *net.OpError
	if errors.As(err, &opErr) || pkgerrors.As(err, &opErr) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isResetOrEOFError(urlErr.Unwrap())
	}
	if pkgerrors.As(err, &urlErr) {
		return isResetOrEOFError(urlErr.Unwrap())
	}

	return false
}

func TestIsConnectionError_detectsUnexpectedEOF(t *testing.T) {
	err := pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: errors.New("unexpected EOF"),
	})
	require.True(t, isConnectionError(err))
}

func TestIsConnectionError_detectsResetByPeer(t *testing.T) {
	err := pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: &net.OpError{
			Op:  "read",
			Err: errors.New("connection reset by peer"),
		},
	})
	require.True(t, isConnectionError(err))
}

func TestIsResetOrEOFError_detectsDocumentedOutcomes(t *testing.T) {
	require.True(t, isResetOrEOFError(pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: errors.New("unexpected EOF"),
	})))
	require.True(t, isResetOrEOFError(pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: &net.OpError{Op: "read", Err: errors.New("connection reset by peer")},
	})))
}
