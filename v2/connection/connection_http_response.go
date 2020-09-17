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

package connection

import (
	"net/http"
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

func (j httpResponse) Code() int {
	return j.response.StatusCode
}

func (j httpResponse) Content() string {
	return j.response.Header.Get(ContentType)
}
