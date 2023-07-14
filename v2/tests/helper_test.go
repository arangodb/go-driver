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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func WithDatabase(t testing.TB, client arangodb.Client, opts *arangodb.CreateDatabaseOptions, f func(db arangodb.Database)) {
	name := fmt.Sprintf("test-DB-%s", uuid.New().String())

	info(t)("Creating DB %s", name)

	withContextT(t, 2*time.Minute, func(ctx context.Context, _ testing.TB) {
		db, err := client.CreateDatabase(ctx, name, opts)
		require.NoError(t, err)

		defer func() {
			withContextT(t, 2*time.Minute, func(ctx context.Context, _ testing.TB) {
				info(t)("Removing DB %s", db.Name())
			})
		}()

		f(db)
	})
}

func WithCollection(t testing.TB, db arangodb.Database, opts *arangodb.CreateCollectionOptions, f func(col arangodb.Collection)) {
	name := fmt.Sprintf("test-COL-%s", uuid.New().String())

	info(t)("Creating COL %s", name)

	withContextT(t, 2*time.Minute, func(ctx context.Context, _ testing.TB) {
		col, err := db.CreateCollection(ctx, name, opts)
		require.NoError(t, err)

		NewTimeout(func() error {
			_, err := db.Collection(ctx, name)
			if err == nil {
				return Interrupt{}
			}

			if shared.IsNotFound(err) {
				return nil
			}

			return err
		}).TimeoutT(t, 15*time.Second, 125*time.Millisecond)

		f(col)
	})
}

func WithUserDocs(t *testing.T, col arangodb.Collection, f func(users []UserDoc)) {
	users := []UserDoc{
		{Name: "John", Age: 13},
		{Name: "Jake", Age: 25},
		{Name: "Clair", Age: 12},
		{Name: "Johnny", Age: 42},
		{Name: "Blair", Age: 67},
	}

	withContextT(t, 2*time.Minute, func(ctx context.Context, tb testing.TB) {
		_, err := col.CreateDocuments(ctx, users)
		require.NoError(t, err)

		f(users)
	})
}

func getBool(b *bool, d bool) bool {
	if b == nil {
		return d
	}

	return *b
}

func newBool(b bool) *bool {
	return &b
}
