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
		return newCollectionDocumentReplaceResponseReader(&arr, opts), nil
	default:
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func newCollectionDocumentReplaceResponseReader(array *connection.Array, options *CollectionDocumentReplaceOptions) *collectionDocumentReplaceResponseReader {
	c := &collectionDocumentReplaceResponseReader{array: array, options: options}

	if c.options != nil {
		// Cache reflection types once during initialization for performance
		if c.options.OldObject != nil {
			c.oldType = reflect.TypeOf(c.options.OldObject).Elem()
		}
		if c.options.NewObject != nil {
			c.newType = reflect.TypeOf(c.options.NewObject).Elem()
		}

		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}
	c.ReadAllReader = shared.ReadAllReader[CollectionDocumentReplaceResponse, *collectionDocumentReplaceResponseReader]{Reader: c}
	return c
}

var _ CollectionDocumentReplaceResponseReader = &collectionDocumentReplaceResponseReader{}

type collectionDocumentReplaceResponseReader struct {
	array    *connection.Array
	options  *CollectionDocumentReplaceOptions
	response struct {
		*DocumentMetaWithOldRev
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}
	shared.ReadAllReader[CollectionDocumentReplaceResponse, *collectionDocumentReplaceResponseReader]

	// Cache for len() method - allows Read() to work after Len() is called
	cachedResults []CollectionDocumentReplaceResponse
	cachedErrors  []error
	cached        bool
	readIndex     int // Track position in cache for Read() after Len()

	// Performance: Cache reflection types to avoid repeated lookups
	oldType reflect.Type
	newType reflect.Type
}

func (c *collectionDocumentReplaceResponseReader) Read() (CollectionDocumentReplaceResponse, error) {
	// If Len() was called, serve from cache
	if c.cached {
		if c.readIndex >= len(c.cachedResults) {
			return CollectionDocumentReplaceResponse{}, shared.NoMoreDocumentsError{}
		}
		result := c.cachedResults[c.readIndex]
		err := c.cachedErrors[c.readIndex]
		c.readIndex++
		return result, err
	}

	// Normal streaming read
	if !c.array.More() {
		return CollectionDocumentReplaceResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentReplaceResponse

	// Create new instances for each document to avoid pointer reuse
	// Use cached types for performance
	if c.oldType != nil {
		meta.Old = reflect.New(c.oldType).Interface()
	}
	if c.newType != nil {
		meta.New = reflect.New(c.newType).Interface()
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

	// Copy data from the new instances back to the original option objects for backward compatibility
	if c.options != nil {
		if c.options.OldObject != nil && meta.Old != nil {
			oldValue := reflect.ValueOf(meta.Old).Elem()
			originalValue := reflect.ValueOf(c.options.OldObject).Elem()
			originalValue.Set(oldValue)
		}
		if c.options.NewObject != nil && meta.New != nil {
			newValue := reflect.ValueOf(meta.New).Elem()
			originalValue := reflect.ValueOf(c.options.NewObject).Elem()
			originalValue.Set(newValue)
		}
	}

	return meta, nil
}

// Len returns the number of items in the response.
// After calling Len(), you can still use Read() to iterate through items.
func (c *collectionDocumentReplaceResponseReader) Len() int {
	if !c.cached {
		c.cachedResults, c.cachedErrors = c.ReadAll()
		c.cached = true
		c.readIndex = 0 // Reset read position to allow Read() after Len()
	}
	return len(c.cachedResults)
}
