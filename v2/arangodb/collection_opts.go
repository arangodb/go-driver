//
// DISCLAIMER
//
// Copyright 2017-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package arangodb

import (
	"time"
)

// CollectionInfo contains basic information about a collection.
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
	// Global unique name for the collection
	GloballyUniqueId string `json:"globallyUniqueId,omitempty"`
}

// CollectionExtendedInfo contains extended information about a collection.
type CollectionExtendedInfo struct {
	CollectionInfo
	// CacheEnabled set cacheEnabled option in collection properties.
	CacheEnabled bool `json:"cacheEnabled,omitempty"`
	KeyOptions   struct {
		// Type specifies the type of the key generator. The currently available generators are traditional and autoincrement.
		Type KeyGeneratorType `json:"type,omitempty"`
		// AllowUserKeys; if set to true, then it is allowed to supply own key values in the _key attribute of a document.
		// If set to false, then the key generator is solely responsible for generating keys and supplying own key values in
		// the _key attribute of documents is considered an error.
		AllowUserKeys bool `json:"allowUserKeys,omitempty"`
	} `json:"keyOptions,omitempty"`
	// Deprecated: use 'WriteConcern' instead.
	MinReplicationFactor int `json:"minReplicationFactor,omitempty"`
	// NumberOfShards is the number of shards of the collection.
	// Only available in cluster setup.
	NumberOfShards int `json:"numberOfShards,omitempty"`
	// This attribute specifies the name of the sharding strategy to use for the collection.
	// Can not be changed after creation.
	ShardingStrategy ShardingStrategy `json:"shardingStrategy,omitempty"`
	// ShardKeys contains the names of document attributes that are used to determine the target shard for documents.
	// Only available in cluster setup.
	ShardKeys []string `json:"shardKeys,omitempty"`
	// ReplicationFactor contains how many copies of each shard are kept on different DBServers.
	// Only available in cluster setup.
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// WaitForSync; If true then creating, changing or removing documents will wait
	// until the data has been synchronized to disk.
	WaitForSync bool `json:"waitForSync,omitempty"`
	// WriteConcern contains how many copies must be available before a collection can be written.
	// It is required that 1 <= WriteConcern <= ReplicationFactor.
	// Default is 1. Not available for satellite collections.
	// Available from 3.6 arangod version.
	WriteConcern int `json:"writeConcern,omitempty"`
}

// CollectionProperties contains extended information about a collection.
type CollectionProperties struct {
	CollectionExtendedInfo
	// DoCompact specifies whether or not the collection will be compacted.
	DoCompact bool `json:"doCompact,omitempty"`
	// JournalSize is the maximal size setting for journals / datafiles in bytes.
	JournalSize int64 `json:"journalSize,omitempty"`
	// SmartJoinAttribute
	// See documentation for smart joins.
	// This requires ArangoDB Enterprise Edition.
	SmartJoinAttribute string `json:"smartJoinAttribute,omitempty"`
	// This attribute specifies that the sharding of a collection follows that of another
	// one.
	DistributeShardsLike string `json:"distributeShardsLike,omitempty"`
	// This attribute specifies if the new format introduced in 3.7 is used for this
	// collection.
	UsesRevisionsAsDocumentIds bool `json:"usesRevisionsAsDocumentIds,omitempty"`
	// The following attribute specifies if the new MerkleTree based sync protocol
	// can be used on the collection.
	SyncByRevision bool `json:"syncByRevision,omitempty"`
	// Schema for collection validation
	Schema *CollectionSchemaOptions `json:"schema,omitempty"`
}

const (
	// ReplicationFactorSatellite represents a satellite collection's replication factor
	ReplicationFactorSatellite int = -1
)

// IsSatellite returns true if the collection is a satellite collection
func (p *CollectionProperties) IsSatellite() bool {
	return p.ReplicationFactor == ReplicationFactorSatellite
}

// SetCollectionPropertiesOptions contains data for Collection.SetProperties.
type SetCollectionPropertiesOptions struct {
	// If true then creating or changing a document will wait until the data has been synchronized to disk.
	WaitForSync *bool `json:"waitForSync,omitempty"`
	// The maximal size of a journal or datafile in bytes. The value must be at least 1048576 (1 MB). Note that when changing the journalSize value, it will only have an effect for additional journals or datafiles that are created. Already existing journals or datafiles will not be affected.
	JournalSize int64 `json:"journalSize,omitempty"`
	// ReplicationFactor contains how many copies of each shard are kept on different DBServers.
	// Only available in cluster setup.
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Deprecated: use 'WriteConcern' instead
	MinReplicationFactor int `json:"minReplicationFactor,omitempty"`
	// WriteConcern contains how many copies must be available before a collection can be written.
	// Available from 3.6 arangod version.
	WriteConcern int `json:"writeConcern,omitempty"`
	// CacheEnabled set cacheEnabled option in collection properties
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`
	// Schema for collection validation
	Schema *CollectionSchemaOptions `json:"schema,omitempty"`
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
	Count int64 `json:"count,omitempty"`
	// The maximal size of a journal or datafile in bytes.
	JournalSize int64 `json:"journalSize,omitempty"`
	Figures     struct {
		DataFiles struct {
			// The number of datafiles.
			Count int64 `json:"count,omitempty"`
			// The total filesize of datafiles (in bytes).
			FileSize int64 `json:"fileSize,omitempty"`
		} `json:"datafiles"`
		// The number of markers in the write-ahead log for this collection that have not been transferred to journals or datafiles.
		UncollectedLogfileEntries int64 `json:"uncollectedLogfileEntries,omitempty"`
		// The number of references to documents in datafiles that JavaScript code currently holds. This information can be used for debugging compaction and unload issues.
		DocumentReferences int64 `json:"documentReferences,omitempty"`
		CompactionStatus   struct {
			// The action that was performed when the compaction was last run for the collection. This information can be used for debugging compaction issues.
			Message string `json:"message,omitempty"`
			// The point in time the compaction for the collection was last executed. This information can be used for debugging compaction issues.
			Time time.Time `json:"time,omitempty"`
		} `json:"compactionStatus"`
		Compactors struct {
			// The number of compactor files.
			Count int64 `json:"count,omitempty"`
			// The total filesize of all compactor files (in bytes).
			FileSize int64 `json:"fileSize,omitempty"`
		} `json:"compactors"`
		Dead struct {
			// The number of dead documents. This includes document versions that have been deleted or replaced by a newer version. Documents deleted or replaced that are contained the write-ahead log only are not reported in this figure.
			Count int64 `json:"count,omitempty"`
			// The total number of deletion markers. Deletion markers only contained in the write-ahead log are not reporting in this figure.
			Deletion int64 `json:"deletion,omitempty"`
			// The total size in bytes used by all dead documents.
			Size int64 `json:"size,omitempty"`
		} `json:"dead"`
		Indexes struct {
			// The total number of indexes defined for the collection, including the pre-defined indexes (e.g. primary index).
			Count int64 `json:"count,omitempty"`
			// The total memory allocated for indexes in bytes.
			Size int64 `json:"size,omitempty"`
		} `json:"indexes"`
		ReadCache struct {
			// The number of revisions of this collection stored in the document revisions cache.
			Count int64 `json:"count,omitempty"`
			// The memory used for storing the revisions of this collection in the document revisions cache (in bytes). This figure does not include the document data but only mappings from document revision ids to cache entry locations.
			Size int64 `json:"size,omitempty"`
		} `json:"readcache"`
		// An optional string value that contains information about which object type is at the head of the collection's cleanup queue. This information can be used for debugging compaction and unload issues.
		WaitingFor string `json:"waitingFor,omitempty"`
		Alive      struct {
			// The number of currently active documents in all datafiles and journals of the collection. Documents that are contained in the write-ahead log only are not reported in this figure.
			Count int64 `json:"count,omitempty"`
			// The total size in bytes used by all active documents of the collection. Documents that are contained in the write-ahead log only are not reported in this figure.
			Size int64 `json:"size,omitempty"`
		} `json:"alive"`
		// The tick of the last marker that was stored in a journal of the collection. This might be 0 if the collection does not yet have a journal.
		LastTick int64 `json:"lastTick,omitempty"`
		Journals struct {
			// The number of journal files.
			Count int64 `json:"count,omitempty"`
			// The total filesize of all journal files (in bytes).
			FileSize int64 `json:"fileSize,omitempty"`
		} `json:"journals"`
		Revisions struct {
			// The number of revisions of this collection managed by the storage engine.
			Count int64 `json:"count,omitempty"`
			// The memory used for storing the revisions of this collection in the storage engine (in bytes). This figure does not include the document data but only mappings from document revision ids to storage engine datafile positions.
			Size int64 `json:"size,omitempty"`
		} `json:"revisions"`
	} `json:"figures"`
}

// CollectionShards contains shards information about a collection.
type CollectionShards struct {
	CollectionExtendedInfo
	// Set to create a smart edge or vertex collection.
	// This requires ArangoDB Enterprise Edition.
	IsSmart bool `json:"isSmart,omitempty"`

	// Shards is a list of shards that belong to the collection.
	// Each shard contains a list of DB servers where the first one is the leader and the rest are followers.
	Shards map[ShardID][]ServerID `json:"shards,omitempty"`

	// StatusString represents status as a string.
	StatusString string `json:"statusString,omitempty"`
}
