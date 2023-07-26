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

package tests

import "github.com/google/uuid"

type DocWithRev struct {
	Rev       string         `json:"_rev,omitempty"`
	Key       string         `json:"_key,omitempty"`
	Name      string         `json:"name"`
	Age       *int           `json:"age"`
	Countries map[string]int `json:"countries"`
}

type DocIDGetter interface {
	GetKey() string
}

type basicDocuments []basicDocument

func (b basicDocuments) getKeys() []string {
	l := make([]string, len(b))

	for i, g := range b {
		l[i] = g.GetKey()
	}

	return l
}

type basicDocument struct {
	Key string `json:"_key"`
}

func newBasicDocument() basicDocument {
	return basicDocument{
		Key: uuid.New().String(),
	}
}

func (d basicDocument) GetKey() string {
	return d.Key
}

type documents []document

func (d documents) asBasic() basicDocuments {
	z := make([]basicDocument, len(d))

	for i, q := range d {
		z[i] = q.basicDocument
	}

	return z
}

type document struct {
	basicDocument `json:",inline"`
	Fields        interface{} `json:"data,omitempty"`
}

func newDocs(c int) documents {
	r := make([]document, c)

	for i := 0; i < c; i++ {
		r[i].basicDocument = newBasicDocument()
	}

	return r
}
