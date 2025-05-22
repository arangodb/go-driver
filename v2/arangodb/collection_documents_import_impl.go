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
	"fmt"
	"net/http"
	"reflect"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/pkg/errors"
)

func newCollectionDocumentImport(collection *collection) *collectionDocumentImport {
	return &collectionDocumentImport{
		collection: collection,
	}
}

var _ CollectionDocumentImport = &collectionDocumentImport{}

type collectionDocumentImport struct {
	collection *collection
}

func (c collectionDocumentImport) ImportDocuments(ctx context.Context, documents string, documentsType CollectionDocumentImportDocumentType) (CollectionDocumentImportResponse, error) {
	return c.ImportDocumentsWithOptions(ctx, documents, documentsType, nil)
}

func (c collectionDocumentImport) ImportDocumentsWithOptions(ctx context.Context, documents string, documentsType CollectionDocumentImportDocumentType, opts *CollectionDocumentImportOptions) (CollectionDocumentImportResponse, error) {
	documentsVal := reflect.ValueOf(documents)
	switch documentsVal.Kind() {
	case reflect.String:
		// OK
	default:
		return CollectionDocumentImportResponse{}, errors.WithStack(shared.InvalidArgumentError{Message: fmt.Sprintf("documents data The body must either be a JSON-encoded array of objects or a string with multiple JSON objects separated by newlines got %s", documentsVal.Kind())})
	}

	url := c.collection.db.url("_api/import")
	// print(url)

	var response struct {
		shared.ResponseStruct            `json:",inline"`
		CollectionDocumentImportResponse `json:",inline"`
	}

	request := &CollectionDocumentImportRequest{
		Collection: &c.collection.name,
		Type:       &documentsType,
	}
	if opts != nil {
		request.CollectionDocumentImportOptions = *opts
	}
	resp, err := connection.CallPost(
		ctx, c.collection.connection(), url, &response,
		[]byte(documents), c.collection.withModifiers(request.modifyRequest)...)

	if err != nil {
		return CollectionDocumentImportResponse{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return response.CollectionDocumentImportResponse, nil
	default:
		return CollectionDocumentImportResponse{}, response.AsArangoErrorWithCode(code)
	}
}
