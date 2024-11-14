//
// DISCLAIMER
//
// Copyright 2017-2024 ArangoDB GmbH, Cologne, Germany
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
)

type ContextKey string

const (
	keyAsyncRequest ContextKey = "arangodb-async-request"
	keyAsyncID      ContextKey = "arangodb-async-id"
)

// contextOrBackground returns the given context if it is not nil.
// Returns context.Background() otherwise.
func contextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// WithAsync is used to configure a context to make an async operation - requires Connection with Async wrapper!
func WithAsync(parent context.Context) context.Context {
	return context.WithValue(contextOrBackground(parent), keyAsyncRequest, true)
}

// WithAsyncID is used to check an async operation result - requires Connection with Async wrapper!
func WithAsyncID(parent context.Context, asyncID string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyAsyncID, asyncID)
}

//
// READ METHODS
//

// IsAsyncRequest returns true if the given context is an async request.
func IsAsyncRequest(ctx context.Context) bool {
	if ctx != nil {
		if v := ctx.Value(keyAsyncRequest); v != nil {
			if isAsync, ok := v.(bool); ok && isAsync {
				return true
			}
		}
	}

	return false
}

// HasAsyncID returns the async Job ID from the given context.
func HasAsyncID(ctx context.Context) (string, bool) {
	if ctx != nil {
		if q := ctx.Value(keyAsyncID); q != nil {
			if v, ok := q.(string); ok {
				return v, true
			}
		}
	}

	return "", false
}
