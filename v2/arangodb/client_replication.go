//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

import "context"

// ClientReplication defines replication API methods.
type ClientReplication interface {
	// CreateNewBatch creates a new replication batch.
	CreateNewBatch(ctx context.Context, dbName string, DBserver *string, state *bool, opt CreateNewBatchOptions) (CreateNewBatchResponse, error)
}

// CreateNewBatchOptions represents the request body for creating a batch.
type CreateNewBatchOptions struct {
	Ttl int `json:"ttl"`
}

// CreateNewBatchResponse represents the response for batch creation.
type CreateNewBatchResponse struct {
	// The ID of the created batch
	ID string `json:"id"`
	// The last tick of the created batch
	LastTick string `json:"lastTick"`
	// Only present if the state URL parameter was set to true
	State map[string]interface{} `json:"state,omitempty"`
}
