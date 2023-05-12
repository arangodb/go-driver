//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

// Test_ExplainQuery tries to explain several AQL queries.
func Test_ExplainQuery(t *testing.T) {
	rf := arangodb.ReplicationFactor(2)
	options := arangodb.CreateCollectionOptions{
		ReplicationFactor: rf,
		NumberOfShards:    2,
	}
	ctx := context.Background()
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, &options, func(col arangodb.Collection) {
				// Setup tests
				tests := []struct {
					Query         string
					BindVars      map[string]interface{}
					Opts          *arangodb.ExplainQueryOptions
					ExpectSuccess bool
				}{
					{
						Query:         fmt.Sprintf("FOR d IN `%s` SORT d.Title RETURN d", col.Name()),
						ExpectSuccess: true,
					},
					{
						Query: fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						BindVars: map[string]interface{}{
							"title": "Defending the Undefendable",
						},
						ExpectSuccess: true,
					},
					{
						Query: fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						BindVars: map[string]interface{}{
							"title": "Democracy: God That Failed",
						},
						Opts: &arangodb.ExplainQueryOptions{
							AllPlans:  true,
							Optimizer: arangodb.ExplainQueryOptimizerOptions{},
						},
						ExpectSuccess: true,
					},
					{
						Query:         fmt.Sprintf("FOR d IN `%s` FILTER d.Title==@title SORT d.Title RETURN d", col.Name()),
						ExpectSuccess: false, // bindVars not provided
					},
					{
						Query:         fmt.Sprintf("FOR u IN `%s` FILTER u.age>>>100 SORT u.name RETURN u", col.Name()),
						ExpectSuccess: false, // syntax error
					},
					{
						Query:         "",
						ExpectSuccess: false,
					},
				}
				for i, test := range tests {
					t.Run(fmt.Sprintf("Case %d", i), func(t *testing.T) {
						_, err := db.ExplainQuery(ctx, test.Query, test.BindVars, test.Opts)
						if test.ExpectSuccess {
							require.NoError(t, err, "case %d", i)
						} else {
							require.Error(t, err, "case %d", i)
						}
					})
				}
			})
		})
	})
}
