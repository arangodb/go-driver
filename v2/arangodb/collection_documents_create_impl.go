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
		return newCollectionDocumentCreateResponseReader(&arr, opts), nil
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

func newCollectionDocumentCreateResponseReader(array *connection.Array, options *CollectionDocumentCreateOptions) *collectionDocumentCreateResponseReader {
	c := &collectionDocumentCreateResponseReader{array: array, options: options}

	if c.options != nil {
		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}

	return c
}

var _ CollectionDocumentCreateResponseReader = &collectionDocumentCreateResponseReader{}

type collectionDocumentCreateResponseReader struct {
	array    *connection.Array
	options  *CollectionDocumentCreateOptions
	response struct {
		*DocumentMeta
		*shared.ResponseStruct `json:",inline"`
		Old                    *UnmarshalInto `json:"old,omitempty"`
		New                    *UnmarshalInto `json:"new,omitempty"`
	}
}

func (c *collectionDocumentCreateResponseReader) Read() (CollectionDocumentCreateResponse, error) {
	if !c.array.More() {
		return CollectionDocumentCreateResponse{}, shared.NoMoreDocumentsError{}
	}

	var meta CollectionDocumentCreateResponse

	if c.options != nil {
		meta.Old = c.options.OldObject
		meta.New = c.options.NewObject
	}

	c.response.DocumentMeta = &meta.DocumentMeta
	c.response.ResponseStruct = &meta.ResponseStruct

	if err := c.array.Unmarshal(&c.response); err != nil {
		if err == io.EOF {
			return CollectionDocumentCreateResponse{}, shared.NoMoreDocumentsError{}
		}
		return CollectionDocumentCreateResponse{}, err
	}

	if meta.Error != nil && *meta.Error {
		return meta, meta.AsArangoError()
	}

	return meta, nil
}
