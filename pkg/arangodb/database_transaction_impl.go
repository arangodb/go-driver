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
	"net/http"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/pkg/connection"
	"github.com/pkg/errors"
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

func (d databaseTransaction) WithTransaction(ctx context.Context, cols driver.TransactionCollections, opts *BeginTransactionOptions, commitOptions *driver.CommitTransactionOptions, abortOptions *driver.AbortTransactionOptions, w TransactionWrap) error {
	return d.withTransactionPanic(ctx, cols, opts, commitOptions, abortOptions, w)
}

func (d databaseTransaction) withTransactionPanic(ctx context.Context, cols driver.TransactionCollections, opts *BeginTransactionOptions, commitOptions *driver.CommitTransactionOptions, abortOptions *driver.AbortTransactionOptions, w TransactionWrap) (transactionError error) {
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
			if err = t.Commit(ctx, commitOptions); err != nil {
				transactionError = err
			}
		}
	}()

	transactionError = w(ctx, t)

	return
}

func (d databaseTransaction) BeginTransaction(ctx context.Context, cols driver.TransactionCollections, opts *BeginTransactionOptions) (Transaction, error) {
	url := d.db.url("_api", "transaction", "begin")

	input := struct {
		*BeginTransactionOptions
		Collections driver.TransactionCollections `json:"collections,omitempty"`
	}{
		BeginTransactionOptions: opts,
		Collections:             cols,
	}

	output := struct {
		ResponseStruct

		Response struct {
			TransactionID driver.TransactionID `json:"id,omitempty"`
		} `json:"result"`
	}{}

	resp, err := connection.CallPost(ctx, d.db.connection(), url, &output, &input)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusCreated:
		return newTransaction(d.db, output.Response.TransactionID), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (d databaseTransaction) Transaction(ctx context.Context, id driver.TransactionID) (Transaction, error) {
	url := d.db.url("_api", "transaction", string(id))
	resp, err := connection.CallGet(ctx, d.db.connection(), url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return newTransaction(d.db, id), nil
	default:
		return nil, connection.NewError(resp.Code(), "unexpected code")
	}
}
