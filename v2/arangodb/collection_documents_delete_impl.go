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
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

func newCollectionDocumentDelete(collection *collection) *collectionDocumentDelete {
	return &collectionDocumentDelete{
		collection: collection,
	}
}

var _ CollectionDocumentDelete = &collectionDocumentDelete{}

type collectionDocumentDelete struct {
	collection *collection
}

func (c collectionDocumentDelete) DeleteDocument(ctx context.Context, key string) (CollectionDocumentDeleteResponse, error) {
	return c.DeleteDocumentWithOptions(ctx, key, nil)
}

func (c collectionDocumentDelete) DeleteDocumentWithOptions(ctx context.Context, key string, opts *CollectionDocumentDeleteOptions) (CollectionDocumentDeleteResponse, error) {
	url := c.collection.url("document", key)

	var meta CollectionDocumentDeleteResponse
	if opts != nil {
		meta.Old = opts.OldObject
	}

	resp, err := connection.CallDelete(ctx, c.collection.connection(), url, &meta, c.collection.withModifiers(opts.modifyRequest)...)
	if err != nil {
		return CollectionDocumentDeleteResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK, http.StatusAccepted:
		return meta, nil
	default:
		return CollectionDocumentDeleteResponse{}, meta.AsArangoErrorWithCode(code)
	}
}

func (c collectionDocumentDelete) DeleteDocuments(ctx context.Context, keys []string) (CollectionDocumentDeleteResponseReader, error) {
	return c.DeleteDocumentsWithOptions(ctx, keys, nil)
}

func (c collectionDocumentDelete) DeleteDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentDeleteOptions) (CollectionDocumentDeleteResponseReader, error) {
	if !utils.IsListPtr(documents) && !utils.IsList(documents) {
		return nil, errors.Errorf("Input documents should be list")
	}

	// Get document count from input (same as v1 approach)
	documentsVal := reflect.ValueOf(documents)
	if documentsVal.Kind() == reflect.Ptr {
		documentsVal = documentsVal.Elem()
	}
	documentCount := documentsVal.Len()

	url := c.collection.url("document")

	req, err := c.collection.connection().NewRequest(http.MethodDelete, url)
	if err != nil {
		return nil, err
	}

	for _, modifier := range c.collection.withModifiers(opts.modifyRequest, connection.WithBody(documents)) {
		if err = modifier(req); err != nil {
			return nil, err
		}
	}

	var arr connection.Array

	_, err = c.collection.connection().Do(ctx, req, &arr, http.StatusOK, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	return newCollectionDocumentDeleteResponseReader(&arr, opts, documentCount), nil
}

func newCollectionDocumentDeleteResponseReader(array *connection.Array, options *CollectionDocumentDeleteOptions, documentCount int) *collectionDocumentDeleteResponseReader {
	c := &collectionDocumentDeleteResponseReader{
		array:         array,
		options:       options,
		documentCount: documentCount,
	}

	if options != nil && options.OldObject != nil {
		c.oldObjectType = reflect.TypeOf(options.OldObject)
	}

	c.ReadAllIntoReader = shared.ReadAllIntoReader[CollectionDocumentDeleteResponse, *collectionDocumentDeleteResponseReader]{Reader: c}
	return c
}

var _ CollectionDocumentDeleteResponseReader = &collectionDocumentDeleteResponseReader{}

type collectionDocumentDeleteResponseReader struct {
	array         *connection.Array
	options       *CollectionDocumentDeleteOptions
	documentCount int          // Store input document count for Len() without caching
	oldObjectType reflect.Type // Cached type for OldObject to avoid repeated reflection
	shared.ReadAllIntoReader[CollectionDocumentDeleteResponse, *collectionDocumentDeleteResponseReader]
	mu sync.Mutex
}

func (c *collectionDocumentDeleteResponseReader) Read(i interface{}) (CollectionDocumentDeleteResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.array.More() {
		return CollectionDocumentDeleteResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentDeleteResponse

	var response Unmarshal[shared.ResponseStruct, Unmarshal[DocumentMeta, UnmarshalData]]

	if err := c.array.Unmarshal(&response); err != nil {
		if err == io.EOF {
			return CollectionDocumentDeleteResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentDeleteResponse{}, err
	}

	if q := response.Current; q != nil {
		meta.ResponseStruct = *q
	}

	if q := response.Object.Current; q != nil {
		meta.DocumentMeta = *q
	}

	if meta.Error != nil && *meta.Error {
		return meta, meta.AsArangoError()
	}

	if err := response.Object.Object.Inject(i); err != nil {
		return CollectionDocumentDeleteResponse{}, err
	}

	if c.options != nil && c.options.OldObject != nil && c.oldObjectType != nil {
		// Create a new instance for each document to avoid pointer reuse
		if c.oldObjectType.Kind() == reflect.Ptr {
			meta.Old = reflect.New(c.oldObjectType.Elem()).Interface()

			// Extract old data into the new instance
			if err := response.Object.Object.Extract("old").Inject(meta.Old); err != nil {
				return CollectionDocumentDeleteResponse{}, err
			}

			// Copy data from the new instance to the original OldObject for backward compatibility.
			// NOTE: The mutex protects concurrent Read() calls on this reader instance, but does not protect
			// the options object itself. If the same options object is shared across multiple readers or
			// accessed from other goroutines, there will be a data race. Options objects should not be
			// shared across concurrent operations.
			oldValue := reflect.ValueOf(meta.Old)
			originalValue := reflect.ValueOf(c.options.OldObject)
			if oldValue.IsValid() && oldValue.Kind() == reflect.Ptr && !oldValue.IsNil() &&
				originalValue.IsValid() && originalValue.Kind() == reflect.Ptr && !originalValue.IsNil() {
				originalValue.Elem().Set(oldValue.Elem())
			}
		}
	}

	return meta, nil
}

// Len returns the number of items in the response.
// Returns the input document count immediately without reading/caching (same as v1 behavior).
// After calling Len(), you can still use Read() to iterate through items.
func (c *collectionDocumentDeleteResponseReader) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.documentCount
}
