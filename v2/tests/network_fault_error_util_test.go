//go:build resiliency || toxiproxy

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
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

// isEOFError reports abrupt stream close surfaced as io.EOF or a plain ": EOF" message.
// HTTP/1 often returns "Post \"…\": EOF"; HTTP/2 more often uses "unexpected EOF".
func isEOFError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || pkgerrors.Is(err, io.EOF) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isEOFError(urlErr.Unwrap())
	}
	if pkgerrors.As(err, &urlErr) {
		return isEOFError(urlErr.Unwrap())
	}

	msg := strings.ToLower(err.Error())
	return strings.HasSuffix(msg, ": eof") || strings.Contains(msg, "unexpected eof")
}

// isConnectionError reports whether err indicates a transport-level connection failure.
// These are not HTTP status codes (401, 503, etc.) — they come from the TCP/HTTP stack
// when the connection is reset, closed, or otherwise broken before a response is received.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if isEOFError(err) {
		return true
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
	if isEOFError(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection reset") {
		return true
	}

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

// isContextDeadlineExceeded reports whether err is a context timeout from the caller.
func isContextDeadlineExceeded(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if pkgerrors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded")
}

// isDriverTimeoutError reports transport- or context-level timeouts (not connection loss).
func isDriverTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if isContextDeadlineExceeded(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "i/o timeout")
}

// isIntermittentNetworkError reports transport failures expected under partial packet loss.
func isIntermittentNetworkError(err error) bool {
	return isConnectionError(err) || isResetOrEOFError(err) || isDriverTimeoutError(err)
}

// isArangoGatewayOrCursorError reports HTTP status codes returned when ingress or coordinators
// are unavailable or the cursor no longer exists on the serving coordinator.
func isArangoGatewayOrCursorError(err error) bool {
	var ae shared.ArangoError
	if errors.As(err, &ae) || pkgerrors.As(err, &ae) {
		switch ae.Code {
		case http.StatusGone, http.StatusNotFound, http.StatusConflict,
			http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}

// isNonJSONProxyResponseError reports JSON decode failures when ingress/nginx returns HTML
// error pages (e.g. 502/503) instead of a JSON ArangoDB API response.
func isNonJSONProxyResponseError(err error) bool {
	if err == nil {
		return false
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || pkgerrors.As(err, &syntaxErr) {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "looking for beginning of value") ||
		strings.Contains(msg, "invalid character '<'")
}

// isCoordinatorKillInterruptedError reports acceptable failures when a cursor is interrupted by coordinator kill.
func isCoordinatorKillInterruptedError(err error) bool {
	return isStreamingInterruptError(err) ||
		isArangoGatewayOrCursorError(err) ||
		isNonJSONProxyResponseError(err)
}

// isStreamingInterruptError reports clean failures when a query or cursor is interrupted.
func isStreamingInterruptError(err error) bool {
	return isWriteOrStreamInterruptError(err)
}

// isWriteInterruptError reports clean failures when an insert or transaction commit
// is interrupted mid-flight. The driver cannot know whether the server committed.
func isWriteInterruptError(err error) bool {
	return isWriteOrStreamInterruptError(err)
}

// isWriteOrStreamInterruptError reports clean transport failures for mid-flight cuts.
func isWriteOrStreamInterruptError(err error) bool {
	if err == nil {
		return false
	}
	if isIntermittentNetworkError(err) {
		return true
	}
	if errors.Is(err, context.Canceled) || pkgerrors.Is(err, context.Canceled) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection") ||
		isEOFError(err) ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "context canceled")
}

// isResiliencyTransientError reports acceptable Version/insert failures during ingress or coordinator chaos.
func isResiliencyTransientError(err error) bool {
	if err == nil {
		return false
	}
	if isIntermittentNetworkError(err) {
		return true
	}
	if isNonJSONProxyResponseError(err) {
		return true
	}
	return isArangoGatewayOrCursorError(err)
}

// assertResiliencyDuringErrors verifies failures observed during the fault window are transient.
func assertResiliencyDuringErrors(t testing.TB, duringErrors []error) {
	t.Helper()
	if len(duringErrors) == 0 {
		return
	}

	seen := make(map[string]struct{})
	for _, err := range duringErrors {
		require.Error(t, err)
		msg := err.Error()
		if _, ok := seen[msg]; !ok {
			seen[msg] = struct{}{}
			t.Logf("during-fault error: %s (transient=%t)", msg, isResiliencyTransientError(err))
		}
		require.True(t, isResiliencyTransientError(err),
			"unexpected error during fault window: %v", err)
	}
}

func TestIsConnectionError_detectsUnexpectedEOF(t *testing.T) {
	err := pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: errors.New("unexpected EOF"),
	})
	require.True(t, isConnectionError(err))
}

func TestIsConnectionError_detectsPlainEOF(t *testing.T) {
	err := pkgerrors.WithStack(&url.Error{
		Op:  "Post",
		URL: "http://127.0.0.1:17001/_api/cursor/1/2",
		Err: io.EOF,
	})
	require.True(t, isConnectionError(err))
	require.True(t, isStreamingInterruptError(err))
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

func TestIsContextDeadlineExceeded_detectsTimeout(t *testing.T) {
	require.True(t, isContextDeadlineExceeded(context.DeadlineExceeded))
	require.True(t, isContextDeadlineExceeded(pkgerrors.WithStack(context.DeadlineExceeded)))
	require.True(t, isContextDeadlineExceeded(pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: context.DeadlineExceeded,
	})))
}

func TestIsDriverTimeoutError_detectsResponseHeaderTimeout(t *testing.T) {
	require.True(t, isDriverTimeoutError(pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://127.0.0.1:17001/_api/version",
		Err: errors.New("net/http: timeout awaiting response headers"),
	})))
	require.True(t, isDriverTimeoutError(context.DeadlineExceeded))
}

// isDeadCursorError reports errors when reading from a cursor after its server connection died.
func isDeadCursorError(err error) bool {
	return isCoordinatorKillInterruptedError(err)
}

func TestIsResiliencyTransientError_acceptsGatewayCodes(t *testing.T) {
	require.True(t, isResiliencyTransientError(shared.ArangoError{HasError: true, Code: http.StatusServiceUnavailable}))
	require.True(t, isResiliencyTransientError(shared.ArangoError{HasError: true, Code: http.StatusGone}))
	require.True(t, isResiliencyTransientError(errors.New("invalid character '<' looking for beginning of value")))
	require.True(t, isResiliencyTransientError(pkgerrors.WithStack(&url.Error{
		Op:  "Get",
		URL: "http://arangodb.local/_api/version",
		Err: context.DeadlineExceeded,
	})))
}

func TestIsCoordinatorKillInterruptedError_acceptsCursorGone(t *testing.T) {
	require.True(t, isCoordinatorKillInterruptedError(shared.ArangoError{HasError: true, Code: http.StatusGone}))
	require.True(t, isCoordinatorKillInterruptedError(shared.ArangoError{HasError: true, Code: http.StatusBadGateway}))
	require.True(t, isDeadCursorError(shared.ArangoError{HasError: true, Code: http.StatusConflict}))
	require.True(t, isCoordinatorKillInterruptedError(pkgerrors.WithStack(&url.Error{
		Op:  "Post",
		URL: "http://127.0.0.1:17001/_api/cursor/1/2",
		Err: io.EOF,
	})))
	require.True(t, isCoordinatorKillInterruptedError(&json.SyntaxError{Offset: 0}))
	require.True(t, isCoordinatorKillInterruptedError(errors.New("invalid character '<' looking for beginning of value")))
}
