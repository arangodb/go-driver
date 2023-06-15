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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

type CollectionDocumentUpdate interface {

	// UpdateDocument updates a single document with given key in the collection.
	// The document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	UpdateDocument(ctx context.Context, key string, document interface{}) (CollectionDocumentUpdateResponse, error)

	// UpdateDocumentWithOptions updates a single document with given key in the collection.
	// The document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	UpdateDocumentWithOptions(ctx context.Context, key string, document interface{}, options *CollectionDocumentUpdateOptions) (CollectionDocumentUpdateResponse, error)

	// UpdateDocuments updates multiple document with given keys in the collection.
	// The updates are loaded from the given updates slice, the documents meta data are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// If keys is nil, each element in the updates slice must contain a `_key` field.
	UpdateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentUpdateResponseReader, error)

	// UpdateDocumentsWithOptions updates multiple document with given keys in the collection.
	// The updates are loaded from the given updates slice, the documents meta data are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// If keys is nil, each element in the updates slice must contain a `_key` field.
	UpdateDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentUpdateOptions) (CollectionDocumentUpdateResponseReader, error)
}

type CollectionDocumentUpdateResponseReader interface {
	Read() (CollectionDocumentUpdateResponse, error)
}

type CollectionDocumentUpdateResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	Old, New              interface{}
}

type CollectionDocumentUpdateOptions struct {
	WithWaitForSync *bool
	NewObject       interface{}
	OldObject       interface{}
	// RefillIndexCaches if set to true then refills the in-memory index caches.
	RefillIndexCaches *bool
}

func (c *CollectionDocumentUpdateOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WithWaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WithWaitForSync))
	}

	if c.NewObject != nil {
		r.AddQuery("returnNew", "true")
	}

	if c.OldObject != nil {
		r.AddQuery("returnOld", "true")
	}

	if c.RefillIndexCaches != nil {
		r.AddQuery("refillIndexCaches", boolToString(*c.RefillIndexCaches))
	}

	return nil
}
