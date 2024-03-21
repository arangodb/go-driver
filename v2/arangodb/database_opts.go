//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

// DatabaseInfo contains information about a database
type DatabaseInfo struct {
	// The identifier of the database.
	ID string `json:"id,omitempty"`

	// The name of the database.
	Name string `json:"name,omitempty"`

	// The filesystem path of the database.
	Path string `json:"path,omitempty"`

	// If true then the database is the _system database.
	IsSystem bool `json:"isSystem,omitempty"`

	// Default replication factor for collections in database
	ReplicationFactor ReplicationFactor `json:"replicationFactor,omitempty"`

	// Default write concern for collections in database
	WriteConcern int `json:"writeConcern,omitempty"`

	// Default sharding for collections in database
	Sharding DatabaseSharding `json:"sharding,omitempty"`

	// Replication version used for this database
	ReplicationVersion DatabaseReplicationVersion `json:"replicationVersion,omitempty"`
}

// EngineType indicates type of database engine being used.
type EngineType string

const (
	EngineTypeMMFiles = EngineType("mmfiles")
	EngineTypeRocksDB = EngineType("rocksdb")
)

func (t EngineType) String() string {
	return string(t)
}

// EngineInfo contains information about the database engine being used.
type EngineInfo struct {
	Type EngineType `json:"name"`
}
