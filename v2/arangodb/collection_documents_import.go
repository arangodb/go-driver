//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/connection"
)

const (
	QueryFromPrefix  = "fromPrefix"
	QueryToPrefix    = "toPrefix"
	QueryComplete    = "complete"
	QueryOnDuplicate = "onDuplicate"
)

// CollectionDocumentDelete removes document(s) with given key(s) from the collection
// https://docs.arangodb.com/stable/develop/http-api/documents/#remove-a-document
type CollectionDocumentImport interface {

	// ImportDocuments imports one or more documents into the collection. // TODO FIX
	// The document data is loaded from the given documents argument, statistics are returned.
	// The documents argument can be one of the following:
	// - An array of structs: All structs will be imported as individual documents.
	// - An array of maps: All maps will be imported as individual documents.
	// To wait until all documents have been synced to disk, prepare a context with `WithWaitForSync`.
	// To return details about documents that could not be imported, prepare a context with `WithImportDetails`.
	ImportDocuments(ctx context.Context, documents string, documentsType CollectionDocumentImportDocumentType) (CollectionDocumentImportResponse, error)
	ImportDocumentsWithOptions(ctx context.Context, documents string, documentsType CollectionDocumentImportDocumentType, options *CollectionDocumentImportOptions) (CollectionDocumentImportResponse, error)
}

type CollectionDocumentImportResponse struct {
	CollectionDocumentImportStatistics `json:",inline"`
}

// ImportDocumentRequest holds Query parameters for /import.
type CollectionDocumentImportRequest struct {
	CollectionDocumentImportOptions `json:",inline"`
	Collection                      *string                               `json:"collection,inline"`
	Type                            *CollectionDocumentImportDocumentType `json:"type,inline"`
}

// ImportDocumentOptions holds optional options that control the import document process.
type CollectionDocumentImportOptions struct {
	// FromPrefix is an optional prefix for the values in _from attributes. If specified, the value is automatically
	// prepended to each _from input value. This allows specifying just the keys for _from.
	FromPrefix *string `json:"fromPrefix,omitempty"`
	// ToPrefix is an optional prefix for the values in _to attributes. If specified, the value is automatically
	// prepended to each _to input value. This allows specifying just the keys for _to.
	ToPrefix *string `json:"toPrefix,omitempty"`
	// Overwrite is a flag that if set, then all data in the collection will be removed prior to the import.
	// Note that any existing index definitions will be preseved.
	Overwrite *bool `json:"overwrite,omitempty"`
	// OnDuplicate controls what action is carried out in case of a unique key constraint violation.
	// Possible values are:
	// - ImportOnDuplicateError
	// - ImportOnDuplicateUpdate
	// - ImportOnDuplicateReplace
	// - ImportOnDuplicateIgnore
	OnDuplicate *CollectionDocumentImportOnDuplicate `json:"onDuplicate,omitempty"`
	// Complete is a flag that if set, will make the whole import fail if any error occurs.
	// Otherwise the import will continue even if some documents cannot be imported.
	Complete *bool `json:"complete,omitempty"`

	// Wait until the deletion operation has been synced to disk.
	WithWaitForSync *bool
}

type CollectionDocumentImportDocumentType string

const (
	// ImportDocumentTypeDocuments
	//   Each line is expected to be one JSON object.
	//   example :
	//	  {"_key":"john","name":"John Smith","age":35}
	//	  {"_key":"katie","name":"Katie Foster","age":28}
	ImportDocumentTypeDocuments CollectionDocumentImportDocumentType = CollectionDocumentImportDocumentType("documents")

	// ImportDocumentTypeArray
	//   The request body is expected to be a JSON array of objects.
	//   example :
	//	  [
	//      {"_key":"john","name":"John Smith","age":35},
	//      {"_key":"katie","name":"Katie Foster","age":28}
	//    ]
	ImportDocumentTypeArray CollectionDocumentImportDocumentType = CollectionDocumentImportDocumentType("array")

	// ImportDocumentTypeAuto
	//   Automatically determines the type either documents(ImportDocumentTypeDocumentsError) or array(ImportDocumentTypeArrayError)
	ImportDocumentTypeAuto CollectionDocumentImportDocumentType = CollectionDocumentImportDocumentType("auto")

	// ImportDocumentTypeTabular
	//   The first line is an array of strings that defines the attribute keys. The subsequent lines are arrays with the attribute values.
	//   The keys and values are matched by the order of the array elements.
	//   example:
	//     ["_key","name","age"]
	//     ["john","John Smith",35]
	//     ["katie","Katie Foster",28]
	ImportDocumentTypeTabular CollectionDocumentImportDocumentType = CollectionDocumentImportDocumentType("")
)

type CollectionDocumentImportOnDuplicate string

const (
	// ImportOnDuplicateError will not import the current document because of the unique key constraint violation.
	// This is the default setting.
	ImportOnDuplicateError CollectionDocumentImportOnDuplicate = CollectionDocumentImportOnDuplicate("error")
	// ImportOnDuplicateUpdate will update an existing document in the database with the data specified in the request.
	// Attributes of the existing document that are not present in the request will be preserved.
	ImportOnDuplicateUpdate CollectionDocumentImportOnDuplicate = CollectionDocumentImportOnDuplicate("update")
	// ImportOnDuplicateReplace will replace an existing document in the database with the data specified in the request.
	ImportOnDuplicateReplace CollectionDocumentImportOnDuplicate = CollectionDocumentImportOnDuplicate("replace")
	// ImportOnDuplicateIgnore will not update an existing document and simply ignore the error caused by a unique key constraint violation.
	ImportOnDuplicateIgnore CollectionDocumentImportOnDuplicate = CollectionDocumentImportOnDuplicate("ignore")
)

// CollectionDocumentImportResponse holds statistics of an import action.
type CollectionDocumentImportStatistics struct {
	// Created holds the number of documents imported.
	Created int64 `json:"created,omitempty"`
	// Errors holds the number of documents that were not imported due to an error.
	Errors int64 `json:"errors,omitempty"`
	// Empty holds the number of empty lines found in the input (will only contain a value greater zero for types documents or auto).
	Empty int64 `json:"empty,omitempty"`
	// Updated holds the number of updated/replaced documents (in case onDuplicate was set to either update or replace).
	Updated int64 `json:"updated,omitempty"`
	// Ignored holds the number of failed but ignored insert operations (in case onDuplicate was set to ignore).
	Ignored int64 `json:"ignored,omitempty"`
	// if query parameter details is set to true, the result will contain a details attribute which is an array
	// with more detailed information about which documents could not be inserted.
	Details []string
}

func (c *CollectionDocumentImportOptions) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.FromPrefix != nil {
		r.AddQuery(QueryFromPrefix, *c.FromPrefix)
	}

	if c.ToPrefix != nil {
		r.AddQuery(QueryToPrefix, *c.ToPrefix)
	}

	if c.Overwrite != nil {
		r.AddQuery(QueryOverwrite, boolToString(*c.Overwrite))
	}

	if c.OnDuplicate != nil {
		r.AddQuery(QueryOnDuplicate, string(*c.OnDuplicate))
	}

	if c.Complete != nil {
		r.AddQuery(QueryComplete, boolToString(*c.Complete))
	}

	if c.WithWaitForSync != nil {
		r.AddQuery(QueryWaitForSync, boolToString(*c.WithWaitForSync))
	}

	return nil
}

func (c *CollectionDocumentImportRequest) modifyRequest(r connection.Request) error {
	if c == nil {
		return nil
	}

	if c.Collection != nil {
		r.AddQuery(QueryCollection, *c.Collection)
	}

	if c.Type != nil && string(*c.Type) != "" {
		r.AddQuery(QueryType, string(*c.Type))
	}

	r.AddHeader(connection.ContentType, "text/plain")
	r.AddHeader("Accept", "text/plain")

	c.CollectionDocumentImportOptions.modifyRequest(r)

	return nil
}
