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

package http

import (
	"strings"
	"testing"
)

type Sample struct {
	Title string `json:"a"`
	Age   int    `json:"b,omitempty"`
}

func TestSetBodyImportArrayStructs(t *testing.T) {
	r := &httpJSONRequest{}
	docs := []Sample{
		Sample{"Foo", 2},
		Sample{"Dunn", 23},
		Sample{"Short", 0},
		Sample{"Sample", 45},
	}
	expected := strings.Join([]string{
		`{"a":"Foo","b":2}`,
		`{"a":"Dunn","b":23}`,
		`{"a":"Short"}`,
		`{"a":"Sample","b":45}`,
	}, "\n")
	if _, err := r.SetBodyImportArray(docs); err != nil {
		t.Fatalf("SetBodyImportArray failed: %v", err)
	}
	data := strings.TrimSpace(string(r.body))
	if data != expected {
		t.Errorf("Encoding failed: Expected\n%s\nGot\n%s\n", expected, data)
	}
}

func TestSetBodyImportArrayStructPtrs(t *testing.T) {
	r := &httpJSONRequest{}
	docs := []*Sample{
		&Sample{"Foo", 2},
		&Sample{"Dunn", 23},
		&Sample{"Short", 0},
		&Sample{"Sample", 45},
	}
	expected := strings.Join([]string{
		`{"a":"Foo","b":2}`,
		`{"a":"Dunn","b":23}`,
		`{"a":"Short"}`,
		`{"a":"Sample","b":45}`,
	}, "\n")
	if _, err := r.SetBodyImportArray(docs); err != nil {
		t.Fatalf("SetBodyImportArray failed: %v", err)
	}
	data := strings.TrimSpace(string(r.body))
	if data != expected {
		t.Errorf("Encoding failed: Expected\n%s\nGot\n%s\n", expected, data)
	}
}

func TestSetBodyImportArrayStructPtrsNil(t *testing.T) {
	r := &httpJSONRequest{}
	docs := []*Sample{
		&Sample{"Foo", 2},
		nil,
		&Sample{"Dunn", 23},
		&Sample{"Short", 0},
		nil,
		&Sample{"Sample", 45},
	}
	expected := strings.Join([]string{
		`{"a":"Foo","b":2}`,
		``,
		`{"a":"Dunn","b":23}`,
		`{"a":"Short"}`,
		``,
		`{"a":"Sample","b":45}`,
	}, "\n")
	if _, err := r.SetBodyImportArray(docs); err != nil {
		t.Fatalf("SetBodyImportArray failed: %v", err)
	}
	data := strings.TrimSpace(string(r.body))
	if data != expected {
		t.Errorf("Encoding failed: Expected\n%s\nGot\n%s\n", expected, data)
	}
}

func TestSetBodyImportArrayMaps(t *testing.T) {
	r := &httpJSONRequest{}
	docs := []map[string]interface{}{
		map[string]interface{}{"a": 5, "b": "c", "c": true},
		map[string]interface{}{"a": 77, "c": false},
	}
	expected := strings.Join([]string{
		`{"a":5,"b":"c","c":true}`,
		`{"a":77,"c":false}`,
	}, "\n")
	if _, err := r.SetBodyImportArray(docs); err != nil {
		t.Fatalf("SetBodyImportArray failed: %v", err)
	}
	data := strings.TrimSpace(string(r.body))
	if data != expected {
		t.Errorf("Encoding failed: Expected\n%s\nGot\n%s\n", expected, data)
	}
}
