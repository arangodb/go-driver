//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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
	"net/http"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newCollectionDocuments(collection *collection) *collectionDocuments {
	d := &collectionDocuments{collection: collection}

	d.collectionDocumentUpdate = newCollectionDocumentUpdate(d.collection)
	d.collectionDocumentReplace = newCollectionDocumentReplace(d.collection)
	d.collectionDocumentRead = newCollectionDocumentRead(d.collection)
	d.collectionDocumentCreate = newCollectionDocumentCreate(d.collection)
	d.collectionDocumentDelete = newCollectionDocumentDelete(d.collection)

	return d
}

var (
	_ CollectionDocuments = &collectionDocuments{}
)

type collectionDocuments struct {
	collection *collection

	*collectionDocumentUpdate
	*collectionDocumentReplace
	*collectionDocumentRead
	*collectionDocumentCreate
	*collectionDocumentDelete
}

func (c collectionDocuments) DocumentExists(ctx context.Context, key string) (bool, error) {
	url := c.collection.url("document", key)

	resp, err := connection.CallHead(ctx, c.collection.connection(), url, nil, c.collection.withModifiers()...)

	if err != nil {
		if shared.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}
