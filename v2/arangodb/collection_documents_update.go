//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

// CollectionDocumentUpdate Partially updates document(s) with given key in the collection.
// https://docs.arangodb.com/stable/develop/http-api/documents/#update-a-document
type CollectionDocumentUpdate interface {
	// UpdateDocument updates a single document with a given key in the collection.
	// The document metadata is returned.
	// If no document exists with a given key, a NotFoundError is returned.
	// If `_id` field is present in the document body, it is always ignored.
	UpdateDocument(ctx context.Context, key string, document interface{}) (CollectionDocumentUpdateResponse, error)

	// UpdateDocumentWithOptions updates a single document with a given key in the collection.
	// The document metadata is returned.
	// If no document exists with a given key, a NotFoundError is returned.
	// If `_id` field is present in the document body, it is always ignored.
	UpdateDocumentWithOptions(ctx context.Context, key string, document interface{}, options *CollectionDocumentUpdateOptions) (CollectionDocumentUpdateResponse, error)

	// UpdateDocuments updates multiple documents
	// The updates are loaded from the given updates slice, the documents metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// Each element in the update slice must contain a `_key` field.
	// If `_id` field is present in the document body, it is always ignored.
	UpdateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentUpdateResponseReader, error)

	// UpdateDocumentsWithOptions updates multiple documents
	// The updates are loaded from the given updates slice, the documents metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// Each element in the update slice must contain a `_key` field.
	// If `_id` field is present in the document body, it is always ignored.
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
	// Conditionally update a document based on a target revision id
	// IMPORTANT: This will work only for single document updates operations (CollectionDocumentUpdate.UpdateDocument,
	// CollectionDocumentUpdate.UpdateDocumentWithOptions)
	IfMatch string

	// By default, or if this is set to true, the _rev attributes in the given document is ignored.
	// If this is set to false, then the _rev attribute given in the body document is taken as a precondition.
	// The document is only updated if the current revision is the one specified.
	IgnoreRevs *bool

	// Wait until document has been synced to disk.
	WithWaitForSync *bool

	// If set to true, an empty object is returned as response if the document operation succeeds.
	// No meta-data is returned for the created document. If the operation raises an error, an error object is returned.
	// You can use this option to save network traffic.
	Silent *bool

	// Additionally return the complete new document
	NewObject interface{}

	// Additionally return the complete old document under the attribute.
	// Only available if the overwrite option is used.
	OldObject interface{}

	// RefillIndexCaches if set to true then refills the in-memory index caches.
	RefillIndexCaches *bool

	// If the intention is to delete existing attributes with the update-insert command, set it to false.
	// This modifies the behavior of the patch command to remove top-level attributes and sub-attributes from
	// the existing document that are contained in the patch document with an attribute value of null
	// (but not attributes of objects that are nested inside of arrays).
	// This option controls the update-insert behavior only.
	KeepNull *bool

	// Controls whether objects (not arrays) are merged if present in both, the existing and the update-insert document.
	// If set to false, the value in the patch document overwrites the existing document’s value.
	// If set to true, objects are merged. The default is true. This option controls the update-insert behavior only.
	MergeObjects *bool

	// Specify any top-level attribute to compare whether the version number is higher
	// than the currently stored one when updating or replacing documents.
	VersionAttribute string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

func (c *CollectionDocumentUpdateOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, c.IfMatch)
	}

	if c.WithWaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*c.WithWaitForSync))
	}

	if c.Silent != nil {
		r.AddQuery(QuerySilent, boolToString(*c.Silent))
	}

	if c.NewObject != nil {
		r.AddQuery(QueryReturnNew, "true")
	}

	if c.OldObject != nil {
		r.AddQuery(QueryReturnOld, "true")
	}

	if c.RefillIndexCaches != nil {
		r.AddQuery(QueryRefillIndexCaches, boolToString(*c.RefillIndexCaches))
	}

	if c.KeepNull != nil {
		r.AddQuery(QueryKeepNull, boolToString(*c.KeepNull))
	}

	if c.MergeObjects != nil {
		r.AddQuery(QueryMergeObjects, boolToString(*c.MergeObjects))
	}

	if c.IgnoreRevs != nil {
		r.AddQuery(QueryIgnoreRevs, boolToString(*c.IgnoreRevs))
	}

	if c.VersionAttribute != "" {
		r.AddQuery(QueryVersionAttribute, c.VersionAttribute)
	}
	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}

	return nil
}
