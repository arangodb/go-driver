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

package driver

import "context"

// Collection provides access to the documents in a single collection.
type Collection interface {
	// Name returns the name of the collection.
	Name() string

	// ReadDocument reads a single document with given key from the collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error)

	// UpdateDocument updates a single document with given key in the collection.
	// The document data is stored into result, the document meta data is returned.
	// If no document exists with given key, a NotFoundError is returned.
	UpdateDocument(ctx context.Context, key string, update map[string]interface{}, resultOld, resultNew interface{}) (DocumentMeta, error)
}
