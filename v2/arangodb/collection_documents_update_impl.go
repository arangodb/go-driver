//
// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

func newCollectionDocumentUpdate(collection *collection) *collectionDocumentUpdate {
	return &collectionDocumentUpdate{
		collection: collection,
	}
}

var _ CollectionDocumentUpdate = &collectionDocumentUpdate{}

type collectionDocumentUpdate struct {
	collection *collection
}

func (c collectionDocumentUpdate) UpdateDocument(ctx context.Context, key string, document interface{}) (CollectionDocumentUpdateResponse, error) {
	return c.UpdateDocumentWithOptions(ctx, key, document, nil)
}

func (c collectionDocumentUpdate) UpdateDocumentWithOptions(ctx context.Context, key string, document interface{}, options *CollectionDocumentUpdateOptions) (CollectionDocumentUpdateResponse, error) {
	url := c.collection.url("document", key)

	var meta CollectionDocumentUpdateResponse

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

	resp, err := connection.CallPatch(ctx, c.collection.connection(), url, &response, document, c.collection.withModifiers(options.modifyRequest)...)
	if err != nil {
		return CollectionDocumentUpdateResponse{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return CollectionDocumentUpdateResponse{}, response.AsArangoErrorWithCode(code)
	}
}

func (c collectionDocumentUpdate) UpdateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentUpdateResponseReader, error) {
	return c.UpdateDocumentsWithOptions(ctx, documents, nil)
}

func (c collectionDocumentUpdate) UpdateDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentUpdateOptions) (CollectionDocumentUpdateResponseReader, error) {
	if !utils.IsListPtr(documents) && !utils.IsList(documents) {
		return nil, errors.Errorf("Input documents should be list")
	}

	url := c.collection.url("document")

	req, err := c.collection.connection().NewRequest(http.MethodPatch, url)
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
		return newCollectionDocumentUpdateResponseReader(&arr, opts), nil
	default:
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func newCollectionDocumentUpdateResponseReader(array *connection.Array, options *CollectionDocumentUpdateOptions) *collectionDocumentUpdateResponseReader {
	c := &collectionDocumentUpdateResponseReader{array: array, options: options}

	if c.options != nil {
		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}
	c.ReadAllReader = shared.ReadAllReader[CollectionDocumentUpdateResponse, *collectionDocumentUpdateResponseReader]{Reader: c}
	return c
}

var _ CollectionDocumentUpdateResponseReader = &collectionDocumentUpdateResponseReader{}

type collectionDocumentUpdateResponseReader struct {
	array    *connection.Array
	options  *CollectionDocumentUpdateOptions
	response struct {
		*DocumentMetaWithOldRev
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}
	shared.ReadAllReader[CollectionDocumentUpdateResponse, *collectionDocumentUpdateResponseReader]

	// Cache for len() method
	cachedResults []CollectionDocumentUpdateResponse
	cachedErrors  []error
	cached        bool
}

func (c *collectionDocumentUpdateResponseReader) Read() (CollectionDocumentUpdateResponse, error) {
	if !c.array.More() {
		return CollectionDocumentUpdateResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentUpdateResponse

	if c.options != nil {
		// Create new instances for each document to avoid reusing the same pointers
		if c.options.OldObject != nil {
			oldObjectType := reflect.TypeOf(c.options.OldObject).Elem()
			meta.Old = reflect.New(oldObjectType).Interface()
		}
		if c.options.NewObject != nil {
			newObjectType := reflect.TypeOf(c.options.NewObject).Elem()
			meta.New = reflect.New(newObjectType).Interface()
		}
	}

	c.response.DocumentMetaWithOldRev = &meta.DocumentMetaWithOldRev
	c.response.ResponseStruct = &meta.ResponseStruct

	if err := c.array.Unmarshal(&c.response); err != nil {
		if err == io.EOF {
			return CollectionDocumentUpdateResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentUpdateResponse{}, err
	}

	// Update meta with the unmarshaled data
	meta.DocumentMetaWithOldRev = *c.response.DocumentMetaWithOldRev
	meta.ResponseStruct = *c.response.ResponseStruct
	meta.Old = c.response.Old
	meta.New = c.response.New

	if meta.Error != nil && *meta.Error {
		return meta, meta.AsArangoError()
	}

	return meta, nil
}

// Len returns the number of items in the response
func (c *collectionDocumentUpdateResponseReader) Len() int {
	if !c.cached {
		c.cachedResults, c.cachedErrors = c.ReadAll()
		c.cached = true
	}
	return len(c.cachedResults)
}
