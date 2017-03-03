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

// TestCreateFullTextIndex creates a collection with a full text index.
func TestIndexes(t *testing.T) {
	c := createClientFromEnv(t, true)
	db := ensureDatabase(nil, c, "index_test", nil, t)
	col := ensureCollection(nil, db, "indexes_test", nil, t)

	// Create some indexes
	if _, _, err := col.EnsureFullTextIndex(nil, []string{"name"}, nil); err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}
	if _, _, err := col.EnsureHashIndex(nil, []string{"age", "gender"}, nil); err != nil {
		t.Fatalf("Failed to create new index: %s", describe(err))
	}

	// Get list of indexes
	if idxs, err := col.Indexes(context.Background()); err != nil {
		t.Fatalf("Failed to get indexes: %s", describe(err))
	} else {
		if len(idxs) != 3 {
			// We made 2 indexes, 1 is always added by the system
			t.Errorf("Expected 3 indexes, got %d", len(idxs))
		}

		// Try opening the indexes 1 by 1
		for _, x := range idxs {
			if idx, err := col.Index(nil, x.Name()); err != nil {
				t.Errorf("Failed to open index '%s': %s", x.Name(), describe(err))
			} else if idx.Name() != x.Name() {
				t.Errorf("Got different index name. Expected '%s', got '%s'", x.Name(), idx.Name())
			}
		}
	}
}
