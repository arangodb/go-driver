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
	Name string `arangodb:"name" json:"name"`
	Age  int    `arangodb:"age" json:"age"`
}

type UserDocWithKey struct {
	Key  string `arangodb:"_key" json:"_key"`
	Name string `arangodb:"name" json:"name"`
	Age  int    `arangodb:"age" json:"age"`
}

type Account struct {
	ID   string   `arangodb:"id" json:"id"`
	User *UserDoc `arangodb:"user" json:"user"`
}

type Book struct {
	Title string
}

type RouteEdge struct {
	From     string `arangodb:"_from,omitempty" json:"_from,omitempty"`
	To       string `arangodb:"_to,omitempty" json:"_to,omitempty"`
	Distance int    `arangodb:"distance,omitempty" json:"distance,omitempty"`
}

type RouteEdgeWithKey struct {
	Key      string `arangodb:"_key" json:"_key"`
	From     string `arangodb:"_from,omitempty" json:"_from,omitempty"`
	To       string `arangodb:"_to,omitempty" json:"_to,omitempty"`
	Distance int    `arangodb:"distance,omitempty" json:"distance,omitempty"`
}

type RelationEdge struct {
	From string `arangodb:"_from,omitempty" json:"_from,omitempty"`
	To   string `arangodb:"_to,omitempty" json:"_to,omitempty"`
	Type string `arangodb:"type,omitempty" json:"type,omitempty"`
}

type AccountEdge struct {
	From string   `arangodb:"_from,omitempty" json:"_from,omitempty"`
	To   string   `arangodb:"_to,omitempty" json:"_to,omitempty"`
	User *UserDoc `arangodb:"user" json:"user"`
}
