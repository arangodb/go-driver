//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

func TestDatabaseTransaction(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.2", t)
	// for disabling v8 tests
	skipAboveVersion(c, "3.12.6-1", t)
	db := ensureDatabase(nil, c, "transaction_test", nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()

	const colName = "books"
	ensureCollection(context.Background(), db, colName, nil, t)

	txOptions := &driver.TransactionOptions{
		ReadCollections:      []string{colName},
		WriteCollections:     []string{colName},
		ExclusiveCollections: []string{colName},
	}
	testCases := []struct {
		name           string
		action         string
		options        *driver.TransactionOptions
		expectResult   interface{}
		expectErrorStr string
	}{
		{"ReturnValue", "function () { return 'worked!'; }", txOptions, "worked!", ""},
		{"ReturnError", "function () { error error; }", txOptions, nil, "Uncaught SyntaxError: Unexpected identifier"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := db.Transaction(nil, testCase.action, testCase.options)
			if !reflect.DeepEqual(testCase.expectResult, result) {
				t.Errorf("expected result %v, got %v", testCase.expectResult, result)
			}
			if testCase.expectErrorStr != "" {
				if strings.Index(err.Error(), testCase.expectErrorStr) < 0 {
					t.Errorf("expected error to contain '%v', got '%v'", testCase.expectErrorStr, err.Error())
				}
			}
		})
	}
}

func insertDocument(ctx context.Context, col driver.Collection, t *testing.T) driver.DocumentMeta {
	doc := struct {
		Name string `json:"name,omitempty"`
	}{
		Name: "Hello World",
	}
	if meta, err := col.CreateDocument(ctx, &doc); err != nil {
		t.Fatalf("Failed to create document: %s", describe(err))
	} else {
		return meta
	}
	return driver.DocumentMeta{}
}

func documentExists(ctx context.Context, col driver.Collection, key string, exists bool, t *testing.T) {
	if found, err := col.DocumentExists(ctx, key); err != nil {
		t.Fatalf("DocumentExists failed: %s", describe(err))
	} else {
		if exists != found {
			t.Errorf("Document status not as expected: expected: %t, actual: %t", exists, found)
		}
	}
}

func TestTransactionCommit(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	colname := "trx_test_col"
	ctx := context.Background()
	db := ensureDatabase(ctx, c, "trx_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, colname, nil, t)

	trxid, err := db.BeginTransaction(ctx, driver.TransactionCollections{Exclusive: []string{colname}}, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %s", describe(err))
	}

	tctx := driver.WithTransactionID(ctx, trxid)
	meta1 := insertDocument(tctx, col, t)

	// document should not exist without transaction
	documentExists(ctx, col, meta1.Key, false, t)

	// document should exist with transaction
	documentExists(tctx, col, meta1.Key, true, t)

	trxidRead, exist := driver.HasTransactionID(tctx)
	require.True(t, exist, "Transaction ID should be set")
	require.Equal(t, trxid, trxidRead, "Transaction ID should be the same")

	// Now commit the transaction
	if err := db.CommitTransaction(ctx, trxid, nil); err != nil {
		t.Fatalf("Failed to commit transaction: %s", describe(err))
	}

	// document should exist
	documentExists(ctx, col, meta1.Key, true, t)
}

func TestTransactionAbort(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.5", t)
	colname := "trx_test_col_abort"
	ctx := context.Background()
	db := ensureDatabase(ctx, c, "trx_test", nil, t)
	defer func() {
		err := db.Remove(ctx)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(ctx, db, colname, nil, t)

	trxid, err := db.BeginTransaction(ctx, driver.TransactionCollections{Exclusive: []string{colname}}, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %s", describe(err))
	}

	tctx := driver.WithTransactionID(ctx, trxid)
	meta1 := insertDocument(tctx, col, t)

	// document should not exist without transaction
	documentExists(ctx, col, meta1.Key, false, t)

	// document should exist with transaction
	documentExists(tctx, col, meta1.Key, true, t)

	// Now abort the transaction
	if err := db.AbortTransaction(ctx, trxid, nil); err != nil {
		t.Fatalf("Failed to abort transaction: %s", describe(err))
	}

	// document should exist
	documentExists(ctx, col, meta1.Key, false, t)
}
