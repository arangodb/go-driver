package test

import (
	"context"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
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
	requireNoError(tb, err)
	client, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	requireNoError(tb, err)

	return &environment{
		ctx:      context.Background(),
		client:   client,
		tearDown: func() {},
	}
}

func requireNoError(tb testing.TB, err error) {
	if err != nil {
		tb.Fatalf("expected no error, got %v", err)
	}
}

func assertEqual(tb testing.TB, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		tb.Errorf("expected %v, got %v", expected, actual)
	}
}
