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

type ResponseStruct struct {
	Error        *bool   `json:"error"`
	Code         *int    `json:"code"`
	ErrorMessage *string `json:"errorMessage"`
	ErrorNum     *int    `json:"errorNum"`
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
