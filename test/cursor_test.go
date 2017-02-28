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
			ExpectedDocuments: []interface{}{collectionData["users"][2], collectionData["users"][0]},
			DocumentType:      reflect.TypeOf(UserDoc{}),
		},
		queryTest{
			Query:         "FOR u IN users FILTER u.age<@maxAge SORT u.name RETURN u",
			BindVars:      map[string]interface{}{"maxage": 20},
			ExpectSuccess: false, // `@maxage` versus `@maxAge`
		},
	}

	// Setup context alternatives
	contexts := []context.Context{
		nil,
		context.Background(),
		driver.WithQueryCount(nil, true),
		driver.WithQueryCount(nil, false),
		driver.WithQueryBatchSize(nil, 1),
		driver.WithQueryCache(nil, true),
		driver.WithQueryCache(nil, false),
		driver.WithQueryMemoryLimit(nil, 10000),
		driver.WithQueryTTL(nil, time.Minute),
	}

	// Run tests for every context alternative
	for _, ctx := range contexts {
		for i, test := range tests {
			cursor, err := db.Query(ctx, test.Query, test.BindVars)
			if err == nil {
				defer cursor.Close()
			}
			if test.ExpectSuccess {
				if err != nil {
					t.Errorf("Expected success in query %d (%s), got '%s'", i, test.Query, describe(err))
					continue
				}
				var result []interface{}
				for {
					doc := reflect.New(test.DocumentType)
					if _, err := cursor.ReadDocument(ctx, doc.Interface()); driver.IsNoMoreDocuments(err) {
						break
					} else if err != nil {
						t.Errorf("Failed to result document %d: %s", len(result), describe(err))
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
			} else {
				if err == nil {
					t.Errorf("Expected error in query %d (%s), got '%s'", i, test.Query, describe(err))
					continue
				}
			}
		}
	}
}
