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

// CollectionDocumentRead contains methods for reading documents from a collection.
// https://docs.arangodb.com/stable/develop/http-api/documents/#get-a-document
type CollectionDocumentRead interface {
	// ReadDocument reads a single document with given key from the collection.
	// The document data is stored into result, the document metadata is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error)

	// ReadDocumentWithOptions reads a single document with given key from the collection.
	// The document data is stored into result, the document metadata is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocumentWithOptions(ctx context.Context, key string, result interface{}, opts *CollectionDocumentReadOptions) (DocumentMeta, error)

	// ReadDocuments reads multiple documents with given keys from the collection.
	// The documents data is stored into elements of the given results slice,
	// the documents metadata is returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	ReadDocuments(ctx context.Context, keys []string) (CollectionDocumentReadResponseReader, error)

	// ReadDocumentsWithOptions reads multiple documents with given keys from the collection.
	// The documents data is stored into elements of the given results slice and the documents metadata is returned.
	// If no document exists with a given key, a NotFoundError is returned at its errors index.
	// 'documents' must be a slice of structs with a `_key` field or a slice of keys.
	ReadDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentReadOptions) (CollectionDocumentReadResponseReader, error)
}

type CollectionDocumentReadResponseReader interface {
	Read(i interface{}) (CollectionDocumentReadResponse, error)
}

type CollectionDocumentReadResponse struct {
	DocumentMeta          `json:",inline"`
	shared.ResponseStruct `json:",inline"`
}

type CollectionDocumentReadOptions struct {
	// If the “If-Match” header is given, then it must contain exactly one ETag (_rev).
	// The document is returned, if it has the same revision as the given ETag
	// IMPORTANT: This will work only for single document read operations (CollectionDocumentRead.ReadDocument,
	// CollectionDocumentRead.ReadDocumentWithOptions)
	IfMatch string

	// If the “If-None-Match” header is given, then it must contain exactly one ETag (_rev).
	// The document is returned, if it has a different revision than the given ETag
	// IMPORTANT: This will work only for single document read operations (CollectionDocumentRead.ReadDocument,
	// CollectionDocumentRead.ReadDocumentWithOptions)
	IfNoneMatch string

	// By default, or if this is set to true, the _rev attributes in the given document is ignored.
	// If this is set to false, then the _rev attribute given in the body document is taken as a precondition.
	// The document is only removed if the current revision is the one specified.
	// This works only with multiple documents removal method CollectionDocumentRead.ReadDocumentsWithOptions
	IgnoreRevs *bool

	// Set this to true to allow the Coordinator to ask any shard replica for the data, not only the shard leader.
	// This may result in “dirty reads”.
	// This option is ignored if this operation is part of a DatabaseTransaction (TransactionID option).
	// The header set when creating the transaction decides about dirty reads for the entire transaction,
	// not the individual read operations.
	AllowDirtyReads *bool

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

func (c *CollectionDocumentReadOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.IfMatch != "" {
		r.AddHeader(HeaderIfMatch, c.IfMatch)
	}

	if c.IfNoneMatch != "" {
		r.AddHeader(HeaderIfNoneMatch, c.IfNoneMatch)
	}

	if c.IgnoreRevs != nil {
		r.AddQuery(QueryIgnoreRevs, boolToString(*c.IgnoreRevs))
	}

	if c.AllowDirtyReads != nil {
		r.AddHeader(HeaderDirtyReads, boolToString(*c.AllowDirtyReads))
	}

	if c.TransactionID != "" {
		r.AddHeader(HeaderTransaction, c.TransactionID)
	}

	return nil
}
