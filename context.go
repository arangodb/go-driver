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

const (
	keyRevision    = "arangodb-revision"
	keyReturnNew   = "arangodb-returnNew"
	keyReturnOld   = "arangodb-returnOld"
	keyWaitForSync = "arangodb-waitForSync"
)

// WithRevision is used to configure a context to make document
// functions specify an explicit revision of the document using an `If-Match` condition.
func WithRevision(parent context.Context, revision string) context.Context {
	return context.WithValue(parent, keyRevision, revision)
}

// WithReturnNew is used to configure a context to make create, update & replace document
// functions return the new document into the given result.
func WithReturnNew(parent context.Context, result interface{}) context.Context {
	return context.WithValue(parent, keyReturnNew, result)
}

// WithReturnOld is used to configure a context to make update & replace document
// functions return the old document into the given result.
func WithReturnOld(parent context.Context, result interface{}) context.Context {
	return context.WithValue(parent, keyReturnOld, result)
}

// WithWaitForSync is used to configure a context to make modification
// functions wait until the data has been synced to disk.
func WithWaitForSync(parent context.Context) context.Context {
	return context.WithValue(parent, keyWaitForSync, true)
}
