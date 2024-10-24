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

// CollectionDocumentReplace replaces document(s) with given key(s) in the collection
// https://docs.arangodb.com/stable/develop/http-api/documents/#replace-a-document
type CollectionDocumentReplace interface {

	// ReplaceDocument replaces a single document with given key in the collection.
	// If no document exists with given key, a NotFoundError is returned.
	// If `_id` field is present in the document body, it is always ignored.
	ReplaceDocument(ctx context.Context, key string, document interface{}) (CollectionDocumentReplaceResponse, error)

	// ReplaceDocumentWithOptions replaces a single document with given key in the collection.
	// If no document exists with given key, a NotFoundError is returned.
	// If `_id` field is present in the document body, it is always ignored.
	ReplaceDocumentWithOptions(ctx context.Context, key string, document interface{}, options *CollectionDocumentReplaceOptions) (CollectionDocumentReplaceResponse, error)

	// ReplaceDocuments replaces multiple document with given keys in the collection.
	// The replaces are loaded from the given replaces slice, the documents metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// Each element in the replaces slice must contain a `_key` field.
	// If `_id` field is present in the document body, it is always ignored.
	ReplaceDocuments(ctx context.Context, documents interface{}) (CollectionDocumentReplaceResponseReader, error)

	// ReplaceDocumentsWithOptions replaces multiple document with given keys in the collection.
	// The replaces are loaded from the given replaces slice, the documents metadata are returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// Each element in the replaces slice must contain a `_key` field.
	// If `_id` field is present in the document body, it is always ignored.
	ReplaceDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentReplaceOptions) (CollectionDocumentReplaceResponseReader, error)
}

type CollectionDocumentReplaceResponseReader interface {
	Read() (CollectionDocumentReplaceResponse, error)
}

type CollectionDocumentReplaceResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	Old, New              interface{}
}

type CollectionDocumentReplaceOptions struct {
	// Conditionally replace a document based on a target revision id
	// IMPORTANT: This will work only for single document replace operations (CollectionDocumentReplace.ReplaceDocument,
	// CollectionDocumentReplace.ReplaceDocumentWithOptions)
	IfMatch string `json:"ifMatch,omitempty"`

	// By default, or if this is set to true, the _rev attributes in the given document is ignored.
	// If this is set to false, then the _rev attribute given in the body document is taken as a precondition.
	// The document is only replaced if the current revision is the one specified.
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

	// IsRestore is used to make insert functions use the "isRestore=<value>" setting.
	// Note: This option is intended for internal (replication) use.
	// It is NOT intended to be used by normal client. Use on your own risk!
	IsRestore *bool

	// Specify any top-level attribute to compare whether the version number is higher
	// than the currently stored one when updating or replacing documents.
	VersionAttribute string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

func (c *CollectionDocumentReplaceOptions) modifyRequest(r connection.Request) error {
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

	if c.IgnoreRevs != nil {
		r.AddQuery(QueryIgnoreRevs, boolToString(*c.IgnoreRevs))
	}

	if c.IsRestore != nil {
		r.AddQuery(QueryIsRestore, boolToString(*c.IsRestore))
	}

	if c.VersionAttribute != "" {
		r.AddQuery(QueryVersionAttribute, c.VersionAttribute)
	}

	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}
	return nil
}
