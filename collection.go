//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import "context"

// Collection provides access to the documents in a single collection.
type Collection interface {
	// Name returns the name of the collection.
	Name() string

	// ReadDocument reads a single document with given key from the collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error)

	// CreateDocument creates a single document in the collection.
	// The document data is loaded from the given document, the document meta data is returned.
	// If the document data already contains a `_key` field, this will be used as key of the new document,
	// otherwise a unique key is created.
	// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
	// To return the NEW document, prepare a context with `WithReturnNew`.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	CreateDocument(ctx context.Context, document interface{}) (DocumentMeta, error)

	// UpdateDocument updates a single document with given key in the collection.
	// The document meta data is returned.
	// To return the NEW document, prepare a context with `WithReturnNew`.
	// To return the OLD document, prepare a context with `WithReturnOld`.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	// If no document exists with given key, a NotFoundError is returned.
	UpdateDocument(ctx context.Context, key string, update map[string]interface{}) (DocumentMeta, error)

	// ReplaceDocument replaces a single document with given key in the collection with the document given in the document argument.
	// The document meta data is returned.
	// To return the NEW document, prepare a context with `WithReturnNew`.
	// To return the OLD document, prepare a context with `WithReturnOld`.
	// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
	// If no document exists with given key, a NotFoundError is returned.
	ReplaceDocument(ctx context.Context, key string, update map[string]interface{}) (DocumentMeta, error)

	// RemoveDocument removes a single document with given key from the collection.
	// The document meta data is returned.
	// To return the OLD document, prepare a context with `WithReturnOld`.
	// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
	// If no document exists with given key, a NotFoundError is returned.
	RemoveDocument(ctx context.Context, key string) (DocumentMeta, error)
}
