package test

import (
	"context"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestReplaceDocument creates a document, remove it and then checks the removal has succeeded.
func TestRemoveDocument(t *testing.T) {
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
	if _, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentReturnOld creates a document, removes it checks the ReturnOld value.
func TestRemoveDocumentReturnOld(t *testing.T) {
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
	var old UserDoc
	ctx = driver.WithReturnOld(ctx, &old)
	if _, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	}
	// Check old document
	if !reflect.DeepEqual(doc, old) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, old)
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentSilent creates a document, removes it with Silent() and then checks the meta is indeed empty.
func TestRemoveDocumentSilent(t *testing.T) {
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
	ctx = driver.WithSilent(ctx)
	if rmeta, err := col.RemoveDocument(ctx, meta.Key); err != nil {
		t.Fatalf("Failed to remove document '%s': %s", meta.Key, describe(err))
	} else if rmeta.Key != "" {
		t.Errorf("Expected empty meta, got %v", rmeta)
	}
	// Should not longer exist
	var readDoc Account
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); !driver.IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got  %s", describe(err))
	}
}

// TestRemoveDocumentKeyEmpty removes a document it with an empty key.
func TestRemoveDocumentKeyEmpty(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	if _, err := col.RemoveDocument(nil, ""); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
