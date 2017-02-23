//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package test

import (
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestCreateDocument creates a document and then checks that it exists.
func TestCreateDocument(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Document must exists now
	var readDoc UserDoc
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}

// TestCreateDocumentNil creates a document with a nil document.
func TestCreateDocumentNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.CreateDocument(nil, nil); !driver.IsInvalidArgument(err) {
		t.Fatalf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
