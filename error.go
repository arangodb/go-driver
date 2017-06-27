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

import (
	"context"
	"net"
	"net/url"
	"os"
)

// ArangoError is a Go error with arangodb specific error information.
type ArangoError struct {
	HasError     bool   `arangodb:"error"`
	Code         int    `arangodb:"code"`
	ErrorNum     int    `arangodb:"errorNum"`
	ErrorMessage string `arangodb:"errorMessage"`
}

// Error returns the error message of an ArangoError.
func (ae ArangoError) Error() string {
	return ae.ErrorMessage
}

// newArangoError creates a new ArangoError with given values.
func newArangoError(code, errorNum int, errorMessage string) error {
	return ArangoError{
		HasError:     true,
		Code:         code,
		ErrorNum:     errorNum,
		ErrorMessage: errorMessage,
	}
}

// IsArangoError returns true when the given error is an ArangoError.
func IsArangoError(err error) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.HasError
}

// IsArangoErrorWithCode returns true when the given error is an ArangoError and its Code field is equal to the given code.
func IsArangoErrorWithCode(err error, code int) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.Code == code
}

// IsArangoErrorWithErrorNum returns true when the given error is an ArangoError and its ErrorNum field is equal to one of the given numbers.
func IsArangoErrorWithErrorNum(err error, errorNum ...int) bool {
	ae, ok := Cause(err).(ArangoError)
	if !ok {
		return false
	}
	for _, x := range errorNum {
		if ae.ErrorNum == x {
			return true
		}
	}
	return false
}

// IsInvalidRequest returns true if the given error is an ArangoError with code 400, indicating an invalid request.
func IsInvalidRequest(err error) bool {
	return IsArangoErrorWithCode(err, 400)
}

// IsUnauthorized returns true if the given error is an ArangoError with code 401, indicating an unauthorized request.
func IsUnauthorized(err error) bool {
	return IsArangoErrorWithCode(err, 401)
}

// IsForbidden returns true if the given error is an ArangoError with code 403, indicating a forbidden request.
func IsForbidden(err error) bool {
	return IsArangoErrorWithCode(err, 403)
}

// IsNotFound returns true if the given error is an ArangoError with code 404, indicating a object not found.
func IsNotFound(err error) bool {
	return IsArangoErrorWithCode(err, 404) || IsArangoErrorWithErrorNum(err, 1202, 1203)
}

// IsConflict returns true if the given error is an ArangoError with code 409, indicating a conflict.
func IsConflict(err error) bool {
	return IsArangoErrorWithCode(err, 409) || IsArangoErrorWithErrorNum(err, 1702)
}

// IsPreconditionFailed returns true if the given error is an ArangoError with code 412, indicating a failed precondition.
func IsPreconditionFailed(err error) bool {
	return IsArangoErrorWithCode(err, 412) || IsArangoErrorWithErrorNum(err, 1200, 1210)
}

// InvalidArgumentError is returned when a go function argument is invalid.
type InvalidArgumentError struct {
	Message string
}

// Error implements the error interface for InvalidArgumentError.
func (e InvalidArgumentError) Error() string {
	return e.Message
}

// IsInvalidArgument returns true if the given error is an InvalidArgumentError.
func IsInvalidArgument(err error) bool {
	_, ok := Cause(err).(InvalidArgumentError)
	return ok
}

// NoMoreDocumentsError is returned by Cursor's, when an attempt is made to read documents when there are no more.
type NoMoreDocumentsError struct{}

// Error implements the error interface for NoMoreDocumentsError.
func (e NoMoreDocumentsError) Error() string {
	return "no more documents"
}

// IsNoMoreDocuments returns true if the given error is an NoMoreDocumentsError.
func IsNoMoreDocuments(err error) bool {
	_, ok := Cause(err).(NoMoreDocumentsError)
	return ok
}

// A ResponseError is returned when a request was completely written to a server, but
// the server did not respond, or some kind of network error occurred during the response.
type ResponseError struct {
	Err error
}

// Error returns the Error() result of the underlying error.
func (e *ResponseError) Error() string {
	return e.Err.Error()
}

// IsResponse returns true if the given error is (or is caused by) a ResponseError.
func IsResponse(err error) bool {
	return isCausedBy(err, func(e error) bool { _, ok := e.(*ResponseError); return ok })
}

// IsCanceled returns true if the given error is the result on a cancelled context.
func IsCanceled(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.Canceled })
}

// IsTimeout returns true if the given error is the result on a deadline that has been exceeded.
func IsTimeout(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.DeadlineExceeded })
}

// isCausedBy returns true if the given error returns true on the given predicate,
// unwrapping various standard library error wrappers.
func isCausedBy(err error, p func(error) bool) bool {
	if p(err) {
		return true
	}
	err = Cause(err)
	for {
		if p(err) {
			return true
		} else if err == nil {
			return false
		}
		if xerr, ok := err.(*ResponseError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*url.Error); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*net.OpError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*os.SyscallError); ok {
			err = xerr.Err
		} else {
			return false
		}
	}
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

// ErrorSlice is a slice of errors
type ErrorSlice []error

// FirstNonNil returns the first error in the slice that is not nil.
// If all errors in the slice are nil, nil is returned.
func (l ErrorSlice) FirstNonNil() error {
	for _, e := range l {
		if e != nil {
			return e
		}
	}
	return nil
}
