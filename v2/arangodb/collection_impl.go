//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package arangodb

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCollection(db *database, name string, modifiers ...connection.RequestModifier) *collection {
	d := &collection{db: db, name: name, modifiers: append(db.modifiers, modifiers...)}

	d.collectionDocuments = newCollectionDocuments(d)

	return d
}

var _ Collection = &collection{}

type collection struct {
	name string

	db *database

	modifiers []connection.RequestModifier

	*collectionDocuments
}

func (c collection) Remove(ctx context.Context) error {
	url := c.url("collection")

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallDelete(ctx, c.connection(), url, &response)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (c collection) Name() string {
	return c.name
}

func (c collection) Database() Database {
	return c.db
}

func (c collection) withModifiers(modifiers ...connection.RequestModifier) []connection.RequestModifier {
	if len(modifiers) == 0 {
		return c.modifiers
	}

	z := len(c.modifiers)

	d := make([]connection.RequestModifier, len(modifiers)+z)

	copy(d, c.modifiers)

	for i, v := range modifiers {
		d[i+z] = v
	}

	return d
}

func (c collection) connection() connection.Connection {
	return c.db.connection()
}

func (c collection) url(api string, parts ...string) string {
	return c.db.url(append([]string{"_api", api, c.name}, parts...)...)
}

// Shards fetches shards information of the collection.
func (c *collection) Shards(ctx context.Context, details bool) (CollectionShards, error) {
	var body struct {
		shared.ResponseStruct `json:",inline"`
		CollectionShards      `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, c.connection(), c.url("collection", "shards"), &body,
		connection.WithQuery("details", "true"))
	if err != nil {
		return CollectionShards{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return body.CollectionShards, nil
	default:
		return CollectionShards{}, body.AsArangoErrorWithCode(code)
	}
}
