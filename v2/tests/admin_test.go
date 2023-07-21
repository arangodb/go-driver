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
