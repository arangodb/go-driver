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

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func WithDatabase(t testing.TB, client arangodb.Client, opts *arangodb.CreateDatabaseOptions, f func(db arangodb.Database)) {
	name := fmt.Sprintf("test-%s", uuid.New().String())

	info(t)("Creating DB %s", name)

	withContext(2*time.Minute, func(ctx context.Context) error {
		db, err := client.CreateDatabase(ctx, name, opts)
		require.NoError(t, err)

		defer func() {
			withContext(2*time.Minute, func(ctx context.Context) error {
				info(t)("Removing DB %s", db.Name())
				return nil
			})
		}()

		f(db)

		return nil
	})
}

func WithCollection(t testing.TB, db arangodb.Database, opts *arangodb.CreateCollectionOptions, f func(col arangodb.Collection)) {
	name := fmt.Sprintf("test-%s", uuid.New().String())

	info(t)("Creating COL %s", name)

	withContext(2*time.Minute, func(ctx context.Context) error {
		col, err := db.CreateCollection(ctx, name, opts)
		require.NoError(t, err)

		NewTimeout(func() error {
			_, err := db.Collection(ctx, name)
			if err == nil {
				return Interrupt{}
			}

			if arangodb.IsNotFound(err) {
				return nil
			}

			return err
		}).TimeoutT(t, 15*time.Second, 125*time.Millisecond)

		defer func() {
			withContext(2*time.Minute, func(ctx context.Context) error {
				info(t)("Removing COL %s", name)
				require.NoError(t, col.Remove(ctx))
				return nil
			})
		}()

		f(col)

		return nil
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
