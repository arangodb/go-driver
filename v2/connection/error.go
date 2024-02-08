//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"errors"
	"fmt"
	"net/http"
)

func NewErrorf(code int, message string, args ...interface{}) error {
	return NewError(code, fmt.Sprintf(message, args...))
}

func NewError(code int, message string) error {
	return Error{
		Code:    code,
		Message: message,
	}
}

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("Code %d, Error: %s", e.Code, e.Message)
}

type cause interface {
	Cause() error
}

func IsCodeError(err error, code int) bool {
	var codeErr Error
	if errors.As(err, &codeErr) {
		return codeErr.Code == code
	}

	if c, ok := err.(cause); ok {
		return IsCodeError(c.Cause(), code)
	}

	return false
}

func NewNotFoundError(msg string) error {
	return NewError(http.StatusNotFound, msg)
}

func IsNotFoundError(err error) bool {
	return IsCodeError(err, http.StatusNotFound)
}
