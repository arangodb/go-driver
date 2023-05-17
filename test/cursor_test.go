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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

func TestCreateCursorWithMaxRuntime(t *testing.T) {
	// Arrange
	collectionName := "cursor_max_retry_test"
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.6", t)
	db := ensureDatabase(context.Background(), c, "cursor_test", nil, t)
	ensureCollection(context.Background(), db, collectionName, nil, t)

	tests := []struct {
		Name          string
		SleepQuery    string
		MaxRuntime    float64
		ExpectedError string
	}{
		{
			Name:          "Too long query interrupted",
			SleepQuery:    "1",
			MaxRuntime:    0.01,
			ExpectedError: "query killed (while executing)",
		},
		{
			Name:       "Query passed before max runtime",
			SleepQuery: "0.01",
			MaxRuntime: 20.0,
		},
		{
			Name:       "Query passed without max runtime",
			SleepQuery: "0.01",
			MaxRuntime: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Act
			ctx := driver.WithQueryMaxRuntime(nil, test.MaxRuntime)
			_, err := db.Query(ctx, "LET y = SLEEP("+test.SleepQuery+") INSERT {t:''} INTO "+collectionName, nil)

			// Assert
			if test.ExpectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.ExpectedError)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestWithQueryOptimizerRules(t *testing.T) {
	collectionName := "col_query_optimizer_rules"
	fieldName := "value"
	c := createClientFromEnv(t, true)
	db := ensureDatabase(context.Background(), c, "query_optimizer_rules", nil, t)
	col := ensureCollection(context.Background(), db, collectionName, nil, t)

	tests := map[string]struct {
		OptimizerRules   []string
		ExpectedRules    []string
		NotExpectedRules []string
	}{
		"include optimizer rule: use-indexes": {
			OptimizerRules: []string{"+use-indexes"},
			ExpectedRules:  []string{"use-indexes"},
		},
		"exclude optimizer rule: use-indexes": {
			OptimizerRules:   []string{"-use-indexes"},
			NotExpectedRules: []string{"use-indexes"},
		},
		"overwrite excluded optimizer rule: use-indexes": {
			OptimizerRules: []string{"-use-indexes", "+use-indexes"},
			ExpectedRules:  []string{"use-indexes"},
		},
		"overwrite included optimizer rule: use-indexes": {
			OptimizerRules:   []string{"+use-indexes", "-use-indexes"},
			NotExpectedRules: []string{"use-indexes"},
		},
		"turn off all optimizer rule": {
			OptimizerRules:   []string{"-all"},        // some rules can not be disabled.
			NotExpectedRules: []string{"use-indexes"}, // this rule will be disabled with all disabled rules.
		},
		"turn on all optimizer rule": {
			OptimizerRules: []string{"+all"},
			ExpectedRules:  []string{"use-indexes"}, // this rule will be enabled with all enabled rules.
		},
	}

	ctxBackground := context.Background()
	type testDoc struct {
		Value int `json:"value"` // variable fieldName
	}

	err := CreateDocuments(ctxBackground, col, 100, func(index int) any {
		return testDoc{Value: index}
	})
	require.NoErrorf(t, err, "failed to create exemplary documents")

	_, _, err = col.EnsurePersistentIndex(ctxBackground, []string{fieldName}, &driver.EnsurePersistentIndexOptions{
		Name: "index",
	})
	require.NoErrorf(t, err, "failed to index for collection \"%s\"", collectionName)

	ctxWithProfile := driver.WithQueryProfile(ctxBackground, 2)
	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			ctx := ctxWithProfile
			if len(test.OptimizerRules) > 0 {
				ctx = driver.WithQueryOptimizerRules(ctxWithProfile, test.OptimizerRules)
			}

			query := fmt.Sprintf("FOR i IN %s FILTER i.%s > 97 SORT i.%s RETURN i.%s", collectionName, fieldName,
				fieldName, fieldName)
			cursor, err := db.Query(ctx, query, nil)
			require.NoError(t, err, "query \"%s\" failed", query)

			planRaw, ok, err := cursor.Extra().GetPlanRaw()
			require.NoErrorf(t, err, "failed to get query's plan")
			require.True(t, ok)

			type result struct {
				Rules []string `json:"rules,omitempty"`
			}
			var r result
			err = json.Unmarshal(planRaw, &r)

			for _, rule := range test.ExpectedRules {
				require.Contains(t, r.Rules, rule)
			}

			for _, rule := range test.NotExpectedRules {
				require.NotContains(t, r.Rules, rule)
			}
		})
	}
}

// TestCreateCursor creates several cursors.
func TestCreateCursor(t *testing.T) {
	ctx := context.Background()
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)
	db := ensureDatabase(ctx, c, "cursor_test", nil, t)

	// Create data set
	collectionData := map[string][]interface{}{
		"books": {
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
		"users": {
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
	tests := []struct {
		Query             string
		BindVars          map[string]interface{}
		ExpectSuccess     bool
		ExpectedDocuments []interface{}
		DocumentType      reflect.Type
		HasFullCount      bool
		ExpectedFullCount int64
	}{
		{
			Query:             "FOR d IN books SORT d.Title RETURN d",
			ExpectSuccess:     true,
			ExpectedDocuments: collectionData["books"],
			DocumentType:      reflect.TypeOf(Book{}),
		},
		{
			Query:             "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars:          map[string]interface{}{"title": "Book 02"},
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{collectionData["books"][1]},
			DocumentType:      reflect.TypeOf(Book{}),
		},
		{
			Query:         "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			BindVars:      map[string]interface{}{"somethingelse": "Book 02"},
			ExpectSuccess: false, // Unknown `@title`
		},
		{
			Query:             "FOR u IN users FILTER u.age>100 SORT u.name RETURN u",
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{},
			DocumentType:      reflect.TypeOf(UserDoc{}),
		},
		{
			Query:         "FOR u IN users FILTER u.age<@maxAge SORT u.name RETURN u",
			BindVars:      map[string]interface{}{"maxAge": 20},
			ExpectSuccess: true,
			ExpectedDocuments: []interface{}{
				collectionData["users"][2],
				collectionData["users"][0],
				collectionData["users"][5],
			},
			DocumentType: reflect.TypeOf(UserDoc{}),
		},
		{
			Query:         "FOR u IN users FILTER u.age<@maxAge SORT u.name RETURN u",
			BindVars:      map[string]interface{}{"maxage": 20},
			ExpectSuccess: false, // `@maxage` versus `@maxAge`
		},
		{
			Query:             "FOR u IN users SORT u.age RETURN u.age",
			ExpectedDocuments: []interface{}{12, 12, 13, 25, 42, 67},
			DocumentType:      reflect.TypeOf(12),
			ExpectSuccess:     true,
		},
		{
			Query:             "FOR p IN users COLLECT a = p.age WITH COUNT INTO c SORT a RETURN [a, c]",
			ExpectedDocuments: []interface{}{[]int{12, 2}, []int{13, 1}, []int{25, 1}, []int{42, 1}, []int{67, 1}},
			DocumentType:      reflect.TypeOf([]int{}),
			ExpectSuccess:     true,
		},
		{
			Query:             "FOR u IN users SORT u.name RETURN u.name",
			ExpectedDocuments: []interface{}{"Blair", "Clair", "Jake", "John", "Johnny", "Zz"},
			DocumentType:      reflect.TypeOf("foo"),
			ExpectSuccess:     true,
		},
		{
			Query:             "FOR d IN books SORT d.Title LIMIT 1, 1 RETURN d",
			ExpectSuccess:     true,
			ExpectedDocuments: []interface{}{collectionData["books"][1]},
			DocumentType:      reflect.TypeOf(Book{}),
			HasFullCount:      true,
			ExpectedFullCount: 20,
		},
	}

	// Setup context alternatives
	contexts := []struct {
		Context         context.Context
		ExpectCount     bool
		ExpectFullCount bool
	}{
		{Context: nil},
		{Context: context.Background()},
		{Context: driver.WithQueryCount(nil), ExpectCount: true},
		{Context: driver.WithQueryCount(nil, true), ExpectCount: true},
		{Context: driver.WithQueryCount(nil, false)},
		{Context: driver.WithQueryBatchSize(nil, 1)},
		{Context: driver.WithQueryCache(nil)},
		{Context: driver.WithQueryCache(nil, true)},
		{Context: driver.WithQueryCache(nil, false)},
		{Context: driver.WithQueryMemoryLimit(nil, 600000)},
		{Context: driver.WithQueryTTL(nil, time.Minute)},
		{Context: driver.WithQueryBatchSize(driver.WithQueryCount(nil), 1), ExpectCount: true},
		{
			Context:     driver.WithQueryCache(driver.WithQueryCount(driver.WithQueryBatchSize(nil, 2))),
			ExpectCount: true,
		},
		{Context: driver.WithQueryFullCount(nil, true), ExpectFullCount: true},
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

				executionTime := cursor.Statistics().ExecutionTime()
				require.Greaterf(t, executionTime.Nanoseconds(), int64(0), "execution time must be greater than 0")

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
	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, true, false)

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
	docs := 10000
	sendBulks(t, col, ctx, func(t *testing.T, i int) interface{} {
		return UserDoc{Name: "John", Age: i}
	}, docs)
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
