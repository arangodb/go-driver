package driver_test

import (
	"fmt"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseTransaction(t *testing.T) {
	e := setUpTestEnvironment(t)
	defer e.tearDown()

	db, err := e.client.CreateDatabase(e.ctx, "test", nil)
	require.NoError(t, err)
	defer db.Remove(e.ctx)

	testCases := []struct {
		name         string
		action       string
		options      *driver.TransactionOptions
		expectResult interface{}
		expectError  error
	}{
		{"ReturnValue", "function () { return 'worked!'; }", nil, "worked!", nil},
		{"ReturnError", "function () { error error; }", nil, nil, fmt.Errorf("10: missing/invalid action definition for transaction - Uncaught SyntaxError: Unexpected identifier - SyntaxError: Unexpected identifier\n    at new Function (<anonymous>)")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := db.Transaction(e.ctx, testCase.action, testCase.options)
			assert.Equal(t, testCase.expectResult, result)
			assert.Equal(t, testCase.expectError, err)
		})
	}
}
