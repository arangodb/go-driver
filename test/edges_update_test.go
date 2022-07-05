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
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestUpdateEdges creates documents, updates them and then checks the updates have succeeded.
func TestUpdateEdges(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"type": "Updated1",
		},
		{
			"type": "Updated2",
		},
	}
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc RelationEdge
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		doc := docs[i]
		doc.Type = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(doc, readDoc) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, readDoc)
		}
	}
}

// TestUpdateEdgesReturnOld creates documents, updates them checks the ReturnOld values.
func TestUpdateEdgesReturnOld(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_returnOld_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"type": "Updated1",
		},
		{
			"type": "Updated2",
		},
	}
	oldDocs := make([]RelationEdge, len(docs))
	ctx = driver.WithReturnOld(ctx, oldDocs)
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Check old documents
	for i, doc := range docs {
		if !reflect.DeepEqual(doc, oldDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, doc, oldDocs[i])
		}
	}
}

// TestUpdateEdgesReturnNew creates documents, updates them checks the ReturnNew values.
func TestUpdateEdgesReturnNew(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.4", t) // See https://github.com/arangodb/arangodb/issues/2363
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_returnOld_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RelationEdge{
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			Type: "friend",
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"type": "Updated1",
		},
		{
			"type": "Updated2",
		},
	}
	newDocs := make([]RelationEdge, len(docs))
	ctx = driver.WithReturnNew(ctx, newDocs)
	if _, _, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Check new documents
	for i, doc := range docs {
		expected := doc
		expected.Type = fmt.Sprintf("Updated%d", i+1)
		if !reflect.DeepEqual(expected, newDocs[i]) {
			t.Errorf("Got wrong document %d. Expected %+v, got %+v", i, expected, newDocs[i])
		}
	}
}

// TestUpdateEdgesKeepNullTrue creates documents, updates them with KeepNull(true) and then checks the updates have succeeded.
func TestUpdateEdgesKeepNullTrue(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	conn := c.Connection()
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_keepNullTrue_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []AccountEdge{
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			User: &UserDoc{
				"Greata",
				77,
			},
		},
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			User: &UserDoc{
				"Mathilda",
				45,
			},
		},
	}

	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Update documents
	updates := []map[string]interface{}{
		{
			"to":   from.ID.String(),
			"user": nil,
		},
		{
			"from": to.ID.String(),
			"user": nil,
		},
	}
	if _, _, err := ec.UpdateDocuments(driver.WithKeepNull(ctx, true), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		var readDoc map[string]interface{}
		var rawResponse []byte
		ctx = driver.WithRawResponse(ctx, &rawResponse)
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document %d '%s': %s", i, meta.Key, describe(err))
		}
		// We parse to this type of map, since unmarshalling nil values to a map of type map[string]interface{}
		// will cause the entry to be deleted.
		var jsonMap map[string]*driver.RawObject
		if err := conn.Unmarshal(rawResponse, &jsonMap); err != nil {
			t.Fatalf("Failed to parse raw response: %s", describe(err))
		}
		// Get "edge" field and unmarshal it
		if raw, found := jsonMap["edge"]; !found {
			t.Errorf("Expected edge to be found but got not found")
		} else {
			jsonMap = nil
			if err := conn.Unmarshal(*raw, &jsonMap); err != nil {
				t.Fatalf("Failed to parse raw edge object: %s", describe(err))
			}
			if raw, found := jsonMap["user"]; !found {
				t.Errorf("Expected user to be found but got not found")
			} else if raw != nil {
				t.Errorf("Expected user to be found and nil, got %s", string(*raw))
			}
		}
	}
}

// TestUpdateEdgesKeepNullFalse creates documents, updates them with KeepNull(false) and then checks the updates have succeeded.
func TestUpdateEdgesKeepNullFalse(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_keepNullFalse_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []AccountEdge{
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			User: &UserDoc{
				"Piere",
				77,
			},
		},
		{
			From: from.ID.String(),
			To:   to.ID.String(),
			User: &UserDoc{
				"Joan",
				45,
			},
		},
	}

	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}
	// Update document
	updates := []map[string]interface{}{
		{
			"to":   from.ID.String(),
			"user": nil,
		},
		{
			"from": to.ID.String(),
			"user": nil,
		},
	}
	if _, _, err := ec.UpdateDocuments(driver.WithKeepNull(ctx, false), metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	}
	// Read updated documents
	for i, meta := range metas {
		readDoc := docs[i]
		if _, err := ec.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if readDoc.User == nil {
			t.Errorf("Expected user to be untouched, got %v", readDoc.User)
		}
	}
}

// TestUpdateEdgesSilent creates documents, updates them with Silent() and then checks the metas are indeed empty.
func TestUpdateEdgesSilent(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_silent_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 7,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 88,
		},
	}
	metas, _, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new documents: %s", describe(err))
	}
	// Update documents
	updates := []map[string]interface{}{
		{
			"distance": 61,
		},
		{
			"distance": 16,
		},
	}
	ctx = driver.WithSilent(ctx)
	if metas, errs, err := ec.UpdateDocuments(ctx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	} else if strings.Join(metas.Keys(), "") != "" {
		t.Errorf("Expected empty meta, got %v", metas)
	}
}

// TestUpdateEdgesRevision creates documents, updates them with a specific (correct) revisions.
// Then it attempts an update with an incorrect revisions which must fail.
func TestUpdateEdgesRevision(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_revision_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)
	male := ensureCollection(ctx, db, prefix+"male", nil, t)
	female := ensureCollection(ctx, db, prefix+"female", nil, t)
	from := createDocument(ctx, male, map[string]interface{}{"name": "Jan"}, t)
	to := createDocument(ctx, female, map[string]interface{}{"name": "Alice"}, t)

	docs := []RouteEdge{
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 7,
		},
		{
			From:     from.ID.String(),
			To:       to.ID.String(),
			Distance: 88,
		},
	}
	metas, errs, err := ec.CreateDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to create new document: %s", describe(err))
	} else if len(metas) != len(docs) {
		t.Fatalf("Expected %d metas, got %d", len(docs), len(metas))
	} else if err := errs.FirstNonNil(); err != nil {
		t.Fatalf("Expected no errors, got first: %s", describe(err))
	}

	// Update documents with correct revisions
	updates := []map[string]interface{}{
		{
			"distance": 34,
		},
		{
			"distance": 77,
		},
	}
	initialRevCtx := driver.WithRevisions(ctx, metas.Revs())
	var updatedRevCtx context.Context
	if metas2, _, err := ec.UpdateDocuments(initialRevCtx, metas.Keys(), updates); err != nil {
		t.Fatalf("Failed to update documents: %s", describe(err))
	} else {
		updatedRevCtx = driver.WithRevisions(ctx, metas2.Revs())
		if strings.Join(metas2.Revs(), ",") == strings.Join(metas.Revs(), ",") {
			t.Errorf("Expected revision to change, got initial revision '%s', updated revision '%s'", strings.Join(metas.Revs(), ","), strings.Join(metas2.Revs(), ","))
		}
	}

	// Update documents with incorrect revisions
	updates[0]["distance"] = 35
	var rawResponse []byte
	if _, errs, err := ec.UpdateDocuments(driver.WithRawResponse(initialRevCtx, &rawResponse), metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	} else {
		for _, err := range errs {
			if !driver.IsPreconditionFailed(err) {
				t.Errorf("Expected PreconditionFailedError, got %s (resp: %s", describe(err), string(rawResponse))
			}
		}
	}

	// Update documents once more with correct revisions
	updates[0]["distance"] = 36
	if _, _, err := ec.UpdateDocuments(updatedRevCtx, metas.Keys(), updates); err != nil {
		t.Errorf("Expected success, got %s", describe(err))
	}
}

// TestUpdateEdgesKeyEmpty updates documents with an empty key.
func TestUpdateEdgesKeyEmpty(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_keyEmpty_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)

	// Update document
	updates := []map[string]interface{}{
		{
			"name": "Updated",
		},
	}
	if _, _, err := ec.UpdateDocuments(nil, []string{""}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateEdgesUpdateNil updates documents it with a nil update.
func TestUpdateEdgesUpdateNil(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_updateNil_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)

	if _, _, err := ec.UpdateDocuments(nil, []string{"validKey"}, nil); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}

// TestUpdateEdgesUpdateLenDiff updates documents with a different number of updates, keys.
func TestUpdateEdgesUpdateLenDiff(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "edges_test", nil, t)
	prefix := "update_edges_updateLenDiff_"
	g := ensureGraph(ctx, db, prefix+"graph", nil, t)
	ec := ensureEdgeCollection(ctx, g, prefix+"relation", []string{prefix + "male", prefix + "female"}, []string{prefix + "male", prefix + "female"}, t)

	updates := []map[string]interface{}{
		{
			"name": "name1",
		},
		{
			"name": "name2",
		},
	}
	if _, _, err := ec.UpdateDocuments(nil, []string{"only1"}, updates); !driver.IsInvalidArgument(err) {
		t.Errorf("Expected InvalidArgumentError, got %s", describe(err))
	}
}
