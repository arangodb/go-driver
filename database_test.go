package driver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseTransaction(t *testing.T) {
	e := setUpTestEnvironment(t)
	defer e.tearDown()

	db, err := e.client.CreateDatabase(e.ctx, "test", nil)
	require.NoError(t, err)
	defer db.Remove(e.ctx)

	result, err := db.Transaction(e.ctx, nil, nil, "function () { return 'worked!'; }", nil)
	require.NoError(t, err)

	assert.Equal(t, "worked!", result)
}

func TestDatabaseTransactionError(t *testing.T) {
	e := setUpTestEnvironment(t)
	defer e.tearDown()

	db, err := e.client.CreateDatabase(e.ctx, "test", nil)
	require.NoError(t, err)
	defer db.Remove(e.ctx)

	result, err := db.Transaction(e.ctx, nil, nil, "function () { error error; }", nil)
	require.Nil(t, result)
	require.Error(t, err)

	assert.Equal(t, "10: missing/invalid action definition for transaction - Uncaught SyntaxError: Unexpected identifier - SyntaxError: Unexpected identifier\n    at new Function (<anonymous>)", err.Error())
}
