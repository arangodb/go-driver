//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
// Author Tomasz Mielech
//

package arangodb

import "context"

type Collection interface {
	Name() string
	Database() Database

	// Shards fetches shards information of the collection.
	Shards(ctx context.Context, details bool) (CollectionShards, error)

	// Remove removes the entire collection.
	// If the collection does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	CollectionDocuments
	CollectionIndexes
}
