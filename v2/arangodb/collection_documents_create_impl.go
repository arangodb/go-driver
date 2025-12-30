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
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

func newCollectionDocumentCreate(collection *collection) *collectionDocumentCreate {
	return &collectionDocumentCreate{
		collection: collection,
	}
}

var _ CollectionDocumentCreate = &collectionDocumentCreate{}

type collectionDocumentCreate struct {
	collection *collection
}

func (c collectionDocumentCreate) CreateDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponseReader, error) {
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

	req, err := c.collection.connection().NewRequest(http.MethodPost, url)
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
		return newCollectionDocumentCreateResponseReader(&arr, opts, documentCount), nil
	default:
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c collectionDocumentCreate) CreateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentCreateResponseReader, error) {
	return c.CreateDocumentsWithOptions(ctx, documents, nil)
}

func (c collectionDocumentCreate) CreateDocumentWithOptions(ctx context.Context, document interface{}, options *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponse, error) {
	url := c.collection.url("document")

	var meta CollectionDocumentCreateResponse

	if options != nil {
		meta.Old = options.OldObject
		meta.New = options.NewObject
	}

	response := struct {
		*DocumentMeta          `json:",inline"`
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}{
		DocumentMeta:   &meta.DocumentMeta,
		ResponseStruct: &meta.ResponseStruct,

		Old: newUnmarshalInto(meta.Old),
		New: newUnmarshalInto(meta.New),
	}

	resp, err := connection.CallPost(ctx, c.collection.connection(), url, &response, document, c.collection.withModifiers(options.modifyRequest)...)
	if err != nil {
		return CollectionDocumentCreateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return CollectionDocumentCreateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collectionDocumentCreate) CreateDocument(ctx context.Context, document interface{}) (CollectionDocumentCreateResponse, error) {
	return c.CreateDocumentWithOptions(ctx, document, nil)
}

func newCollectionDocumentCreateResponseReader(array *connection.Array, options *CollectionDocumentCreateOptions, documentCount int) *collectionDocumentCreateResponseReader {
	c := &collectionDocumentCreateResponseReader{
		array:         array,
		options:       options,
		documentCount: documentCount,
	}

	if c.options != nil {
		if c.options.OldObject != nil {
			c.oldObjectType = reflect.TypeOf(c.options.OldObject)
		}
		if c.options.NewObject != nil {
			c.newObjectType = reflect.TypeOf(c.options.NewObject)
		}
		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}

	c.ReadAllReader = shared.ReadAllReader[CollectionDocumentCreateResponse, *collectionDocumentCreateResponseReader]{Reader: c}
	return c
}

var _ CollectionDocumentCreateResponseReader = &collectionDocumentCreateResponseReader{}

type collectionDocumentCreateResponseReader struct {
	array         *connection.Array
	options       *CollectionDocumentCreateOptions
	documentCount int          // Store input document count for Len() without caching
	oldObjectType reflect.Type // Cached type for OldObject to avoid repeated reflection
	newObjectType reflect.Type // Cached type for NewObject to avoid repeated reflection
	response      struct {
		*DocumentMeta
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}
	shared.ReadAllReader[CollectionDocumentCreateResponse, *collectionDocumentCreateResponseReader]
	mu sync.Mutex
}

func (c *collectionDocumentCreateResponseReader) Read() (CollectionDocumentCreateResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.array.More() {
		return CollectionDocumentCreateResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentCreateResponse

	// Create new instances for each document to avoid pointer reuse
	if c.options != nil {
		if c.options.OldObject != nil && c.oldObjectType != nil && c.oldObjectType.Kind() == reflect.Ptr {
			meta.Old = reflect.New(c.oldObjectType.Elem()).Interface()
		}
		if c.options.NewObject != nil && c.newObjectType != nil && c.newObjectType.Kind() == reflect.Ptr {
			meta.New = reflect.New(c.newObjectType.Elem()).Interface()
		}
	}

	c.response.DocumentMeta = &meta.DocumentMeta
	c.response.ResponseStruct = &meta.ResponseStruct
	c.response.Old = newUnmarshalInto(meta.Old)
	c.response.New = newUnmarshalInto(meta.New)

	if err := c.array.Unmarshal(&c.response); err != nil {
		if err == io.EOF {
			return CollectionDocumentCreateResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentCreateResponse{}, err
	}

	if meta.Error != nil && *meta.Error {
		return meta, meta.AsArangoError()
	}

	// Copy data from the new instances back to the original option objects for backward compatibility.
	// NOTE: The mutex protects both the reader's internal state AND writes to c.options.OldObject/NewObject.
	// Multiple goroutines calling Read() on the same reader will serialize through the mutex, preventing races.
	// However, if callers access c.options.OldObject/NewObject from other goroutines (outside of Read()),
	// they must provide their own synchronization as those accesses are not protected by this reader's mutex.
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
func (c *collectionDocumentCreateResponseReader) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.documentCount
}
