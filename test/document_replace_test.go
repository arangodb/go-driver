package test

import (
	"context"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestReplaceDocument creates a document, replaces it and then checks the replacement has succeeded.
func TestReplaceDocument(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Piere",
		23,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replacement doc
	replacement := Account{
		ID:   "foo",
		User: &UserDoc{},
	}
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Read replaces document
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	if !reflect.DeepEqual(replacement, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", replacement, readDoc)
	}
}

// TestReplaceDocumentReturnOld creates a document, replaces it checks the ReturnOld value.
func TestReplaceDocumentReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Tim",
		27,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Replace document
	replacement := Book{
		Title: "Golang 1.8",
	}
	var old UserDoc
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
}

// TestReplaceDocumentReturnNew creates a document, replaces it checks the ReturnNew value.
func TestReplaceDocumentReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Tim",
		27,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := Book{
		Title: "Golang 1.8",
	}
	var newDoc Book
	ctx = driver.WithReturnNew(ctx, &newDoc)
	if _, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to replace document '%s': %s", meta.Key, describe(err))
	}
	// Check new document
	expected := replacement
	if !reflect.DeepEqual(expected, newDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", expected, newDoc)
	}
}

// TestReplaceDocumentSilent creates a document, replaces it with Silent() and then checks the meta is indeed empty.
func TestReplaceDocumentSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "document_test", nil, t)
	col := ensureCollection(ctx, db, "document_test", nil, t)
	doc := UserDoc{
		"Angela",
		91,
	}
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	replacement := Book{
		Title: "Jungle book",
	}
	ctx = driver.WithSilent(ctx)
	if meta, err := col.ReplaceDocument(ctx, meta.Key, replacement); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	} else if meta.Key != "" {
		t.Errorf("Expected empty meta, got %v", meta)
	}
}

// TestReplaceDocumentKeyEmpty replaces a document it with an empty key.
func TestReplaceDocumentKeyEmpty(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	// Update document
	replacement := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := col.ReplaceDocument(nil, "", replacement); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestReplaceDocumentUpdateNil replaces a document it with a nil update.
func TestReplaceDocumentUpdateNil(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.ReplaceDocument(nil, "validKey", nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
