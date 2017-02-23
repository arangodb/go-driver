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
