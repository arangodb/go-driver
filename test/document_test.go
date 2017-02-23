package test

import (
	"reflect"
	"testing"
)

type UserDoc struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// TestCreateDatabase creates a document and then checks that it exists.
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

// TestUpdateDocument1 creates a document, updates it and then checks the update has succeeded.
func TestUpdateDocument1(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)
	doc := UserDoc{
		"Piere",
		23,
	}
	meta, err := col.CreateDocument(nil, doc)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	}
	// Update document
	update := map[string]interface{}{
		"name": "Updated",
	}
	if _, err := col.UpdateDocument(nil, meta.Key, update); err != nil {
		t.Fatalf("Failed to update document '%s': %s", meta.Key, describe(err))
	}
	// Read updated document
	var readDoc UserDoc
	if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
		t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
	}
	doc.Name = "Updated"
	if !reflect.DeepEqual(doc, readDoc) {
		t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
	}
}
