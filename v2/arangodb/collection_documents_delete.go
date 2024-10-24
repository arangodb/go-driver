//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

// CollectionDocumentDelete removes document(s) with given key(s) from the collection
// https://docs.arangodb.com/stable/develop/http-api/documents/#remove-a-document
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
	// 'documents' must be a slice of structs with a `_key` field or a slice of keys.
	DeleteDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentDeleteOptions) (CollectionDocumentDeleteResponseReader, error)
}

type CollectionDocumentDeleteResponse struct {
	DocumentMeta          `json:",inline"`
	shared.ResponseStruct `json:",inline"`
	Old                   interface{} `json:"old,omitempty"`
}

type CollectionDocumentDeleteResponseReader interface {
	Read(i interface{}) (CollectionDocumentDeleteResponse, error)
}

type CollectionDocumentDeleteOptions struct {
	// Conditionally delete a document based on a target revision id
	// IMPORTANT: This will work only for single document delete operations (CollectionDocumentDelete.DeleteDocument,
	// CollectionDocumentDelete.DeleteDocumentWithOptions)
	IfMatch string

	// By default, or if this is set to true, the _rev attributes in the given document are ignored.
	// If this is set to false, then the _rev attribute given in the body document is taken as a precondition.
	// The document is only removed if the current revision is the one specified.
	// This works only with multiple documents removal method CollectionDocumentDelete.DeleteDocumentsWithOptions
	IgnoreRevs *bool

	// Wait until the deletion operation has been synced to disk.
	WithWaitForSync *bool

	// Return additionally the complete previous revision of the changed document
	OldObject interface{}

	// If set to true, an empty object is returned as response if the document operation succeeds.
	// No meta-data is returned for the deleted document. If the operation raises an error, an error object is returned.
	// You can use this option to save network traffic.
	Silent *bool

	// RefillIndexCaches if set to true then refills the in-memory index caches.
	RefillIndexCaches *bool

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

func (c *CollectionDocumentDeleteOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, c.IfMatch)
	}

	if c.IgnoreRevs != nil {
		r.AddQuery(QueryIgnoreRevs, boolToString(*c.IgnoreRevs))
	}

	if c.WithWaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*c.WithWaitForSync))
	}

	if c.OldObject != nil {
		r.AddQuery(QueryReturnOld, "true")
	}

	if c.Silent != nil {
		r.AddQuery(QuerySilent, boolToString(*c.Silent))
	}

	if c.RefillIndexCaches != nil {
		r.AddQuery(QueryRefillIndexCaches, boolToString(*c.RefillIndexCaches))
	}

	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}
	return nil
}
