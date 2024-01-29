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

package arangodb

import (
	"context"
)

type DatabaseCollection interface {
	// Collection opens a connection to an existing collection within the database.
	// If no collection with given name exists, an NotFoundError is returned.
	// deprecated: use GetCollection instead
	Collection(ctx context.Context, name string) (Collection, error)

	// GetCollection opens a connection to an existing collection within the database.
	// If no collection with given name exists, an NotFoundError is returned.
	GetCollection(ctx context.Context, name string, options *GetCollectionOptions) (Collection, error)

	// CollectionExists returns true if a collection with given name exists within the database.
	CollectionExists(ctx context.Context, name string) (bool, error)

	// Collections returns a list of all collections in the database.
	Collections(ctx context.Context) ([]Collection, error)

	// CreateCollection creates a new collection with given name and options, and opens a connection to it.
	// If a collection with given name already exists within the database, a DuplicateError is returned.
	CreateCollection(ctx context.Context, name string, props *CreateCollectionProperties) (Collection, error)

	// CreateCollectionWithOptions creates a new collection with given name and options, and opens a connection to it.
	// If a collection with given name already exists within the database, a DuplicateError is returned.
	CreateCollectionWithOptions(ctx context.Context, name string, props *CreateCollectionProperties, options *CreateCollectionOptions) (Collection, error)
}

type GetCollectionOptions struct {
	// SkipExistCheck skips checking if collection exists
	SkipExistCheck bool `json:"skipExistCheck,omitempty"`
}
