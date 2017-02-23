//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

package driver

// ArangoError is a Go error with arangodb specific error information.
type ArangoError struct {
	HasError     bool   `json:"error"`
	Code         int    `json:"code"`
	ErrorNum     int    `json:"errorNum"`
	ErrorMessage string `json:"errorMessage"`
}

// Error returns the error message of an ArangoError.
func (ae ArangoError) Error() string {
	return ae.ErrorMessage
}

// IsArangoErrorWithCode returns true when the given error is an ArangoError and its Code field is equal to the given code.
func IsArangoErrorWithCode(err error, code int) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.Code == code
}

// IsInvalidRequest returns true if the given error is an ArangoError with code 400, indicating an invalid request.
func IsInvalidRequest(err error) bool {
	return IsArangoErrorWithCode(err, 400)
}

// IsUnauthorized returns true if the given error is an ArangoError with code 401, indicating an unauthorized request.
func IsUnauthorized(err error) bool {
	return IsArangoErrorWithCode(err, 401)
}

// IsNotFound returns true if the given error is an ArangoError with code 404, indicating a object not found.
func IsNotFound(err error) bool {
	return IsArangoErrorWithCode(err, 404)
}

// IsConflict returns true if the given error is an ArangoError with code 409, indicating a conflict.
func IsConflict(err error) bool {
	return IsArangoErrorWithCode(err, 409)
}

// InvalidArgumentError is returned when a go function argument is invalid.
type InvalidArgumentError struct {
	Message string
}

// Error implements the error interface for InvalidArgumentError.
func (e InvalidArgumentError) Error() string {
	return e.Message
}

// IsInvalidArgument returns true if the given error in an InvalidArgumentError.
func IsInvalidArgument(err error) bool {
	_, ok := Cause(err).(InvalidArgumentError)
	return ok
}

var (
	// WithStack is called on every return of an error to add stacktrace information to the error.
	// When setting this function, also set the Cause function.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	WithStack = func(err error) error { return err }
	// Cause is used to get the root cause of the given error.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	Cause = func(err error) error { return err }
)
