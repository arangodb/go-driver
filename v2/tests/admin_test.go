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

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
)

func Test_ServerMode(t *testing.T) {
	// This test can not run sub-tests parallelly, because it changes admin settings.
	wrapOpts := WrapOptions{
		Parallel: newBool(false),
	}

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			serverMode, err := client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeReadOnly)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeReadOnly, serverMode)

			err = client.SetServerMode(ctx, arangodb.ServerModeDefault)
			require.NoError(t, err)

			serverMode, err = client.ServerMode(ctx)
			require.NoError(t, err)
			require.Equal(t, arangodb.ServerModeDefault, serverMode)
		})
	}, wrapOpts)
}

func Test_ServerID(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, time.Minute, func(ctx context.Context, t testing.TB) {
			id, err := client.ServerID(ctx)
			require.NoError(t, err)
			require.NotEmpty(t, id)
		})
	})
}
