//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"net/http"
	"strings"
)

type httpResponse struct {
	response *http.Response
	request  *httpRequest
}

func (j *httpResponse) Endpoint() string {
	return j.request.Endpoint()
}

func (j *httpResponse) Response() interface{} {
	return j.response
}

func (j *httpResponse) Code() int {
	return j.response.StatusCode
}

func (j *httpResponse) Content() string {
	value := strings.Split(j.response.Header.Get(ContentType), ";")
	if len(value) > 0 {
		// The header can be returned with arguments, e.g.: "Content-Type: text/html; charset=UTF-8".
		return value[0]
	}

	return ""
}

func (j *httpResponse) Header(name string) string {
	if j.response.Header == nil {
		return ""
	}
	return j.response.Header.Get(name)
}

func (j *httpResponse) RawResponse() *http.Response {
	return j.response
}
