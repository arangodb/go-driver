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

package tests

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

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

func DocumentExists(t testing.TB, col arangodb.Collection, doc DocIDGetter) {
	withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
		z := reflect.New(reflect.TypeOf(doc))

		_, err := col.ReadDocument(ctx, doc.GetKey(), z.Interface())
		require.NoError(t, err)
		require.Equal(t, doc, z.Elem().Interface())
	})
}

func DocumentNotExists(t testing.TB, col arangodb.Collection, doc DocIDGetter) {
	withContextT(t, 30*time.Second, func(ctx context.Context, t testing.TB) {
		z := reflect.New(reflect.TypeOf(doc))

		_, err := col.ReadDocument(ctx, doc.GetKey(), z.Interface())
		require.Error(t, err)
		require.True(t, shared.IsNotFound(err))
	})
}
