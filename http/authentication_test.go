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
// Author Tomasz Mielech <tomasz@arangodb.com>
//

package http

import (
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/assert"
)

func TestIsAuthenticationTheSame(t *testing.T) {
	testCases := map[string]struct {
		auth1    driver.Authentication
		auth2    driver.Authentication
		expected bool
	}{
		"Two authentications are nil": {
			expected: true,
		},
		"One authentication is nil": {
			auth1: driver.BasicAuthentication("", ""),
		},
		"Different type of authentication": {
			auth1: driver.BasicAuthentication("", ""),
			auth2: driver.JWTAuthentication("", ""),
		},
		"Raw authentications are different": {
			auth1: driver.RawAuthentication(""),
			auth2: driver.RawAuthentication("test"),
		},
		"Raw authentications are the same": {
			auth1:    driver.RawAuthentication("test"),
			auth2:    driver.RawAuthentication("test"),
			expected: true,
		},
		"Basic authentications are different": {
			auth1: driver.BasicAuthentication("test", "test"),
			auth2: driver.BasicAuthentication("test", "test1"),
		},
		"Basic authentications are the same": {
			auth1:    driver.BasicAuthentication("test", "test"),
			auth2:    driver.BasicAuthentication("test", "test"),
			expected: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			equal := IsAuthenticationTheSame(testCase.auth1, testCase.auth2)
			assert.Equal(t, testCase.expected, equal)
		})
	}

}
