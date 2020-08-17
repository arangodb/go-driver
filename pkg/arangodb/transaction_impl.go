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

func newTransaction(db *database, id driver.TransactionID) *transaction {
	newDb := newDatabase(db.client, db.name, connection.WithTransactionID(id))

	d := &transaction{
		database: newDb,
		id:       id,
	}

	return d
}

var _ Transaction = &transaction{}

type transaction struct {
	*database

	id driver.TransactionID
}

func (t transaction) Commit(ctx context.Context, opts *driver.CommitTransactionOptions) error {
	resp, err := connection.CallPut(ctx, t.database.connection(), t.url(), nil, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return nil
	default:
		return connection.NewError(resp.Code(), "unexpected code")
	}
}

func (t transaction) Abort(ctx context.Context, opts *driver.AbortTransactionOptions) error {
	resp, err := connection.CallDelete(ctx, t.database.connection(), t.url(), nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return nil
	default:
		return connection.NewError(resp.Code(), "unexpected code")
	}
}

func (t transaction) Status(ctx context.Context) (driver.TransactionStatusRecord, error) {
	response := struct {
		ResponseStruct
		Result driver.TransactionStatusRecord `json:"result"`
	}{}

	resp, err := connection.CallGet(ctx, t.database.connection(), t.url(), &response)
	if err != nil {
		return driver.TransactionStatusRecord{}, errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusOK:
		return response.Result, nil
	default:
		return driver.TransactionStatusRecord{}, connection.NewError(resp.Code(), "unexpected code")
	}
}

func (t transaction) ID() driver.TransactionID {
	return t.id
}

func (t transaction) url() string {
	return t.database.url("_api", "transaction", string(t.id))
}
