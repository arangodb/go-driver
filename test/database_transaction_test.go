//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

func TestDatabaseTransaction(t *testing.T) {
	c := createClient(t, nil)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(nil, c, "transaction_test", nil, t)

	const colName = "books"
	ensureCollection(context.Background(), db, colName, nil, t)

	txOptions := &driver.TransactionOptions{
		ReadCollections:      []string{colName},
		WriteCollections:     []string{colName},
		ExclusiveCollections: []string{colName},
	}
	testCases := []struct {
		name         string
		action       string
		options      *driver.TransactionOptions
		expectResult interface{}
		expectError  error
	}{
		{"ReturnValue", "function () { return 'worked!'; }", txOptions, "worked!", nil},
		{"ReturnError", "function () { error error; }", txOptions, nil, fmt.Errorf("missing/invalid action definition for transaction - Uncaught SyntaxError: Unexpected identifier - SyntaxError: Unexpected identifier\n    at new Function (<anonymous>)")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := db.Transaction(nil, testCase.action, testCase.options)
			if !reflect.DeepEqual(testCase.expectResult, result) {
				t.Errorf("expected result %v, got %v", testCase.expectResult, result)
			}
			if testCase.expectError != nil {
				if testCase.expectError.Error() != err.Error() {
					t.Errorf("expected error %v, got %v", testCase.expectError.Error(), err.Error())
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
