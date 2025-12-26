//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
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

func newCollectionDocumentReplace(collection *collection) *collectionDocumentReplace {
	return &collectionDocumentReplace{
		collection: collection,
	}
}

var _ CollectionDocumentReplace = &collectionDocumentReplace{}

type collectionDocumentReplace struct {
	collection *collection
}

func (c collectionDocumentReplace) ReplaceDocument(ctx context.Context, key string, document interface{}) (CollectionDocumentReplaceResponse, error) {
	return c.ReplaceDocumentWithOptions(ctx, key, document, nil)
}

func (c collectionDocumentReplace) ReplaceDocumentWithOptions(ctx context.Context, key string, document interface{}, options *CollectionDocumentReplaceOptions) (CollectionDocumentReplaceResponse, error) {
	url := c.collection.url("document", key)

	var meta CollectionDocumentReplaceResponse

	if options != nil {
		meta.Old = options.OldObject
		meta.New = options.NewObject
	}

	response := struct {
		*DocumentMetaWithOldRev `json:",inline"`
		*shared.ResponseStruct  `json:",inline"`
		Old                     *UnmarshalInto `json:"old,omitempty"`
		New                     *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMetaWithOldRev: &meta.DocumentMetaWithOldRev,
		ResponseStruct:         &meta.ResponseStruct,

		Old: newUnmarshalInto(meta.Old),
		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPut(ctx, c.collection.connection(), url, &response, document, c.collection.withModifiers(options.modifyRequest)...)
	if err != nil {
		return CollectionDocumentReplaceResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return CollectionDocumentReplaceResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collectionDocumentReplace) ReplaceDocuments(ctx context.Context, documents interface{}) (CollectionDocumentReplaceResponseReader, error) {
	return c.ReplaceDocumentsWithOptions(ctx, documents, nil)
}

func (c collectionDocumentReplace) ReplaceDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentReplaceOptions) (CollectionDocumentReplaceResponseReader, error) {
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

	req, err := c.collection.connection().NewRequest(http.MethodPut, url)
	if err != nil {
		return nil, err
	}

	for _, modifier := range c.collection.withModifiers(opts.modifyRequest, connection.WithBody(documents), connection.WithFragment("multiple")) {
		if err = modifier(req); err != nil {
			return nil, err
		}
	}

	var arr connection.Array

	resp, err := c.collection.connection().Do(ctx, req, &arr)
	if err != nil {
		return nil, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return newCollectionDocumentReplaceResponseReader(&arr, opts, documentCount), nil
	default:
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func newCollectionDocumentReplaceResponseReader(array *connection.Array, options *CollectionDocumentReplaceOptions, documentCount int) *collectionDocumentReplaceResponseReader {
	c := &collectionDocumentReplaceResponseReader{
		array:         array,
		options:       options,
		documentCount: documentCount,
	}

	if c.options != nil {
		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}
	c.ReadAllReader = shared.ReadAllReader[CollectionDocumentReplaceResponse, *collectionDocumentReplaceResponseReader]{Reader: c}
	return c
}

var _ CollectionDocumentReplaceResponseReader = &collectionDocumentReplaceResponseReader{}

type collectionDocumentReplaceResponseReader struct {
	array         *connection.Array
	options       *CollectionDocumentReplaceOptions
	documentCount int // Store input document count for Len() without caching
	response      struct {
		*DocumentMetaWithOldRev
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}
	shared.ReadAllReader[CollectionDocumentReplaceResponse, *collectionDocumentReplaceResponseReader]

	mu sync.Mutex
}

func (c *collectionDocumentReplaceResponseReader) Read() (CollectionDocumentReplaceResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.array.More() {
		return CollectionDocumentReplaceResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentReplaceResponse

	// Create new instances for each document to avoid pointer reuse
	if c.options != nil {
		if c.options.OldObject != nil {
			oldObjectType := reflect.TypeOf(c.options.OldObject)
			if oldObjectType != nil && oldObjectType.Kind() == reflect.Ptr {
				meta.Old = reflect.New(oldObjectType.Elem()).Interface()
			}
		}
		if c.options.NewObject != nil {
			newObjectType := reflect.TypeOf(c.options.NewObject)
			if newObjectType != nil && newObjectType.Kind() == reflect.Ptr {
				meta.New = reflect.New(newObjectType.Elem()).Interface()
			}
		}
	}

	c.response.DocumentMetaWithOldRev = &meta.DocumentMetaWithOldRev
	c.response.ResponseStruct = &meta.ResponseStruct
	c.response.Old = newUnmarshalInto(meta.Old)
	c.response.New = newUnmarshalInto(meta.New)

	if err := c.array.Unmarshal(&c.response); err != nil {
		if err == io.EOF {
			return CollectionDocumentReplaceResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentReplaceResponse{}, err
	}

	if meta.Error != nil && *meta.Error {
		return meta, meta.AsArangoError()
	}

	// Copy data from the new instances back to the original option objects for backward compatibility.
	// NOTE: The mutex protects concurrent Read() calls on this reader instance, but does not protect
	// the options object itself. If the same options object is shared across multiple readers or
	// accessed from other goroutines, there will be a data race. Options objects should not be
	// shared across concurrent operations.
	if c.options != nil {
		if c.options.OldObject != nil && meta.Old != nil {
			oldValue := reflect.ValueOf(meta.Old)
			originalValue := reflect.ValueOf(c.options.OldObject)
			if oldValue.IsValid() && oldValue.Kind() == reflect.Ptr && !oldValue.IsNil() &&
				originalValue.IsValid() && originalValue.Kind() == reflect.Ptr && !originalValue.IsNil() {
				originalValue.Elem().Set(oldValue.Elem())
			}
		}
		if c.options.NewObject != nil && meta.New != nil {
			newValue := reflect.ValueOf(meta.New)
			originalValue := reflect.ValueOf(c.options.NewObject)
			if newValue.IsValid() && newValue.Kind() == reflect.Ptr && !newValue.IsNil() &&
				originalValue.IsValid() && originalValue.Kind() == reflect.Ptr && !originalValue.IsNil() {
				originalValue.Elem().Set(newValue.Elem())
			}
		}
	}

	return meta, nil
}

// Len returns the number of items in the response.
// Returns the input document count immediately without reading/caching (same as v1 behavior).
// After calling Len(), you can still use Read() to iterate through items.
func (c *collectionDocumentReplaceResponseReader) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.documentCount
}
