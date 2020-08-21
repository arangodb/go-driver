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

package arangodb

import (
	"context"

	"github.com/arangodb/go-driver/v2/connection"
)

type CollectionDocumentRead interface {
	// ReadDocument reads a single document with given key from the collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error)

	// ReadDocument reads a single document with given key from the collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocumentWithOptions(ctx context.Context, key string, result interface{}, opts *CollectionDocumentReadOptions) (DocumentMeta, error)

	// ReadDocuments reads multiple documents with given keys from the collection.
	// The documents data is stored into elements of the given results slice,
	// the documents meta data is returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	ReadDocuments(ctx context.Context, keys []string) (CollectionDocumentReadResponseReader, error)

	// ReadDocuments reads multiple documents with given keys from the collection.
	// The documents data is stored into elements of the given results slice,
	// the documents meta data is returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	ReadDocumentsWithOptions(ctx context.Context, keys []string, opts *CollectionDocumentReadOptions) (CollectionDocumentReadResponseReader, error)
}

type CollectionDocumentReadResponseReader interface {
	Read(i interface{}) (CollectionDocumentReadResponse, bool, error)
}

type CollectionDocumentReadResponse struct {
	DocumentMeta
}

type CollectionDocumentReadOptions struct {
}

func (c *CollectionDocumentReadOptions) modifyRequest(r connection.Request) error {
	return nil
}
