//
// DISCLAIMER
//
// Copyright 2018-2023 ArangoDB GmbH, Cologne, Germany
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
	"errors"

	"github.com/arangodb/go-driver"
)

// CreateDocuments creates given number of documents for the provided collection.
func CreateDocuments(ctx context.Context, col driver.Collection, docCount int, generator func(i int) any) error {
	if generator == nil {
		return errors.New("document generator can not be nil")
	}
	if col == nil {
		return errors.New("collection can not be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	docs := make([]any, 0, docCount)
	for i := 0; i < docCount; i++ {
		docs = append(docs, generator(i))
	}

	_, _, err := col.CreateDocuments(ctx, docs)
	return err
}
