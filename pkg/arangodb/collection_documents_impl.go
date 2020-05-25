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

package arangodb

import (
	"context"
	"net/http"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/pkg/connection"
	"github.com/arangodb/go-driver/util"
	"github.com/pkg/errors"
)

func newCollectionDocuments(collection *collection) *collectionDocuments {
	d := &collectionDocuments{collection: collection}

	return d
}

var (
	_ CollectionDocuments = &collectionDocuments{}
)

type collectionDocuments struct {
	collection *collection
}

func (c collectionDocuments) ReadDocumentsWithOptions(ctx context.Context, keys []string, opts *CollectionDocumentReadOptions) (CollectionDocumentReadResponseReader, error) {
	url := c.collection.url("document")

	req, err := c.collection.connection().NewRequest(http.MethodPut, url)
	if err != nil {
		return nil, err
	}

	for _, modifier := range c.collection.withModifiers(opts.modifyRequest, connection.WithBody(keys), connection.WithFragment("get"), connection.WithQuery("onlyget", "true")) {
		if err = modifier(req); err != nil {
			return nil, err
		}
	}

	resp, arr, err := c.collection.connection().DoWithArray(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.Code() {
	case http.StatusOK:
		return newCollectionDocumentReadResponseReader(arr, opts), nil
	default:
		arr.Close()
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c collectionDocuments) ReadDocuments(ctx context.Context, keys []string) (CollectionDocumentReadResponseReader, error) {
	return c.ReadDocumentsWithOptions(ctx, keys, nil)
}

func (c collectionDocuments) ReadDocument(ctx context.Context, key string, result interface{}) (driver.DocumentMeta, error) {
	return c.ReadDocumentWithOptions(ctx, key, result, nil)
}

func (c collectionDocuments) ReadDocumentWithOptions(ctx context.Context, key string, result interface{}, opts *CollectionDocumentReadOptions) (driver.DocumentMeta, error) {
	url := c.collection.url("document", key)

	var meta driver.DocumentMeta

	response := struct {
		*driver.DocumentMeta
		*unmarshalInto
	}{
		DocumentMeta:  &meta,
		unmarshalInto: newUnmarshalInto(result),
	}

	resp, err := connection.CallGet(ctx, c.collection.connection(), url, &response, c.collection.modifiers...)
	if err != nil {
		return driver.DocumentMeta{}, err
	}

	switch resp.Code() {
	case http.StatusOK:
		return meta, nil
	default:
		return driver.DocumentMeta{}, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c collectionDocuments) CreateDocumentsWithOptions(ctx context.Context, documents interface{}, opts *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponseReader, error) {
	if !util.IsListPtr(documents) && !util.IsList(documents) {
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

	resp, arr, err := c.collection.connection().DoWithArray(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.Code() {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return &collectionDocumentCreateResponseReader{array: arr, options: opts}, nil
	default:
		arr.Close()
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c collectionDocuments) CreateDocuments(ctx context.Context, documents interface{}) (CollectionDocumentCreateResponseReader, error) {
	return c.CreateDocumentsWithOptions(ctx, documents, nil)
}

func (c collectionDocuments) CreateDocumentWithOptions(ctx context.Context, document interface{}, options *CollectionDocumentCreateOptions) (CollectionDocumentCreateResponse, error) {
	url := c.collection.url("document")

	var meta CollectionDocumentCreateResponse

	if options != nil {
		meta.Old = options.OldObject
		meta.New = options.NewObject
	}

	response := struct {
		*driver.DocumentMeta
		*ResponseStruct

		Old *unmarshalInto `json:"old,omitempty"`
		New *unmarshalInto `json:"new,omitempty"`
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

	switch resp.Code() {
	case http.StatusCreated:
		fallthrough
	case http.StatusAccepted:
		return meta, nil
	default:
		return CollectionDocumentCreateResponse{}, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (c collectionDocuments) CreateDocument(ctx context.Context, document interface{}) (CollectionDocumentCreateResponse, error) {
	return c.CreateDocumentWithOptions(ctx, document, nil)
}

func (c collectionDocuments) DocumentExists(ctx context.Context, key string) (bool, error) {
	url := c.collection.url("document", key)

	resp, err := connection.CallHead(ctx, c.collection.connection(), url, nil, c.collection.withModifiers()...)
	if err != nil {
		if connection.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	switch resp.Code() {
	case http.StatusOK:
		return true, nil
	default:
		return false, connection.NewError(resp.Code(), "unexpected code")
	}
}

// HELPERS

func newCollectionDocumentCreateResponseReader(array connection.Array, options *CollectionDocumentCreateOptions) *collectionDocumentCreateResponseReader {
	c := &collectionDocumentCreateResponseReader{array: array, options: options}

	if c.options != nil {
		c.response.Old = newUnmarshalInto(c.options.OldObject)
		c.response.New = newUnmarshalInto(c.options.NewObject)
	}

	return c
}

var _ CollectionDocumentCreateResponseReader = &collectionDocumentCreateResponseReader{}

type collectionDocumentCreateResponseReader struct {
	array    connection.Array
	options  *CollectionDocumentCreateOptions
	response struct {
		*driver.DocumentMeta
		*ResponseStruct

		Old *unmarshalInto `json:"old,omitempty"`
		New *unmarshalInto `json:"new,omitempty"`
	}
}

func (c *collectionDocumentCreateResponseReader) Close() error {
	return c.array.Close()
}

func (c *collectionDocumentCreateResponseReader) Read() (CollectionDocumentCreateResponse, bool, error) {
	if !c.array.More() {
		return CollectionDocumentCreateResponse{}, false, nil
	}

	var meta CollectionDocumentCreateResponse

	if c.options != nil {
		meta.Old = c.options.OldObject
		meta.New = c.options.NewObject
	}

	c.response.DocumentMeta = &meta.DocumentMeta
	c.response.ResponseStruct = &meta.ResponseStruct

	if err := c.array.Unmarshal(&c.response); err != nil {
		return CollectionDocumentCreateResponse{}, false, err
	}

	return meta, true, nil
}

func newCollectionDocumentReadResponseReader(array connection.Array, options *CollectionDocumentReadOptions) *collectionDocumentReadResponseReader {
	c := &collectionDocumentReadResponseReader{array: array, options: options}

	return c
}

var _ CollectionDocumentReadResponseReader = &collectionDocumentReadResponseReader{}

type collectionDocumentReadResponseReader struct {
	array    connection.Array
	options  *CollectionDocumentReadOptions
	response struct {
		*driver.DocumentMeta
		*unmarshalInto
	}
}

func (c *collectionDocumentReadResponseReader) Close() error {
	return c.array.Close()
}

func (c *collectionDocumentReadResponseReader) Read(i interface{}) (CollectionDocumentReadResponse, bool, error) {
	if !c.array.More() {
		return CollectionDocumentReadResponse{}, false, nil
	}

	var meta CollectionDocumentReadResponse

	c.response.DocumentMeta = &meta.DocumentMeta
	c.response.unmarshalInto = newUnmarshalInto(i)

	if err := c.array.Unmarshal(&c.response); err != nil {
		return CollectionDocumentReadResponse{}, false, err
	}

	return meta, true, nil
}
