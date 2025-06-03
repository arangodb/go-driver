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
	"fmt"
	"testing"
)

// BenchmarkCollectionExists measures the CollectionExists operation.
func BenchmarkCollectionExists(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "collection_exist_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.CollectionExists(nil, col.Name()); err != nil {
			b.Errorf("CollectionExists failed: %s", describe(err))
		}
	}
}

// BenchmarkCollection measures the Collection operation.
func BenchmarkCollection(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	col := ensureCollection(nil, db, "collection_test", nil, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Collection(nil, col.Name()); err != nil {
			b.Errorf("Collection failed: %s", describe(err))
		}
	}
}

// BenchmarkCollections measures the Collections operation.
func BenchmarkCollections(b *testing.B) {
	c := createClient(b, nil)
	db := ensureDatabase(nil, c, "collection_test", nil, b)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	for i := 0; i < 10; i++ {
		ensureCollection(nil, db, fmt.Sprintf("col%d", i), nil, b)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Collections(nil); err != nil {
			b.Errorf("Collections failed: %s", describe(err))
		}
	}
}
