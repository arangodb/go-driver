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

// Database provides access to all collections & graphs in a single database.
type Database interface {
	// Name returns the name of the database.
	Name() string

	// EngineInfo returns information about the database engine being used.
	// Note: When your cluster has multiple endpoints (cluster), you will get information
	// from the server that is currently being used.
	// If you want to know exactly which server the information is from, use a client
	// with only a single endpoint and avoid automatic synchronization of endpoints.
	EngineInfo(ctx context.Context) (EngineInfo, error)

	// Remove removes the entire database.
	// If the database does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Collection functions
	DatabaseCollections

	// Graph functions
	DatabaseGraphs

	// Query performs an AQL query, returning a cursor used to iterate over the returned documents.
	// Note that the returned Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
	Query(ctx context.Context, query string, bindVars map[string]interface{}) (Cursor, error)

	// ValidateQuery validates an AQL query.
	// When the query is valid, nil returned, otherwise an error is returned.
	// The query is not executed.
	ValidateQuery(ctx context.Context, query string) error
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
