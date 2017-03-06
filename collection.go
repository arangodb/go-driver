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

	// Status fetches the current status of the collection.
	Status(ctx context.Context) (CollectionStatus, error)

	// Count fetches the number of document in the collection.
	Count(ctx context.Context) (int64, error)

	// Properties fetches extended information about the collection.
	Properties(ctx context.Context) (CollectionProperties, error)

	// SetProperties changes properties of the collection.
	SetProperties(ctx context.Context, options SetCollectionPropertiesOptions) error

	// Load the collection into memory.
	Load(ctx context.Context) error

	// UnLoad the collection from memory.
	Unload(ctx context.Context) error

	// Rename the collection
	Rename(ctx context.Context, newName string) error

	// Remove removes the entire collection.
	// If the collection does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Truncate removes all documents from the collection, but leaves the indexes intact.
	Truncate(ctx context.Context) error

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

// CollectionProperties contains extended information about a collection.
type CollectionProperties struct {
	CollectionInfo

	// WaitForSync; If true then creating, changing or removing documents will wait until the data has been synchronized to disk.
	WaitForSync bool `json:"waitForSync,omitempty"`
	// DoCompact specifies whether or not the collection will be compacted.
	DoCompact bool `json:"doCompact,omitempty"`
	// JournalSize is the maximal size setting for journals / datafiles in bytes.
	JournalSize int64 `json:"journalSize,omitempty"`
	KeyOptions  struct {
		// Type specifies the type of the key generator. The currently available generators are traditional and autoincrement.
		Type KeyGeneratorType `json:"type,omitempty"`
		// AllowUserKeys; if set to true, then it is allowed to supply own key values in the _key attribute of a document.
		// If set to false, then the key generator is solely responsible for generating keys and supplying own key values in
		// the _key attribute of documents is considered an error.
		AllowUserKeys bool `json:"allowUserKeys,omitempty"`
	} `json:"keyOptions,omitempty"`
	// NumberOfShards is the number of shards of the collection.
	// Only available in cluster setup.
	NumberOfShards int `json:"numberOfShards,omitempty"`
	// ShardKeys contains the names of document attributes that are used to determine the target shard for documents.
	// Only available in cluster setup.
	ShardKeys []string `json:"shardKeys,omitempty"`
	// ReplicationFactor contains how many copies of each shard are kept on different DBServers.
	// Only available in cluster setup.
	ReplicationFactor int `json:"replicationFactor,omitempty"`
}

// SetCollectionPropertiesOptions contains data for Collection.SetProperties.
type SetCollectionPropertiesOptions struct {
	// If true then creating or changing a document will wait until the data has been synchronized to disk.
	WaitForSync *bool `json:"waitForSync,omitempty"`
	// The maximal size of a journal or datafile in bytes. The value must be at least 1048576 (1 MB). Note that when changing the journalSize value, it will only have an effect for additional journals or datafiles that are created. Already existing journals or datafiles will not be affected.
	JournalSize int64 `json:"journalSize,omitempty"`
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
