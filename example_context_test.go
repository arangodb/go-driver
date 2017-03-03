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

// +build !auth

package driver_test

import (
	"context"

	driver "github.com/arangodb/go-driver"
)

func ExampleWithRevision(collection driver.Collection) {
	var result Book
	// Using WithRevision we get an error when the current revision of the document is different.
	ctx := driver.WithRevision(context.Background(), "a-specific-revision")
	if _, err := collection.ReadDocument(ctx, "someValidKey", &result); err != nil {
		// This call will fail when a document does not exist, or when its current revision is different.
	}
}

func ExampleWithSilent(collection driver.Collection) {
	var result Book
	// Using WithSilent we do not care about any returned meta data.
	ctx := driver.WithSilent(context.Background())
	if _, err := collection.ReadDocument(ctx, "someValidKey", &result); err != nil {
		// No meta data is returned
	}
}
