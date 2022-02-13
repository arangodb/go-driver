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
// Author Lars Maier
//

package test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestReadDocumentWithIfMatch creates a document and reads it with a non-matching revision.
func TestReadDocumentWithIfMatch(t *testing.T) {
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(nil, c, "document_read_test", nil, t)
	col := ensureCollection(nil, db, "document_read_test", nil, t)
	doc := UserDoc{
		"Jan",
		40,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	ctx := context.Background()
	ctx = driver.WithRevision(ctx, meta.Rev)

	meta2, err := col.ReadDocument(ctx, meta.Key, &doc)
	if err != nil {
		t.Fatalf("Failed to read document: %s", describe(err))
	}
	if meta2.Key != meta.Key || meta2.Rev != meta.Rev || meta2.ID != meta.ID {
		t.Error("Read wrong meta data.")
	}

	var resp driver.Response
	ctx2 := context.Background()
	ctx2 = driver.WithRevision(ctx2, "nonsense")
	ctx2 = driver.WithResponse(ctx2, &resp)
	_, err = col.ReadDocument(ctx2, meta.Key, &doc)
	if err == nil {
		t.Error("Reading with wrong revision did not fail")
	}
	if resp.StatusCode() != 412 {
		t.Errorf("Expected status code 412, found %d", resp.StatusCode())
	}
}
