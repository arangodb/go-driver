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

package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_DatabaseTransactionsJS(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollectionV2(t, db, nil, func(col arangodb.Collection) {

				t.Run("Transaction ReturnValue", func(t *testing.T) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						txJSOptions := arangodb.TransactionJSOptions{
							Action: "function () { return 'worked!'; }",
							Collections: arangodb.TransactionCollections{
								Read:      []string{col.Name()},
								Write:     []string{col.Name()},
								Exclusive: []string{col.Name()},
							},
						}

						result, err := db.TransactionJS(ctx, txJSOptions)
						require.NoError(t, err)
						require.Equal(t, "worked!", result)
					})
				})

				t.Run("Transaction ReturnError", func(t *testing.T) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						txJSOptions := arangodb.TransactionJSOptions{
							Action: "function () { error error; }",
							Collections: arangodb.TransactionCollections{
								Read:      []string{col.Name()},
								Write:     []string{col.Name()},
								Exclusive: []string{col.Name()},
							},
						}

						_, err := db.TransactionJS(ctx, txJSOptions)
						require.Error(t, err)

						const expectedStr = "Uncaught SyntaxError: Unexpected identifier"
						substrFound := strings.Index(err.Error(), expectedStr) >= 0
						require.Truef(t, substrFound, "expected error to contain '%v', got '%v'", expectedStr, err.Error())
					})
				})

				t.Run("Transaction - fetching command line options ", func(t *testing.T) {
					withContextT(t, defaultTestTimeout, func(ctx context.Context, t testing.TB) {
						txJSOptions := arangodb.TransactionJSOptions{
							Action: `function () { return require("internal").options(); }`,
							Collections: arangodb.TransactionCollections{
								Read:      []string{col.Name()},
								Write:     []string{col.Name()},
								Exclusive: []string{col.Name()},
							},
						}

						result, err := db.TransactionJS(ctx, txJSOptions)
						require.NoError(t, err)

						optionsMap, ok := result.(map[string]interface{})
						require.True(t, ok)
						require.Equal(t, false, optionsMap["cluster.force-one-shard"])
					})

				})
			})
		})
	})
}
