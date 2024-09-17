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

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

func newDatabaseTransaction(db *database) *databaseTransaction {
	return &databaseTransaction{
		db: db,
	}
}

var _ DatabaseTransaction = &databaseTransaction{}

type databaseTransaction struct {
	db *database
}

func (d databaseTransaction) ListTransactions(ctx context.Context) ([]Transaction, error) {
	return d.ListTransactionsWithStatuses(ctx, TransactionRunning, TransactionCommitted, TransactionAborted)
}

func (d databaseTransaction) ListTransactionsWithStatuses(ctx context.Context, statuses ...TransactionStatus) ([]Transaction, error) {
	return d.listTransactionsWithStatuses(ctx, statuses)
}

func (d databaseTransaction) listTransactionsWithStatuses(ctx context.Context, statuses TransactionStatuses) ([]Transaction, error) {
	url := d.db.url("_api", "transaction")

	var result struct {
		Transactions []struct {
			ID    TransactionID     `json:"id"`
			State TransactionStatus `json:"state"`
		} `json:"transactions,omitempty"`
	}

	var response shared.ResponseStruct

	resp, err := connection.CallGet(ctx, d.db.connection(), url, newMultiUnmarshaller(&result, &response), d.db.modifiers...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		var t []Transaction
		for _, r := range result.Transactions {
			if !statuses.Contains(r.State) {
				continue
			}

			t = append(t, newTransaction(d.db, r.ID))
		}

		return t, nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}

func (d databaseTransaction) WithTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions, commitOptions *CommitTransactionOptions, abortOptions *AbortTransactionOptions, w TransactionWrap) error {
	return d.withTransactionPanic(ctx, cols, opts, commitOptions, abortOptions, w)
}

func (d databaseTransaction) withTransactionPanic(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions, commitOptions *CommitTransactionOptions, abortOptions *AbortTransactionOptions, w TransactionWrap) (transactionError error) {
	t, err := d.BeginTransaction(ctx, cols, opts)
	if err != nil {
		return err
	}

	transactionError = nil

	defer func() {
		if transactionError != nil {
			if err = t.Abort(ctx, abortOptions); err != nil {
				transactionError = errors.Wrapf(transactionError, "Transaction abort failed with %s", err.Error())
			}
		} else {
			if p := recover(); p != nil {
				if err = t.Abort(ctx, abortOptions); err != nil {
					transactionError = errors.Wrapf(transactionError, "Transaction abort failed with %s", err.Error())
				}
				panic(p)
			}
			if err = t.Commit(ctx, commitOptions); err != nil {
				transactionError = err
			}
		}
	}()

	transactionError = w(ctx, t)

	return
}

func (d databaseTransaction) BeginTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions) (Transaction, error) {
	url := d.db.url("_api", "transaction", "begin")

	input := struct {
		*BeginTransactionOptions
		Collections TransactionCollections `json:"collections,omitempty"`
	}{
		BeginTransactionOptions: opts,
		Collections:             cols,
	}

	output := struct {
		shared.ResponseStruct `json:",inline"`
		Response              struct {
			TransactionID TransactionID `json:"id,omitempty"`
		} `json:"result"`
	}{}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &output, &input, append(d.db.modifiers, opts.modifyRequest)...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated:
		return newTransaction(d.db, output.Response.TransactionID), nil
	default:
		return nil, output.AsArangoErrorWithCode(code)
	}
}

func (d databaseTransaction) Transaction(ctx context.Context, id TransactionID) (Transaction, error) {
	url := d.db.url("_api", "transaction", string(id))

	var response struct {
		shared.ResponseStruct `json:",inline"`
	}

	resp, err := connection.CallGet(ctx, d.db.connection(), url, &response, d.db.modifiers...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return newTransaction(d.db, id), nil
	default:
		return nil, response.AsArangoErrorWithCode(code)
	}
}
