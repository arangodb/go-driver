//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

type VertexCollection interface {
	// Name returns the name of the vertex collection
	Name() string

	// GetVertex Gets a vertex from the given collection.
	// To get _key and _rev values, embed the DocumentMeta struct in your result struct.
	GetVertex(ctx context.Context, key string, result interface{}, opts *GetVertexOptions) error

	// CreateVertex Adds a vertex to the given collection.
	// To get _key and _rev values, embed the DocumentMeta struct in your result struct and pass to VertexCreateResponse.New.
	CreateVertex(ctx context.Context, vertex interface{}, opts *CreateVertexOptions) (VertexCreateResponse, error)

	// UpdateVertex Updates the data of the specific vertex in the collection.
	UpdateVertex(ctx context.Context, key string, newValue interface{}, opts *VertexUpdateOptions) (VertexUpdateResponse, error)

	// ReplaceVertex Replaces the data of a vertex in the collection.
	ReplaceVertex(ctx context.Context, key string, newValue interface{}, opts *VertexReplaceOptions) (VertexReplaceResponse, error)

	// DeleteVertex Removes a vertex from the collection.
	DeleteVertex(ctx context.Context, key string, opts *DeleteVertexOptions) (VertexDeleteResponse, error)
}

type GetVertexOptions struct {
	// Must contain a revision. If this is set, a document is only returned if it has exactly this revision.
	// Also see if-match header as an alternative to this.
	Rev string `json:"rev,omitempty"`

	// If the “If-Match” header is given, then it must contain exactly one ETag (_rev).
	// The document is returned, if it has the same revision as the given ETag
	IfMatch string

	// If the “If-None-Match” header is given, then it must contain exactly one ETag (_rev).
	// The document is returned, if it has a different revision than the given ETag
	IfNoneMatch string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

type CreateVertexOptions struct {
	// Define if the request should wait until synced to disk.
	WaitForSync *bool `json:"waitForSync,omitempty"`

	// Define if the response should contain the complete new version of the document.
	NewObject interface{}

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

type VertexCreateResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	New                   interface{}
}

type VertexUpdateOptions struct {
	// Define if the request should wait until synced to disk.
	WaitForSync *bool

	// Define if a presentation of the new document should be returned within the response object.
	NewObject interface{}

	// Define if a presentation of the deleted document should be returned within the response object.
	OldObject interface{}

	// Define if values set to null should be stored. By default (true), the given documents attribute(s)
	// are set to null. If this parameter is set to false, top-level attribute and sub-attributes with a null value
	// in the request are removed from the document (but not attributes of objects that are nested inside of arrays).
	KeepNull *bool

	// Conditionally update a vertex based on a target revision id
	// If the “If-Match” header is given, then it must contain exactly one ETag (_rev).
	IfMatch string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

type VertexUpdateResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	Old, New              interface{}
}

type VertexReplaceOptions struct {
	// Define if the request should wait until synced to disk.
	WaitForSync *bool

	// Define if a presentation of the new document should be returned within the response object.
	NewObject interface{}

	// Define if a presentation of the deleted document should be returned within the response object.
	OldObject interface{}

	// Define if values set to null should be stored. By default (true), the given documents attribute(s)
	// are set to null. If this parameter is set to false, top-level attribute and sub-attributes with a null value
	// in the request are removed from the document (but not attributes of objects that are nested inside of arrays).
	KeepNull *bool

	// Conditionally replace a vertex based on a target revision id
	// If the “If-Match” header is given, then it must contain exactly one ETag (_rev).
	IfMatch string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

type VertexReplaceResponse struct {
	DocumentMeta
	shared.ResponseStruct `json:",inline"`
	Old, New              interface{}
}

type DeleteVertexOptions struct {
	// Define if the request should wait until synced to disk.
	WaitForSync *bool `json:"waitForSync,omitempty"`

	// Define if a presentation of the deleted document should be returned within the response object.
	OldObject interface{}

	// Conditionally delete a vertex based on a target revision id
	// If the “If-Match” header is given, then it must contain exactly one ETag (_rev).
	IfMatch string

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string
}

type VertexDeleteResponse struct {
	shared.ResponseStruct `json:",inline"`
	Old                   interface{}
}
