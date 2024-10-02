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

// DatabaseTransaction contains Streaming Transactions functions
// https://docs.arangodb.com/stable/develop/http-api/transactions/stream-transactions/
type DatabaseTransaction interface {
	ListTransactions(ctx context.Context) ([]Transaction, error)
	ListTransactionsWithStatuses(ctx context.Context, statuses ...TransactionStatus) ([]Transaction, error)

	BeginTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions) (Transaction, error)

	Transaction(ctx context.Context, id TransactionID) (Transaction, error)

	WithTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions, commitOptions *CommitTransactionOptions, abortOptions *AbortTransactionOptions, w TransactionWrap) error
}

type TransactionWrap func(ctx context.Context, t Transaction) error
