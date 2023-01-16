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

type UserDoc struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type UserDocWithKey struct {
	Key  string `json:"_key"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type UserDocWithKeyWithOmit struct {
	Key  string `json:"_key,omitempty"`
	Name string `json:"name,omitempty"`
	Age  int    `json:"age,omitempty"`
}

type NestedFieldsDoc struct {
	Name       string      `json:"name"`
	Dimensions []Dimension `json:"dimensions,omitempty"`
}

type Dimension struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type Account struct {
	ID   string   `json:"id"`
	User *UserDoc `json:"user"`
}

type Book struct {
	Title string
}

type BookWithAuthor struct {
	Title  string
	Author string
}

type RouteEdge struct {
	From     string `json:"_from,omitempty"`
	To       string `json:"_to,omitempty"`
	Distance int    `json:"distance,omitempty"`
}

type RouteEdgeWithKey struct {
	Key      string `json:"_key"`
	From     string `json:"_from,omitempty"`
	To       string `json:"_to,omitempty"`
	Distance int    `json:"distance,omitempty"`
}

type RelationEdge struct {
	From string `json:"_from,omitempty"`
	To   string `json:"_to,omitempty"`
	Type string `json:"type,omitempty"`
}

type AccountEdge struct {
	From string   `json:"_from,omitempty"`
	To   string   `json:"_to,omitempty"`
	User *UserDoc `json:"user"`
}
