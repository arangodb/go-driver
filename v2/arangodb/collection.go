//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
)

type Collection interface {
	Name() string
	Database() Database

	// Shards fetches shards information of the collection.
	Shards(ctx context.Context, details bool) (CollectionShards, error)

	// Remove removes the entire collection.
	// If the collection does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// RemoveWithOptions removes the entire collection.
	// If the collection does not exist, a NotFoundError is returned.
	RemoveWithOptions(ctx context.Context, opts *RemoveCollectionOptions) error

	// Truncate removes all documents from the collection, but leaves the indexes intact.
	Truncate(ctx context.Context) error

	// Properties fetches extended information about the collection.
	Properties(ctx context.Context) (CollectionProperties, error)

	// SetProperties allows modifying collection parameters
	SetPropertiesV2(ctx context.Context, options SetCollectionPropertiesOptionsV2) error

	// Count fetches the number of document in the collection.
	Count(ctx context.Context) (int64, error)

	// Statistics returns the number of documents and additional statistical information about the collection.
	Statistics(ctx context.Context, details bool) (CollectionStatistics, error)

	// Revision fetches the revision ID of the collection.
	// The revision ID is a server-generated string that clients can use to check whether data
	// in a collection has changed since the last revision check.
	Revision(ctx context.Context) (CollectionProperties, error)

	// Checksum returns a checksum for the specified collection
	// withRevisions - Whether to include document revision ids in the checksum calculation.
	// withData - Whether to include document body data in the checksum calculation.
	Checksum(ctx context.Context, withRevisions bool, withData bool) (CollectionChecksum, error)

	// ResponsibleShard returns the shard responsible for the given options.
	ResponsibleShard(ctx context.Context, options map[string]interface{}) (string, error)

	// LoadIndexesIntoMemory loads all indexes of the collection into memory.
	LoadIndexesIntoMemory(ctx context.Context) (bool, error)

	// Renaming collections is not supported in cluster deployments.
	// Renaming collections is only supported in single server deployments.
	Rename(ctx context.Context, req RenameCollectionRequest) (CollectionInfo, error)

	// RecalculateCount recalculates the count of documents in the collection.
	RecalculateCount(ctx context.Context) (bool, int64, error)

	//Compacts the data of a collection in order to reclaim disk space.
	// This operation is only supported in single server deployments.
	// In cluster deployments, the compaction is done automatically by the server.
	Compact(ctx context.Context) (CollectionInfo, error)

	CollectionDocuments
	CollectionIndexes
}
