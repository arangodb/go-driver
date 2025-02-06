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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCollectionDocumentRead(collection *collection) *collectionDocumentRead {
	return &collectionDocumentRead{
		collection: collection,
	}
}

var _ CollectionDocumentRead = &collectionDocumentRead{}

type collectionDocumentRead struct {
	collection *collection
}

func (c collectionDocumentRead) ReadDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentReadOptions) (CollectionDocumentReadResponseReader, error) {
	url := c.collection.url("document")

	req, err := c.collection.connection().NewRequest(http.MethodPut, url)
	if err != nil {
		return nil, err
	}

	for _, modifier := range c.collection.withModifiers(opts.modifyRequest, connection.WithBody(documents),
		connection.WithFragment("get"), connection.WithQuery("onlyget", "true")) {
		if err = modifier(req); err != nil {
			return nil, err
		}
	}

	var arr connection.Array
	if _, err := c.collection.connection().Do(ctx, req, &arr, http.StatusOK); err != nil {
		return nil, err
	}
	return newCollectionDocumentReadResponseReader(&arr, opts), nil
}

func (c collectionDocumentRead) ReadDocuments(ctx context.Context, keys []string) (CollectionDocumentReadResponseReader, error) {
	return c.ReadDocumentsWithOptions(ctx, keys, nil)
}

func (c collectionDocumentRead) ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error) {
	return c.ReadDocumentWithOptions(ctx, key, result, nil)
}

func (c collectionDocumentRead) ReadDocumentWithOptions(ctx context.Context, key string, result interface{}, opts *CollectionDocumentReadOptions) (DocumentMeta, error) {
	url := c.collection.url("document", key)

	var response Unmarshal[shared.ResponseStruct, Unmarshal[DocumentMeta, UnmarshalData]]

	resp, err := connection.CallGet(ctx, c.collection.connection(), url, &response, c.collection.withModifiers(opts.modifyRequest)...)
	if err != nil {
		return DocumentMeta{}, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		if err := response.Object.Object.Inject(result); err != nil {
			return DocumentMeta{}, err
		}
		if z := response.Object.Current; z != nil {
			return *z, nil
		}
		return DocumentMeta{}, nil
	default:
		return DocumentMeta{}, response.Current.AsArangoErrorWithCode(code)
	}
}

func newCollectionDocumentReadResponseReader(array *connection.Array, options *CollectionDocumentReadOptions) *collectionDocumentReadResponseReader {
	c := &collectionDocumentReadResponseReader{array: array, options: options}

	return c
}

var _ CollectionDocumentReadResponseReader = &collectionDocumentReadResponseReader{}

type collectionDocumentReadResponseReader struct {
	array   *connection.Array
	options *CollectionDocumentReadOptions
}

func (c *collectionDocumentReadResponseReader) Read(i interface{}) (CollectionDocumentReadResponse, error) {
	if !c.array.More() {
		return CollectionDocumentReadResponse{}, shared.NoMoreDocumentsError{}
	}

	var response Unmarshal[shared.ResponseStruct, Unmarshal[DocumentMeta, UnmarshalData]]

	if err := c.array.Unmarshal(&response); err != nil {
		if err == io.EOF {
			return CollectionDocumentReadResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentReadResponse{}, err
	}

	var meta CollectionDocumentReadResponse

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
		return CollectionDocumentReadResponse{}, err
	}

	return meta, nil
}
