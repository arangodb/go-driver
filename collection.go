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

// Collection provides access to the information of a single collection, all its documents and all its indexes.
type Collection interface {
	// Name returns the name of the collection.
	Name() string

	// Remove removes the entire collection.
	// If the collection does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// All index functions
	CollectionIndexes

	// All document functions
	CollectionDocuments
}

// CollectionInfo contains information about a collection
type CollectionInfo struct {
	// The identifier of the collection.
	ID string `json:"id,omitempty"`
	// The name of the collection.
	Name string `json:"name,omitempty"`
	// The status of the collection
	Status CollectionStatus `json:"status,omitempty"`
	// The type of the collection
	Type CollectionType `json:"type,omitempty"`
	// If true then the collection is a system collection.
	IsSystem bool `json:"isSystem,omitempty"`
}

// CollectionStatus indicates the status of a collection.
type CollectionStatus int

const (
	CollectionStatusNewBorn   = CollectionStatus(1)
	CollectionStatusUnloaded  = CollectionStatus(2)
	CollectionStatusLoaded    = CollectionStatus(3)
	CollectionStatusUnloading = CollectionStatus(4)
	CollectionStatusDeleted   = CollectionStatus(5)
	CollectionStatusLoading   = CollectionStatus(6)
)
