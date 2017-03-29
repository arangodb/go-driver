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
	"testing"
)

type validateQueryTest struct {
	Query         string
	ExpectSuccess bool
}

// TestValidateQuery validates several AQL queries.
func TestValidateQuery(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	db := ensureDatabase(ctx, c, "validate_query_test", nil, t)

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
	tests := []validateQueryTest{
		validateQueryTest{
			Query:         "FOR d IN books SORT d.Title RETURN d",
			ExpectSuccess: true,
		},
		validateQueryTest{
			Query:         "FOR d IN books FILTER d.Title==@title SORT d.Title RETURN d",
			ExpectSuccess: true,
		},
		validateQueryTest{
			Query:         "FOR u IN users FILTER u.age>>>100 SORT u.name RETURN u",
			ExpectSuccess: false,
		},
		validateQueryTest{
			Query:         "",
			ExpectSuccess: false,
		},
		validateQueryTest{
			Query:         "FOR u IN unknown RETURN u",
			ExpectSuccess: false,
		},
	}

	// Run tests for every context alternative
	for i, test := range tests {
		err := db.ValidateQuery(ctx, test.Query)
		if test.ExpectSuccess {
			if err != nil {
				t.Errorf("Expected success in query %d (%s), got '%s'", i, test.Query, describe(err))
				continue
			}
		} else {
			if err == nil {
				t.Errorf("Expected error in query %d (%s), got '%s'", i, test.Query, describe(err))
				continue
			}
		}
	}
}
