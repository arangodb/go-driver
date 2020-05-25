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

package connection

import "github.com/arangodb/go-driver"

func WithTransactionID(transactionID driver.TransactionID) RequestModifier {
	return func(r Request) error {
		r.AddHeader("x-arango-trx-id", string(transactionID))
		return nil
	}
}

func WithFragment(s string) RequestModifier {
	return func(r Request) error {
		r.SetFragment(s)
		return nil
	}
}

func WithQuery(s, value string) RequestModifier {
	return func(r Request) error {
		r.AddQuery(s, value)
		return nil
	}
}
