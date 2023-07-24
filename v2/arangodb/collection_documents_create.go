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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

// CollectionDocumentCreate interface for creating documents in a collection.
// https://www.arangodb.com/docs/devel/http/document.html#create-a-document
type CollectionDocumentCreate interface {

	// CreateDocument creates a single document in the collection.
	// The document data is loaded from the given document, the document metadata is returned.
	// If the document data already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
	CreateDocument(ctx context.Context, document interface{}) (CollectionDocumentCreateResponse, error)

	// CreateDocumentWithOptions creates a single document in the collection.
	// The document data is loaded from the given document, the document metadata is returned.
	// If the document data already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
	CreateDocumentWithOptions(ctx context.Context, document interface{}, options *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponse, error)

	// CreateDocuments creates multiple documents in the collection.
	// The document data is loaded from the given documents slice, the documents metadata is returned.
	// If a documents element already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// If a documents element contains a `_key` field with a duplicate key, other any other field violates an index constraint,
	// a ConflictError is returned in its indeed in the errors slice.
	// To return the NEW documents, prepare a context with `WithReturnNew`. The data argument passed to `WithReturnNew` must be
	// a slice with the same number of entries as the `documents` slice.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	// If the create request itself fails or one of the arguments is invalid, an error is returned.
	CreateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentCreateResponseReader, error)

	// CreateDocumentsWithOptions creates multiple documents in the collection.
	// The document data is loaded from the given documents slice, the documents metadata is returned.
	// If a documents element already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// If a documents element contains a `_key` field with a duplicate key, other any other field violates an index constraint,
	// a ConflictError is returned in its indeed in the errors slice.
	// To return the NEW documents, prepare a context with `WithReturnNew`. The data argument passed to `WithReturnNew` must be
	// a slice with the same number of entries as the `documents` slice.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	// If the create request itself fails or one of the arguments is invalid, an error is returned.
	CreateDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponseReader, error)
}

type CollectionDocumentCreateResponseReader interface {
	Read() (CollectionDocumentCreateResponse, error)
}

type CollectionDocumentCreateResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	Old, New              interface{}
}

type CollectionDocumentCreateOverwriteMode string

func (c *CollectionDocumentCreateOverwriteMode) New() *CollectionDocumentCreateOverwriteMode {
	return c
}

func (c *CollectionDocumentCreateOverwriteMode) Get() CollectionDocumentCreateOverwriteMode {
	if c == nil {
		return CollectionDocumentCreateOverwriteModeConflict
	}

	return *c
}

func (c *CollectionDocumentCreateOverwriteMode) String() string {
	return string(c.Get())
}

const (
	CollectionDocumentCreateOverwriteModeIgnore   CollectionDocumentCreateOverwriteMode = "ignore"
	CollectionDocumentCreateOverwriteModeReplace  CollectionDocumentCreateOverwriteMode = "replace"
	CollectionDocumentCreateOverwriteModeUpdate   CollectionDocumentCreateOverwriteMode = "update"
	CollectionDocumentCreateOverwriteModeConflict CollectionDocumentCreateOverwriteMode = "conflict"
)

type CollectionDocumentCreateOptions struct {
	// Wait until document has been synced to disk.
	WithWaitForSync *bool

	// If set to true, the insert becomes a replace-insert.
	// If a document with the same _key already exists,
	// the new document is not rejected with unique constraint violation error but replaces the old document.
	// Note that operations with overwrite parameter require a _key attribute in the request payload,
	// therefore they can only be performed on collections sharded by _key.
	Overwrite *bool

	// This option supersedes `overwrite` option.
	OverwriteMode *CollectionDocumentCreateOverwriteMode

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
	// This option controls the update-insert behavior only (CollectionDocumentCreateOverwriteModeUpdate).
	KeepNull *bool

	// Controls whether objects (not arrays) are merged if present in both, the existing and the update-insert document.
	// If set to false, the value in the patch document overwrites the existing document’s value.
	// If set to true, objects are merged. The default is true. This option controls the update-insert behavior only.
	// This option controls the update-insert behavior only (CollectionDocumentCreateOverwriteModeUpdate).
	MergeObjects *bool
}

func (c *CollectionDocumentCreateOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.WithWaitForSync != nil {
		r.AddQuery("waitForSync", boolToString(*c.WithWaitForSync))
	}

	if c.Overwrite != nil {
		r.AddQuery("overwrite", boolToString(*c.Overwrite))
	}

	if c.OverwriteMode != nil {
		r.AddQuery("overwriteMode", c.OverwriteMode.String())
	}

	if c.Silent != nil {
		r.AddQuery("silent", boolToString(*c.Silent))
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

	if c.KeepNull != nil {
		r.AddQuery("keepNull", boolToString(*c.KeepNull))
	}

	if c.MergeObjects != nil {
		r.AddQuery("mergeObjects", boolToString(*c.MergeObjects))
	}

	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
