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
	"reflect"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
)

type queryTest struct {
	Query             string
	BindVars          map[string]interface{}
	ExpectSuccess     bool
	ExpectedDocuments []interface{}
	DocumentType      reflect.Type
	HasFullCount      bool
	ExpectedFullCount int64
}

type queryTestContext struct {
	Context         context.Context
	ExpectCount     bool
	ExpectFullCount bool
}

// TestCreateCursor creates several cursors.
func TestCreateCursor(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "cursor_test", nil, t)

	// Create data set
	collectionData := map[string][]interface{}{
		"books": []interface{}{
			Book{Title: "Book 01"},
			Book{Title: "Book 02"},
			Book{Title: "Book 03"},
			Book{Title: "Book 04"},
			Book{Title: "Book 05"},
			Book{Title: "Book 06"},
			Book{Title: "Book 07"},
			Book{Title: "Book 08"},
			Book{Title: "Book 09"},
			Book{Title: "Book 10"},
			Book{Title: "Book 11"},
			Book{Title: "Book 12"},
			Book{Title: "Book 13"},
			Book{Title: "Book 14"},
			Book{Title: "Book 15"},
			Book{Title: "Book 16"},
			Book{Title: "Book 17"},
			Book{Title: "Book 18"},
			Book{Title: "Book 19"},
			Book{Title: "Book 20"},
		},
		"users": []interface{}{
			UserDoc{Name: "John", Age: 13},
			UserDoc{Name: "Jake", Age: 25},
			UserDoc{Name: "Clair", Age: 12},
			UserDoc{Name: "Johnny", Age: 42},
			UserDoc{Name: "Blair", Age: 67},
			UserDoc{Name: "Zz", Age: 12},
		},
	}
	for colName, colDocs := range collectionData {
		col := ensureCollection(ctx, db, colName, nil, t)
		if _, _, err := col.CreateDocuments(ctx, colDocs); err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
	}

	// Setup tests
	tests := []queryTest{
		queryTest{
			Query:             "FOR d IN books SORT d.Title RETURN d",
			ExpectSuccess:     true,
			ExpectedDocuments: collectionData["books"],
			DocumentType:      reflect.TypeOf(Book{}),
		},
		queryTest{
			Query:             "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars:          map[string]interface{}{"title": "Book 02"},
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{collectionData["books"][1]},
			DocumentType:      reflect.TypeOf(Book{}),
		},
		queryTest{
			Query:         "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars:      map[string]interface{}{"somethingelse": "Book 02"},
			ExpectSuccess: false, // Unknown `@title`
		},
		queryTest{
			Query:             "FOR u IN users FILTER u.age>100 SORT u.name RETURN u",
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{},
			DocumentType:      reflect.TypeOf(UserDoc{}),
		},
		queryTest{
			Query:             "FOR u IN users FILTER u.age<@maxAge SORT u.name RETURN u",
			BindVars:          map[string]interface{}{"maxAge": 20},
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{collectionData["users"][2], collectionData["users"][0], collectionData["users"][5]},
			DocumentType:      reflect.TypeOf(UserDoc{}),
		},
		queryTest{
			Query:         "FOR u IN users FILTER u.age<@maxAge SORT u.name RETURN u",
			BindVars:      map[string]interface{}{"maxage": 20},
			ExpectSuccess: false, // `@maxage` versus `@maxAge`
		},
		queryTest{
			Query:             "FOR u IN users SORT u.age RETURN u.age",
			ExpectedDocuments: []interface{}{12, 12, 13, 25, 42, 67},
			DocumentType:      reflect.TypeOf(12),
			ExpectSuccess:     true,
		},
		queryTest{
			Query:             "FOR p IN users COLLECT a = p.age WITH COUNT INTO c SORT a RETURN [a, c]",
			ExpectedDocuments: []interface{}{[]int{12, 2}, []int{13, 1}, []int{25, 1}, []int{42, 1}, []int{67, 1}},
			DocumentType:      reflect.TypeOf([]int{}),
			ExpectSuccess:     true,
		},
		queryTest{
			Query:             "FOR u IN users SORT u.name RETURN u.name",
			ExpectedDocuments: []interface{}{"Blair", "Clair", "Jake", "John", "Johnny", "Zz"},
			DocumentType:      reflect.TypeOf("foo"),
			ExpectSuccess:     true,
		},
		queryTest{
			Query:             "FOR d IN books SORT d.Title LIMIT 1, 1 RETURN d",
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{collectionData["books"][1]},
			DocumentType:      reflect.TypeOf(Book{}),
			HasFullCount:      true,
			ExpectedFullCount: 20,
		},
	}

	// Setup context alternatives
	contexts := []queryTestContext{
		queryTestContext{Context: nil},
		queryTestContext{Context: context.Background()},
		queryTestContext{Context: driver.WithQueryCount(nil), ExpectCount: true},
		queryTestContext{Context: driver.WithQueryCount(nil, true), ExpectCount: true},
		queryTestContext{Context: driver.WithQueryCount(nil, false)},
		queryTestContext{Context: driver.WithQueryBatchSize(nil, 1)},
		queryTestContext{Context: driver.WithQueryCache(nil)},
		queryTestContext{Context: driver.WithQueryCache(nil, true)},
		queryTestContext{Context: driver.WithQueryCache(nil, false)},
		queryTestContext{Context: driver.WithQueryMemoryLimit(nil, 60000)},
		queryTestContext{Context: driver.WithQueryTTL(nil, time.Minute)},
		queryTestContext{Context: driver.WithQueryBatchSize(driver.WithQueryCount(nil), 1), ExpectCount: true},
		queryTestContext{Context: driver.WithQueryCache(driver.WithQueryCount(driver.WithQueryBatchSize(nil, 2))), ExpectCount: true},
		queryTestContext{Context: driver.WithQueryFullCount(nil, true), ExpectFullCount: true},
	}

	// Run tests for every context alternative
	for _, qctx := range contexts {
		ctx := qctx.Context
		for i, test := range tests {
			cursor, err := db.Query(ctx, test.Query, test.BindVars)
			if err == nil {
				// Close upon exit of the function
				defer cursor.Close()
			}
			if test.ExpectSuccess {
				if err != nil {
					t.Errorf("Expected success in query %d (%s), got '%s'", i, test.Query, describe(err))
					continue
				}
				count := cursor.Count()
				if qctx.ExpectCount {
					if count != int64(len(test.ExpectedDocuments)) {
						t.Errorf("Expected count of %d, got %d in query %d (%s)", len(test.ExpectedDocuments), count, i, test.Query)
					}
				} else {
					if count != 0 {
						t.Errorf("Expected count of 0, got %d in query %d (%s)", count, i, test.Query)
					}
				}
				if qctx.ExpectFullCount && test.HasFullCount {
					stat := cursor.Statistics()
					if stat.FullCount() != test.ExpectedFullCount {
						t.Errorf("Expected full count of %d, got %d in query %d (%s)", test.ExpectedFullCount, stat.FullCount(), i, test.Query)
					}
				}
				var result []interface{}
				for {
					hasMore := cursor.HasMore()
					doc := reflect.New(test.DocumentType)
					if _, err := cursor.ReadDocument(ctx, doc.Interface()); driver.IsNoMoreDocuments(err) {
						if hasMore {
							t.Error("HasMore returned true, but ReadDocument returns a IsNoMoreDocuments error")
						}
						break
					} else if err != nil {
						t.Errorf("Failed to result document %d: %s", len(result), describe(err))
					}
					if !hasMore {
						t.Error("HasMore returned false, but ReadDocument returns a document")
					}
					result = append(result, doc.Elem().Interface())
				}
				if len(result) != len(test.ExpectedDocuments) {
					t.Errorf("Expected %d documents, got %d in query %d (%s)", len(test.ExpectedDocuments), len(result), i, test.Query)
				} else {
					for resultIdx, resultDoc := range result {
						if !reflect.DeepEqual(resultDoc, test.ExpectedDocuments[resultIdx]) {
							t.Errorf("Unexpected document in query %d (%s) at index %d: got %+v, expected %+v", i, test.Query, resultIdx, resultDoc, test.ExpectedDocuments[resultIdx])
						}
					}
				}
				// Close anyway (this tests calling Close more than once)
				if err := cursor.Close(); err != nil {
					t.Errorf("Expected success in Close of cursor from query %d (%s), got '%s'", i, test.Query, describe(err))
				}
			} else {
				if err == nil {
					t.Errorf("Expected error in query %d (%s), got '%s'", i, test.Query, describe(err))
					continue
				}
			}
		}
	}
}

// TestCreateCursorReturnNull creates a cursor with a `RETURN NULL` query.
func TestCreateCursorReturnNull(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "cursor_test", nil, t)

	var result interface{}
	query := "return null"
	cursor, err := db.Query(ctx, query, nil)
	if err != nil {
		t.Fatalf("Query(return null) failed: %s", describe(err))
	}
	defer cursor.Close()
	if _, err := cursor.ReadDocument(ctx, &result); err != nil {
		t.Fatalf("ReadDocument failed: %s", describe(err))
	}
	if result != nil {
		t.Errorf("Expected result to be nil, got %#v", result)
	}
}

// Test stream query cursors. The goroutines are technically only
// relevant for the MMFiles engine, but don't hurt on rocksdb either
func TestCreateStreamCursor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c := createClientFromEnv(t, true)

	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	if version.Version.CompareTo("3.4") < 0 {
		t.Skip("This test requires version 3.4")
		return
	}

	db := ensureDatabase(ctx, c, "cursor_stream_test", nil, t)
	col := ensureCollection(ctx, db, "cursor_stream_test", nil, t)

	// Query engine info (on rocksdb, JournalSize is always 0)
	info, err := db.EngineInfo(nil)
	if err != nil {
		t.Fatalf("Failed to get engine info: %s", describe(err))
	}

	// This might take a few seconds
	for i := 0; i < 10000; i++ {
		user := UserDoc{Name: "John", Age: i}
		if _, err := col.CreateDocument(ctx, user); err != nil {
			t.Fatalf("Expected success, got %s", describe(err))
		}
	}
	t.Log("Completed inserting 10k docs")

	const expectedResults int = 10 * 10000
	query := "FOR doc IN cursor_stream_test RETURN doc"
	ctx2 := driver.WithQueryStream(ctx, true)
	var cursors []driver.Cursor

	// create a bunch of read-only cursors
	for i := 0; i < 10; i++ {
		cursor, err := db.Query(ctx2, query, nil)
		if err != nil {
			t.Fatalf("Expected success in query %d (%s), got '%s'", i, query, describe(err))
		}
		defer cursor.Close()
		count := cursor.Count()
		if count != 0 {
			t.Errorf("Expected count of 0, got %d in query %d (%s)", count, i, query)
		}
		stats := cursor.Statistics()
		count = stats.FullCount()
		if count != 0 {
			t.Errorf("Expected fullCount of 0, got %d in query %d (%s)", count, i, query)
		}
		if !cursor.HasMore() {
			t.Errorf("Expected cursor %d to have more documents", i)
		}

		cursors = append(cursors, cursor)
	}

	t.Logf("Created %d cursors", len(cursors))

	// start a write query on the same collection inbetween
	// contrary to normal cursors which are executed right
	// away this will block until all read cursors are resolved
	testReady := make(chan bool)
	go func() {
		query = "FOR doc IN 1..5 LET y = SLEEP(0.01) INSERT {name:'Peter', age:0} INTO cursor_stream_test"
		cursor, err := db.Query(ctx2, query, nil) // should not return immediately
		if err != nil {
			t.Fatalf("Expected success in write-query %s, got '%s'", query, describe(err))
		}
		defer cursor.Close()

		for cursor.HasMore() {
			var data interface{}
			if _, err := cursor.ReadDocument(ctx2, &data); err != nil {
				t.Fatalf("Failed to read document, err: %s", describe(err))
			}
		}
		testReady <- true // signal write done
	}()

	readCount := 0
	go func() {
		// read all cursors until the end, server closes them automatically
		for i, cursor := range cursors {
			for cursor.HasMore() {
				var user UserDoc
				if _, err := cursor.ReadDocument(ctx2, &user); err != nil {
					t.Fatalf("Failed to result document %d: %s", i, describe(err))
				}
				readCount++
			}
		}
		testReady <- false // signal read done
	}()

	writeDone := false
	readDone := false
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout")
		case v := <-testReady:
			if v {
				writeDone = true
			} else {
				readDone = true
			}
		}
		// On MMFiles the read-cursors have to finish first
		if writeDone && !readDone && info.Type == driver.EngineTypeMMFiles {
			t.Error("Write cursor was able to complete before read cursors")
		}

		if writeDone && readDone {
			close(testReady)
			break
		}
	}

	if readCount != expectedResults {
		t.Errorf("Expected to read %d documents, instead got %d", expectedResults, readCount)
	}
}
