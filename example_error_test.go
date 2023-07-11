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

//go:build !auth

package driver_test

import (
	"context"

	driver "github.com/arangodb/go-driver"
)

func ExampleIsNotFound() {
	var sampleCollection driver.Collection
	var result Book

	if _, err := sampleCollection.ReadDocument(nil, "keyDoesNotExist", &result); driver.IsNotFound(err) {
		// No document with given key exists
	}
}

func ExampleIsPreconditionFailed() {
	var sampleCollection driver.Collection
	var result Book

	ctx := driver.WithRevision(context.Background(), "an-old-revision")
	if _, err := sampleCollection.ReadDocument(ctx, "someValidKey", &result); driver.IsPreconditionFailed(err) {
		// The Document is found, but its revision is incorrect
	}
}
