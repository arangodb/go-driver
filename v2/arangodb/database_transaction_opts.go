//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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
	"time"
)

// BeginTransactionOptions provides options for BeginTransaction call
type BeginTransactionOptions struct {
	WaitForSync         bool          `json:"waitForSync,omitempty"`
	AllowImplicit       bool          `json:"allowImplicit,omitempty"`
	LockTimeoutDuration time.Duration `json:"-"`
	LockTimeout         float64       `json:"lockTimeout,omitempty"`
	MaxTransactionSize  uint64        `json:"maxTransactionSize,omitempty"`
}

func (b *BeginTransactionOptions) set() *BeginTransactionOptions {
	if b.LockTimeoutDuration != 0 && b.LockTimeout == 0 {
		b.LockTimeout = float64(b.LockTimeoutDuration) / float64(time.Second)
	}

	return b
}

// TransactionCollections is used to specify which collections are accessed by a transaction and how
type TransactionCollections struct {
	// Collections that the transaction reads from.
	Read []string `json:"read,omitempty"`
	// Collections that the transaction writes to.
	Write []string `json:"write,omitempty"`
	// Collections that the transaction writes exclusively to.
	Exclusive []string `json:"exclusive,omitempty"`
}

// CommitTransactionOptions provides options for CommitTransaction. Currently unused
type CommitTransactionOptions struct{}

// AbortTransactionOptions provides options for CommitTransaction. Currently unused
type AbortTransactionOptions struct{}

// TransactionID identifies a transaction
type TransactionID string

// TransactionStatuses list of transaction statuses
type TransactionStatuses []TransactionStatus

func (t TransactionStatuses) Contains(status TransactionStatus) bool {
	for _, i := range t {
		if i == status {
			return true
		}
	}
	return false
}

// TransactionStatus describes the status of an transaction
type TransactionStatus string

const (
	TransactionRunning   TransactionStatus = "running"
	TransactionCommitted TransactionStatus = "committed"
	TransactionAborted   TransactionStatus = "aborted"
)

// TransactionStatusRecord provides insight about the status of transaction
type TransactionStatusRecord struct {
	Status TransactionStatus
}

// TransactionJSOptions contains options that customize the JavaScript transaction
type TransactionJSOptions struct {
	// The actual transaction operations to be executed, in the form of stringified JavaScript code
	Action string `json:"action"`

	// An optional boolean flag that, if set, will force the transaction to write
	// all data to disk before returning.
	WaitForSync *bool `json:"waitForSync,omitempty"`

	// Allow reading from undeclared collections.
	AllowImplicit *bool `json:"allowImplicit,omitempty"`

	// An optional numeric value that can be used to set a timeout for waiting on collection locks.
	// If not specified, a default value will be used.
	// Setting lockTimeout to 0 will make ArangoDB not time out waiting for a lock.
	LockTimeout *int `json:"lockTimeout,omitempty"`

	// Optional arguments passed to action.
	Params []string `json:"params,omitempty"`

	// Transaction size limit in bytes. Honored by the RocksDB storage engine only.
	MaxTransactionSize *int `json:"maxTransactionSize,omitempty"`

	Collections TransactionCollections `json:"collections"`
}
