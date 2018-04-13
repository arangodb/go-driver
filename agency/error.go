//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package agency

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	driver "github.com/arangodb/go-driver"
)

var (
	ConditionFailedError = errors.New("Condition failed")

	// NotFoundError indicates that right now the object is not found.
	NotFoundError = driver.ArangoError{
		HasError: true,
		Code:     http.StatusNotFound,
	}
	// PreconditionFailedError indicates that a precondition for the request is not existing.
	PreconditionFailedError = driver.ArangoError{
		HasError: true,
		Code:     http.StatusPreconditionFailed,
	}
)

// KeyNotFoundError indicates that a key was not found.
type KeyNotFoundError struct {
	Key []string
}

// Error returns a human readable error string
func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("Key '%s' not found", strings.Join(e.Key, "/"))
}

// IsKeyNotFound returns true if the given error is (or is caused by) a KeyNotFoundError.
func IsKeyNotFound(err error) bool {
	_, ok := driver.Cause(err).(KeyNotFoundError)
	return ok
}

// IsConditionFailed returns true if the given error is (or is caused by) a ConditionFailedError.
func IsConditionFailed(err error) bool {
	return driver.Cause(err) == ConditionFailedError
}

// IsServiceUnavailable returns true if the given error is caused by a ServiceUnavailableError.
func IsServiceUnavailable(err error) bool {
	return driver.IsArangoErrorWithCode(err, http.StatusServiceUnavailable)
}

// IsInternalServer returns true if the given error is caused by a InternalServerError.
func IsInternalServer(err error) bool {
	return driver.IsArangoErrorWithCode(err, http.StatusInternalServerError)
}

// IsRequestTimeout returns true if the given error is caused by a request timeout (408).
func IsRequestTimeout(err error) bool {
	return driver.IsArangoErrorWithCode(err, http.StatusRequestTimeout)
}
