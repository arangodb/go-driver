package driver_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

func TestNewClient(t *testing.T) {
	mockConn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"localhost"},
	})
	require.NoError(t, err)

	cfg := driver.ClientConfig{
		Connection:                   mockConn,
		SynchronizeEndpointsInterval: time.Second * 20,
	}

	var clients = make(map[int]driver.Client)

	before := runtime.NumGoroutine()
	for i := 0; i < 30; i++ {
		c, err := driver.NewClient(cfg)
		require.NoError(t, err, "iter %d", i)

		clients[i] = c
	}

	after := runtime.NumGoroutine()
	require.Less(t, after-before, 32)
}
