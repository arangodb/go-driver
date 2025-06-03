//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

// TestRemoveVertex creates a document, remove it and then checks the removal has succeeded.
func TestRemoveVertex(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "remove_vertex_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "users", t)

	doc := UserDoc{
		Name: "Jones",
		Age:  65,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	if _, err := vc.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Should not longer exist
	var readDoc UserDoc
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}

	// Document must not exist now
	if found, err := vc.DocumentExists(nil, meta.Key); err != nil {
		t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
	} else if found {
		t.Errorf("DocumentExists returned true for '%s', expected false", meta.Key)
	}
}

// TestRemoveVertexReturnOld creates a document, removes it checks the ReturnOld value.
func TestRemoveVertexReturnOld(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2365
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "remove_vertex_returnOld_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "Testing 101",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	require.NoError(t, err)

	var old Book
	ctx = driver.WithReturnOld(ctx, &old)
	_, err = vc.RemoveDocument(ctx, meta.Key)
	require.NoError(t, err)

	// Check an old document
	require.Equal(t, doc, old)
}

// TestRemoveVertexSilent creates a document, removes it with Silent() and then checks the meta is indeed empty.
func TestRemoveVertexSilent(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "remove_vertex_silent_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "books", t)

	doc := Book{
		Title: "Shhh...",
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	ctx = driver.WithSilent(ctx)
	if rmeta, err := vc.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	} else if rmeta.Key != "" {
		t.Errorf("Expected empty meta, got %v", rmeta)
	}
	// Should not longer exist
	var readDoc Book
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveVertexRevision creates a document, removes it with an incorrect revision.
func TestRemoveVertexRevision(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "remove_vertex_revision_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "persons", t)

	doc := UserDoc{
		Name: "Dude",
		Age:  12,
	}
	meta, err := vc.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}

	// Replace the document to get another revision
	replacement := Book{
		Title: "The only way is change",
	}
	meta2, err := vc.ReplaceDocument(ctx, meta.Key, replacement)
	if err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}

	// Try to remove document with initial revision (must fail)
	initialRevCtx := driver.WithRevision(ctx, meta.Rev)
	if _, err := vc.RemoveDocument(initialRevCtx, meta.Key); !driver.IsPreconditionFailed(err) {
		t.Fatalf("Expected PreconditionFailedError, got %s", describe(err))
	}

	// Try to remove document with correct revision (must succeed)
	replacedRevCtx := driver.WithRevision(ctx, meta2.Rev)
	if _, err := vc.RemoveDocument(replacedRevCtx, meta.Key); err != nil {
		t.Fatalf("Expected success, got %s", describe(err))
	}

	// Should not longer exist
	var readDoc Book
	if _, err := vc.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveVertexKeyEmpty removes a document it with an empty key.
func TestRemoveVertexKeyEmpty(t *testing.T) {
	var ctx context.Context
	c := createClient(t, nil)
	db := ensureDatabase(ctx, c, "vertex_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	g := ensureGraph(ctx, db, "remove_vertex_nil_test", nil, t)
	vc := ensureVertexCollection(ctx, g, "hobby", t)

	if _, err := vc.RemoveDocument(nil, ""); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
