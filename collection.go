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

import (
	"context"
	"time"
)

// Collection provides access to the information of a single collection, all its documents and all its indexes.
type Collection interface {
	// Name returns the name of the collection.
	Name() string

	// Status fetches the current status of the collection.
	Status(ctx context.Context) (CollectionStatus, error)

	// Count fetches the number of document in the collection.
	Count(ctx context.Context) (int64, error)

	// Statistics returns the number of documents and additional statistical information about the collection.
	Statistics(ctx context.Context) (CollectionStatistics, error)

	// Revision fetches the revision ID of the collection.
	// The revision ID is a server-generated string that clients can use to check whether data
	// in a collection has changed since the last revision check.
	Revision(ctx context.Context) (string, error)

	// Properties fetches extended information about the collection.
	Properties(ctx context.Context) (CollectionProperties, error)

	// SetProperties changes properties of the collection.
	SetProperties(ctx context.Context, options SetCollectionPropertiesOptions) error

	// Load the collection into memory.
	Load(ctx context.Context) error

	// UnLoad the collection from memory.
	Unload(ctx context.Context) error

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
	ID string `arangodb:"id,omitempty" json:"id,omitempty"`
	// The name of the collection.
	Name string `arangodb:"name,omitempty" json:"name,omitempty"`
	// The status of the collection
	Status CollectionStatus `arangodb:"status,omitempty" json:"status,omitempty"`
	// The type of the collection
	Type CollectionType `arangodb:"type,omitempty" json:"type,omitempty"`
	// If true then the collection is a system collection.
	IsSystem bool `arangodb:"isSystem,omitempty" json:"isSystem,omitempty"`
}

// CollectionProperties contains extended information about a collection.
type CollectionProperties struct {
	CollectionInfo

	// WaitForSync; If true then creating, changing or removing documents will wait until the data has been synchronized to disk.
	WaitForSync bool `arangodb:"waitForSync,omitempty" json:"waitForSync,omitempty"`
	// DoCompact specifies whether or not the collection will be compacted.
	DoCompact bool `arangodb:"doCompact,omitempty" json:"doCompact,omitempty"`
	// JournalSize is the maximal size setting for journals / datafiles in bytes.
	JournalSize int64 `arangodb:"journalSize,omitempty" json:"journalSize,omitempty"`
	KeyOptions  struct {
		// Type specifies the type of the key generator. The currently available generators are traditional and autoincrement.
		Type KeyGeneratorType `arangodb:"type,omitempty" json:"type,omitempty"`
		// AllowUserKeys; if set to true, then it is allowed to supply own key values in the _key attribute of a document.
		// If set to false, then the key generator is solely responsible for generating keys and supplying own key values in
		// the _key attribute of documents is considered an error.
		AllowUserKeys bool `arangodb:"allowUserKeys,omitempty" json:"allowUserKeys,omitempty"`
	} `arangodb:"keyOptions,omitempty" json:"keyOptions,omitempty"`
	// NumberOfShards is the number of shards of the collection.
	// Only available in cluster setup.
	NumberOfShards int `arangodb:"numberOfShards,omitempty" json:"numberOfShards,omitempty"`
	// ShardKeys contains the names of document attributes that are used to determine the target shard for documents.
	// Only available in cluster setup.
	ShardKeys []string `arangodb:"shardKeys,omitempty" json:"shardKeys,omitempty"`
	// ReplicationFactor contains how many copies of each shard are kept on different DBServers.
	// Only available in cluster setup.
	ReplicationFactor int `arangodb:"replicationFactor,omitempty" json:"replicationFactor,omitempty"`
}

// SetCollectionPropertiesOptions contains data for Collection.SetProperties.
type SetCollectionPropertiesOptions struct {
	// If true then creating or changing a document will wait until the data has been synchronized to disk.
	WaitForSync *bool `arangodb:"waitForSync,omitempty" json:"waitForSync,omitempty"`
	// The maximal size of a journal or datafile in bytes. The value must be at least 1048576 (1 MB). Note that when changing the journalSize value, it will only have an effect for additional journals or datafiles that are created. Already existing journals or datafiles will not be affected.
	JournalSize int64 `arangodb:"journalSize,omitempty" json:"journalSize,omitempty"`
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

// CollectionStatistics contains the number of documents and additional statistical information about a collection.
type CollectionStatistics struct {
	//The number of documents currently present in the collection.
	Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
	// The maximal size of a journal or datafile in bytes.
	JournalSize int64 `arangodb:"journalSize,omitempty" json:"journalSize,omitempty"`
	Figures     struct {
		DataFiles struct {
			// The number of datafiles.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total filesize of datafiles (in bytes).
			FileSize int64 `arangodb:"fileSize,omitempty" json:"fileSize,omitempty"`
		} `arangodb:"datafiles" json:"datafiles"`
		// The number of markers in the write-ahead log for this collection that have not been transferred to journals or datafiles.
		UncollectedLogfileEntries int64 `arangodb:"uncollectedLogfileEntries,omitempty" json:"uncollectedLogfileEntries,omitempty"`
		// The number of references to documents in datafiles that JavaScript code currently holds. This information can be used for debugging compaction and unload issues.
		DocumentReferences int64 `arangodb:"documentReferences,omitempty" json:"documentReferences,omitempty"`
		CompactionStatus   struct {
			// The action that was performed when the compaction was last run for the collection. This information can be used for debugging compaction issues.
			Message string `arangodb:"message,omitempty" json:"message,omitempty"`
			// The point in time the compaction for the collection was last executed. This information can be used for debugging compaction issues.
			Time time.Time `arangodb:"time,omitempty" json:"time,omitempty"`
		} `arangodb:"compactionStatus" json:"compactionStatus"`
		Compactors struct {
			// The number of compactor files.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total filesize of all compactor files (in bytes).
			FileSize int64 `arangodb:"fileSize,omitempty" json:"fileSize,omitempty"`
		} `arangodb:"compactors" json:"compactors"`
		Dead struct {
			// The number of dead documents. This includes document versions that have been deleted or replaced by a newer version. Documents deleted or replaced that are contained the write-ahead log only are not reported in this figure.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total number of deletion markers. Deletion markers only contained in the write-ahead log are not reporting in this figure.
			Deletion int64 `arangodb:"deletion,omitempty" json:"deletion,omitempty"`
			// The total size in bytes used by all dead documents.
			Size int64 `arangodb:"size,omitempty" json:"size,omitempty"`
		} `arangodb:"dead" json:"dead"`
		Indexes struct {
			// The total number of indexes defined for the collection, including the pre-defined indexes (e.g. primary index).
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total memory allocated for indexes in bytes.
			Size int64 `arangodb:"size,omitempty" json:"size,omitempty"`
		} `arangodb:"indexes" json:"indexes"`
		ReadCache struct {
			// The number of revisions of this collection stored in the document revisions cache.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The memory used for storing the revisions of this collection in the document revisions cache (in bytes). This figure does not include the document data but only mappings from document revision ids to cache entry locations.
			Size int64 `arangodb:"size,omitempty" json:"size,omitempty"`
		} `arangodb:"readcache" json:"readcache"`
		// An optional string value that contains information about which object type is at the head of the collection's cleanup queue. This information can be used for debugging compaction and unload issues.
		WaitingFor string `arangodb:"waitingFor,omitempty" json:"waitingFor,omitempty"`
		Alive      struct {
			// The number of currently active documents in all datafiles and journals of the collection. Documents that are contained in the write-ahead log only are not reported in this figure.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total size in bytes used by all active documents of the collection. Documents that are contained in the write-ahead log only are not reported in this figure.
			Size int64 `arangodb:"size,omitempty" json:"size,omitempty"`
		} `arangodb:"alive" json:"alive"`
		// The tick of the last marker that was stored in a journal of the collection. This might be 0 if the collection does not yet have a journal.
		LastTick int64 `arangodb:"lastTick,omitempty" json:"lastTick,omitempty"`
		Journals struct {
			// The number of journal files.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The total filesize of all journal files (in bytes).
			FileSize int64 `arangodb:"fileSize,omitempty" json:"fileSize,omitempty"`
		} `arangodb:"journals" json:"journals"`
		Revisions struct {
			// The number of revisions of this collection managed by the storage engine.
			Count int64 `arangodb:"count,omitempty" json:"count,omitempty"`
			// The memory used for storing the revisions of this collection in the storage engine (in bytes). This figure does not include the document data but only mappings from document revision ids to storage engine datafile positions.
			Size int64 `arangodb:"size,omitempty" json:"size,omitempty"`
		} `arangodb:"revisions" json:"revisions"`
	} `arangodb:"figures" json:"figures"`
}
