//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import (
	"context"
	"time"

	"github.com/arangodb/go-driver"
)

type DatabaseTransaction interface {
	BeginTransaction(ctx context.Context, cols driver.TransactionCollections, opts *BeginTransactionOptions) (Transaction, error)

	Transaction(ctx context.Context, id driver.TransactionID) (Transaction, error)

	WithTransaction(ctx context.Context, cols driver.TransactionCollections, opts *BeginTransactionOptions, commitOptions *driver.CommitTransactionOptions, abortOptions *driver.AbortTransactionOptions, w TransactionWrap) error
}

type TransactionWrap func(ctx context.Context, t Transaction) error

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
