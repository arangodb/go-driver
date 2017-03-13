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

package test

import (
	"encoding/json"
	"fmt"
	"testing"

	driver "github.com/arangodb/go-driver"
)

type testEnv interface {
	Error(message ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(message ...interface{})
	Fatalf(format string, args ...interface{})
}

// boolRef returns a reference to a given boolean
func boolRef(v bool) *bool {
	return &v
}

// assertOK fails the test if the given error is not nil.
func assertOK(err error, t *testing.T) {
	if err != nil {
		t.Fatalf("Assertion failed: %s", describe(err))
	}
}

// describe returns a string description of the given error.
func describe(err error) string {
	if err == nil {
		return "nil"
	}
	cause := driver.Cause(err)
	c, _ := json.Marshal(cause)
	if cause.Error() != err.Error() {
		return fmt.Sprintf("%v caused by %v", err, string(c))
	} else {
		return fmt.Sprintf("%v", string(c))
	}
}
