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
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestImportDocumentsWithKeys imports documents and then checks that it exists.
func TestImportDocumentsWithKeys(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_withKeys_test", nil, t)
	docs := []UserDocWithKey{
		UserDocWithKey{
			"jan",
			"Jan",
			40,
		},
		UserDocWithKey{
			"foo",
			"Foo",
			41,
		},
		UserDocWithKey{
			"frank",
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
	}
}

// TestImportDocumentsWithoutKeys imports documents and then checks that it exists.
func TestImportDocumentsWithoutKeys(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "import_withoutKeys_test", nil, t)
	docs := []UserDoc{
		UserDoc{
			"Jan",
			40,
		},
		UserDoc{
			"Foo",
			41,
		},
		UserDoc{
			"Frank",
			42,
		},
	}

	var raw []byte
	ctx := driver.WithRawResponse(nil, &raw)
	stats, err := col.ImportDocuments(ctx, docs, nil)
	if err != nil {
		t.Fatalf("Failed to import documents: %s", describe(err))
	} else {
		if stats.Created != int64(len(docs)) {
			t.Errorf("Expected %d created documents, got %d (json %s)", len(docs), stats.Created, string(raw))
		}
		if stats.Errors != 0 {
			t.Errorf("Expected %d error documents, got %d (json %s)", 0, stats.Errors, string(raw))
		}
	}
}
