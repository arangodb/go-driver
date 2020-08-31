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

type CollectionDocumentCreate interface {

	// CreateDocument creates a single document in the collection.
	// The document data is loaded from the given document, the document meta data is returned.
	// If the document data already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
	CreateDocument(ctx context.Context, document interface{}) (CollectionDocumentCreateResponse, error)

	// CreateDocument creates a single document in the collection.
	// The document data is loaded from the given document, the document meta data is returned.
	// If the document data already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
	CreateDocumentWithOptions(ctx context.Context, document interface{}, options *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponse, error)

	// CreateDocuments creates multiple documents in the collection.
	// The document data is loaded from the given documents slice, the documents meta data is returned.
	// If a documents element already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// If a documents element contains a `_key` field with a duplicate key, other any other field violates an index constraint,
	// a ConflictError is returned in its inded in the errors slice.
	// To return the NEW documents, prepare a context with `WithReturnNew`. The data argument passed to `WithReturnNew` must be
	// a slice with the same number of entries as the `documents` slice.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	// If the create request itself fails or one of the arguments is invalid, an error is returned.
	CreateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentCreateResponseReader, error)

	// CreateDocuments creates multiple documents in the collection.
	// The document data is loaded from the given documents slice, the documents meta data is returned.
	// If a documents element already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// If a documents element contains a `_key` field with a duplicate key, other any other field violates an index constraint,
	// a ConflictError is returned in its inded in the errors slice.
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

func (c CollectionDocumentCreateOverwriteMode) New() *CollectionDocumentCreateOverwriteMode {
	return &c
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
	WithWaitForSync *bool
	Overwrite       *bool
	Silent          *bool
	OverwriteMode   *CollectionDocumentCreateOverwriteMode
	NewObject       interface{}
	OldObject       interface{}
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

	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
