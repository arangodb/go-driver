package driver_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

// TestMain creates a simple connection and waits for the server to be ready.
// This avoid a lot of clutter code in the examples.
func TestMain(m *testing.M) {
	// Wait for database connection to be ready.
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})

	waitUntilServerAvailable(context.Background(), c)

	os.Exit(m.Run())
}

// waitUntilServerAvailable keeps waiting until the server/cluster that the client is addressing is available.
func waitUntilServerAvailable(ctx context.Context, c driver.Client) bool {
	instanceUp := make(chan bool)
	go func() {
		for {
			verCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if _, err := c.Version(verCtx); err == nil {
				cancel()
				instanceUp <- true
				return
			} else {
				cancel()
				time.Sleep(time.Second)
			}
		}
	}()
	select {
	case up := <-instanceUp:
		return up
	case <-ctx.Done():
		return false
	}
}
