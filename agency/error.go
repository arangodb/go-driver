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
	"fmt"
	"net/http"
	"strings"

	driver "github.com/arangodb/go-driver"
)

var (
	// preconditionFailedError indicates that a precondition for the request is not existing.
	preconditionFailedError = driver.ArangoError{
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
