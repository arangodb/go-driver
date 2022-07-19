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
//
// Author Adam Janikowski
//

package arangodb

import (
	"context"
	"net/http"

	"github.com/arangodb/go-driver/v2/arangodb/shared"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/connection"
)

func newTransaction(db *database, id TransactionID) *transaction {
	newDb := newDatabase(db.client, db.name, connection.WithTransactionID(string(id)))

	d := &transaction{
		database: newDb,
		id:       id,
	}

	return d
}

var _ Transaction = &transaction{}

type transaction struct {
	*database

	id TransactionID
}

func (t transaction) Commit(ctx context.Context, opts *CommitTransactionOptions) error {
	response := struct {
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallPut(ctx, t.database.connection(), t.url(), &response, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (t transaction) Abort(ctx context.Context, opts *AbortTransactionOptions) error {
	response := struct {
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallDelete(ctx, t.database.connection(), t.url(), &response)
	if err != nil {
		return errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return nil
	default:
		return response.AsArangoErrorWithCode(code)
	}
}

func (t transaction) Status(ctx context.Context) (TransactionStatusRecord, error) {
	response := struct {
		shared.ResponseStruct `json:",inline"`
		Result                TransactionStatusRecord `json:"result"`
	}{}

	resp, err := connection.CallGet(ctx, t.database.connection(), t.url(), &response)
	if err != nil {
		return TransactionStatusRecord{}, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return response.Result, nil
	default:
		return TransactionStatusRecord{}, response.AsArangoErrorWithCode(code)
	}
}

func (t transaction) ID() TransactionID {
	return t.id
}

func (t transaction) url() string {
	return t.database.url("_api", "transaction", string(t.id))
}
