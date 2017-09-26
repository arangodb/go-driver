package driver_test

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/stretchr/testify/require"
)

type environment struct {
	ctx      context.Context
	client   driver.Client
	tearDown func()
}

func setUpTestEnvironment(tb testing.TB) *environment {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	require.NoError(tb, err)
	client, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	require.NoError(tb, err)

	return &environment{
		ctx:      context.Background(),
		client:   client,
		tearDown: func() {},
	}
}
