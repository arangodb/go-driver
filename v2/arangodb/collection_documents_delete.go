//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

type CollectionDocumentDelete interface {
	// DeleteDocument removes a single document with given key from the collection.
	// The document metadata is returned.
	// If no document exists with given key, a NotFoundError is returned.
	DeleteDocument(ctx context.Context, key string) (CollectionDocumentDeleteResponse, error)

	// DeleteDocumentWithOptions removes a single document with given key from the collection.
	// The document metadata is returned.
	// If no document exists with given key, a NotFoundError is returned.
	DeleteDocumentWithOptions(ctx context.Context, key string, opts *CollectionDocumentDeleteOptions) (CollectionDocumentDeleteResponse, error)

	// DeleteDocuments removes multiple documents with given keys from the collection.
	// The document metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	DeleteDocuments(ctx context.Context, keys []string) (CollectionDocumentDeleteResponseReader, error)

	// DeleteDocumentsWithOptions removes multiple documents with given keys from the collection.
	// The document metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	DeleteDocumentsWithOptions(ctx context.Context, keys []string, opts *CollectionDocumentDeleteOptions) (CollectionDocumentDeleteResponseReader, error)
}

type CollectionDocumentDeleteResponse struct {
	DocumentMeta          `json:",inline"`
	shared.ResponseStruct `json:",inline"`
	Old                   interface{}
}

type CollectionDocumentDeleteResponseReader interface {
	Read(i interface{}) (CollectionDocumentDeleteResponse, error)
}

type CollectionDocumentDeleteOptions struct {
	// Wait until deletion operation has been synced to disk.
	WithWaitForSync *bool

	// Return additionally the complete previous revision of the changed document
	ReturnOld *bool

	// If set to true, an empty object is returned as response if the document operation succeeds.
	// No meta-data is returned for the deleted document. If the operation raises an error, an error object is returned.
	// You can use this option to save network traffic.
	Silent *bool

	// Whether to delete existing entries from in-memory index caches and refill them
	// if document removals affect the edge index or cache-enabled persistent indexes.
	RefillIndexCaches *bool
}

func (c *CollectionDocumentDeleteOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WithWaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WithWaitForSync))
	}

	if c.ReturnOld != nil {
		r.AddQuery("returnOld", boolToString(*c.ReturnOld))
	}

	if c.Silent != nil {
		r.AddQuery("silent", boolToString(*c.Silent))
	}

	if c.RefillIndexCaches != nil {
		r.AddQuery("refillIndexCaches", boolToString(*c.RefillIndexCaches))
	}

	return nil
}
