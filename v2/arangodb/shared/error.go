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

package shared

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	// general errors
	ErrNotImplemented = 9
	ErrForbidden      = 11
	ErrDisabled       = 36

	// HTTP error status codes
	ErrHttpForbidden = 403
	ErrHttpInternal  = 501

	// Internal ArangoDB storage errors
	ErrArangoReadOnly = 1004

	// External ArangoDB storage errors
	ErrArangoCorruptedDatafile    = 1100
	ErrArangoIllegalParameterFile = 1101
	ErrArangoCorruptedCollection  = 1102
	ErrArangoFileSystemFull       = 1104
	ErrArangoDataDirLocked        = 1107

	// General ArangoDB storage errors
	ErrArangoConflict                 = 1200
	ErrArangoDocumentNotFound         = 1202
	ErrArangoDataSourceNotFound       = 1203
	ErrArangoIllegalName              = 1208
	ErrArangoUniqueConstraintViolated = 1210
	ErrArangoDatabaseNotFound         = 1228
	ErrArangoDatabaseNameInvalid      = 1229

	// ArangoDB cluster errors
	ErrClusterReplicationWriteConcernNotFulfilled = 1429
	ErrClusterLeadershipChallengeOngoing          = 1495
	ErrClusterNotLeader                           = 1496

	// User management errors
	ErrUserDuplicate = 1702
)

// ArangoError is a Go error with arangodb specific error information.
type ArangoError struct {
	HasError     bool   `json:"error"`
	Code         int    `json:"code"`
	ErrorNum     int    `json:"errorNum"`
	ErrorMessage string `json:"errorMessage"`
}

// Error returns the error message of an ArangoError.
func (ae ArangoError) Error() string {
	if ae.ErrorMessage != "" {
		return ae.ErrorMessage
	}
	return fmt.Sprintf("ArangoError: Code %d, ErrorNum %d", ae.Code, ae.ErrorNum)
}

// FullError returns the full error message of an ArangoError.
func (ae ArangoError) FullError() string {
	return fmt.Sprintf("ArangoError: Code %d, ErrorNum %d: %s", ae.Code, ae.ErrorNum, ae.ErrorMessage)
}

// Timeout returns true when the given error is a timeout error.
func (ae ArangoError) Timeout() bool {
	return ae.HasError && (ae.Code == http.StatusRequestTimeout || ae.Code == http.StatusGatewayTimeout)
}

// Temporary returns true when the given error is a temporary error.
func (ae ArangoError) Temporary() bool {
	return ae.HasError && ae.Code == http.StatusServiceUnavailable
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

// IsArangoError returns true when the given error is an ArangoError
func IsArangoError(err error) (bool, ArangoError) {
	var arangoErr ArangoError
	ok := errors.As(err, &arangoErr)

	return ok, arangoErr
}

// IsArangoErrorWithCode returns true when the given error is an ArangoError and its Code field is equal to the given code.
func IsArangoErrorWithCode(err error, code int) bool {
	return checkCause(err, func(err error) bool {
		var a ArangoError
		if errors.As(err, &a) {
			return a.HasError && a.Code == code
		}

		return false
	})
}

// IsArangoErrorWithErrorNum returns true when the given error is an ArangoError and its ErrorNum field is equal to one of the given numbers.
func IsArangoErrorWithErrorNum(err error, errorNum ...int) bool {
	return checkCause(err, func(err error) bool {
		var a ArangoError
		if errors.As(err, &a) {
			if !a.HasError {
				return false
			}

			for _, num := range errorNum {
				if num == a.ErrorNum {
					return true
				}
			}
		}

		return false
	})
}

// IsInvalidRequest returns true if the given error is an ArangoError with code 400, indicating an invalid request.
func IsInvalidRequest(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusBadRequest)

}

// IsUnauthorized returns true if the given error is an ArangoError with code 401, indicating an unauthorized request.
func IsUnauthorized(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusUnauthorized)
}

// IsForbidden returns true if the given error is an ArangoError with code 403, indicating a forbidden request.
func IsForbidden(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusForbidden)
}

// IsNotFound returns true if the given error is an ArangoError with code 404, indicating a object not found.
func IsNotFound(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound) ||
		IsArangoErrorWithErrorNum(err, ErrArangoDocumentNotFound, ErrArangoDataSourceNotFound)
}

// IsOperationTimeout returns true if the given error is an ArangoError with code 412, indicating a Operation timeout error
func IsOperationTimeout(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusPreconditionFailed) ||
		IsArangoErrorWithErrorNum(err, ErrArangoConflict)
}

// IsConflict returns true if the given error is an ArangoError with code 409, indicating a conflict.
func IsConflict(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusConflict) || IsArangoErrorWithErrorNum(err, ErrUserDuplicate)
}

// IsPreconditionFailed returns true if the given error is an ArangoError with code 412, indicating a failed precondition.
func IsPreconditionFailed(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusPreconditionFailed) ||
		IsArangoErrorWithErrorNum(err, ErrArangoConflict, ErrArangoUniqueConstraintViolated)
}

// IsNoLeader returns true if the given error is an ArangoError with code 503 error number 1496.
func IsNoLeader(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) && IsArangoErrorWithErrorNum(err, ErrClusterNotLeader)
}

// IsNoLeaderOrOngoing return true if the given error is an ArangoError with code 503 and error number 1496 or 1495
func IsNoLeaderOrOngoing(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) &&
		IsArangoErrorWithErrorNum(err, ErrClusterLeadershipChallengeOngoing, ErrClusterNotLeader)
}

// IsExternalStorageError returns true if ArangoDB is having an error with accessing or writing to storage.
func IsExternalStorageError(err error) bool {
	return IsArangoErrorWithErrorNum(
		err,
		ErrArangoCorruptedDatafile,
		ErrArangoIllegalParameterFile,
		ErrArangoCorruptedCollection,
		ErrArangoFileSystemFull,
		ErrArangoDataDirLocked,
	)
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
	return checkCause(err, func(err error) bool {
		var invalidArgumentError InvalidArgumentError
		if errors.As(err, &invalidArgumentError) {
			return true
		}

		return false
	})
}

// NoMoreDocumentsError is returned by Cursor's, when an attempt is made to read documents when there are no more.
type NoMoreDocumentsError struct{}

// Error implements the error interface for NoMoreDocumentsError.
func (e NoMoreDocumentsError) Error() string {
	return "no more documents"
}

func IsEOF(err error) bool {
	return checkCause(err, func(err error) bool {
		return err == io.EOF
	}) || IsNoMoreDocuments(err)
}

// IsNoMoreDocuments returns true if the given error is an NoMoreDocumentsError.
func IsNoMoreDocuments(err error) bool {
	return checkCause(err, func(err error) bool {
		var noMoreDocumentsError NoMoreDocumentsError
		if errors.As(err, &noMoreDocumentsError) {
			return true
		}

		return false
	})
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
	return checkCause(err, func(err error) bool {
		if _, ok := err.(*ResponseError); ok {
			return true
		}

		return false
	})
}

// IsCanceled returns true if the given error is the result on a cancelled context.
func IsCanceled(err error) bool {
	return checkCause(err, func(err error) bool {
		return err == context.Canceled
	})
}

// IsTimeout returns true if the given error is the result on a deadline that has been exceeded.
func IsTimeout(err error) bool {
	return checkCause(err, func(err error) bool {
		return err == context.DeadlineExceeded
	})
}

type causer interface {
	Cause() error
}

func checkCause(err error, f func(err error) bool) bool {
	if err == nil {
		return false
	}

	if f(err) {
		return true
	}

	if c, ok := err.(causer); ok {
		cErr := c.Cause()

		if err == cErr {
			return false
		}

		return checkCause(cErr, f)
	}

	return false
}

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
