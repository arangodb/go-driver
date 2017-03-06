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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestVersion tests Version functions.
func TestVersion(t *testing.T) {

	tests := []struct {
		Input    driver.Version
		Major    int
		Minor    int
		Sub      string
		SubInt   int
		SubIsInt bool
	}{
		{"1.2.3", 1, 2, "3", 3, true},
		{"", 0, 0, "", 0, false},
		{"1.2.3a", 1, 2, "3a", 0, false},
		{"13.12", 13, 12, "", 0, false},
	}

	for _, test := range tests {
		if v := test.Input.Major(); v != test.Major {
			t.Errorf("Major failed for '%s', expected %d, got %d", test.Input, test.Major, v)
		}
		if v := test.Input.Minor(); v != test.Minor {
			t.Errorf("Minor failed for '%s', expected %d, got %d", test.Input, test.Minor, v)
		}
		if v := test.Input.Sub(); v != test.Sub {
			t.Errorf("Sub failed for '%s', expected '%s', got '%s'", test.Input, test.Sub, v)
		}
		if v, vIsInt := test.Input.SubInt(); vIsInt != test.SubIsInt || v != test.SubInt {
			t.Errorf("SubInt failed for '%s', expected (%d,%v), got (%d,%v)", test.Input, test.SubInt, test.SubIsInt, v, vIsInt)
		}
	}
}

// TestVersionCompareTo tests Version.CompareTo.
func TestVersionCompareTo(t *testing.T) {
	tests := []struct {
		A      driver.Version
		B      driver.Version
		Result int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2", "1.2.3", -1},
		{"1.2.3", "2.3.5", -1},
		{"1.2", "1.1", 1},
		{"1.2.3", "1.1.7", 1},
		{"2.2", "1.2.a", 1},
		{"1", "1.2.3", -1},
		{"1.2.a", "1.2.3", 1},
		{"1.2.3", "1.2.a", -1},
		{"", "", 0},
		{"1", "1", 0},
		{"2.1", "2.1", 0},
	}

	for _, test := range tests {
		if r := test.A.CompareTo(test.B); r != test.Result {
			t.Errorf("CompareTo('%s', '%s') failed, expected %d, got %d", test.A, test.B, test.Result, r)
		}
	}
}
