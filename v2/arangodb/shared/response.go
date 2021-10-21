//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package shared

import (
	"fmt"
)

type Response struct {
	ResponseStruct `json:",inline"`
}

type ResponseStructList []ResponseStruct

func (r ResponseStructList) HasError() bool {
	if len(r) == 0 {
		return false
	}

	for _, resp := range r {
		if err := resp.Error; err != nil || *err {
			return true
		}
	}

	return false
}

func NewResponseStruct() *ResponseStruct {
	return &ResponseStruct{}
}

type ResponseStruct struct {
	Error        *bool   `json:"error,omitempty"`
	Code         *int    `json:"code,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	ErrorNum     *int    `json:"errorNum,omitempty"`
}

func (r ResponseStruct) ExpectCode(codes ...int) error {
	if r.Error == nil || !*r.Error || r.Code == nil {
		return nil
	}

	for _, code := range codes {
		if code == *r.Code {
			return nil
		}
	}

	return r.AsArangoError()
}

func (r *ResponseStruct) AsArangoErrorWithCode(code int) ArangoError {
	if r == nil {
		return (&ResponseStruct{}).AsArangoErrorWithCode(code)
	}
	r.Code = &code
	t := true
	r.Error = &t
	return r.AsArangoError()
}

func (r ResponseStruct) AsArangoError() ArangoError {
	a := ArangoError{}

	if r.Error != nil {
		a.HasError = *r.Error
	}

	if r.Code != nil {
		a.Code = *r.Code
	}

	if r.ErrorNum != nil {
		a.ErrorNum = *r.ErrorNum
	}

	if r.ErrorMessage != nil {
		a.ErrorMessage = *r.ErrorMessage
	}

	return a
}

func (r ResponseStruct) String() string {
	if r.Error == nil || !*r.Error {
		return ""
	}

	s := "Response error"

	if r.Code != nil {
		s = fmt.Sprintf("%s (Code: %d)", s, *r.Code)
	}

	if r.ErrorNum != nil {
		s = fmt.Sprintf("%s (ErrorNum: %d)", s, *r.ErrorNum)
	}

	if r.ErrorMessage != nil {
		s = fmt.Sprintf("%s: %s", s, *r.ErrorMessage)
	}

	return s
}

func (r Response) GetError() bool {
	if r.Error == nil {
		return false
	}

	return *r.Error
}

func (r Response) GetCode() int {
	if r.Code == nil {
		return 0
	}

	return *r.Code
}

func (r Response) GetErrorMessage() string {
	if r.ErrorMessage == nil {
		return ""
	}

	return *r.ErrorMessage
}

func (r Response) GetErrorNum() int {
	if r.ErrorNum == nil {
		return 0
	}

	return *r.ErrorNum
}
