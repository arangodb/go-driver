package test

import (
	"fmt"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
)

func TestDatabaseTransaction(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipBelowVersion(c, "3.2", t)
	db := ensureDatabase(nil, c, "transaction_test", nil, t)

	testCases := []struct {
		name         string
		action       string
		options      *driver.TransactionOptions
		expectResult interface{}
		expectError  error
	}{
		{"ReturnValue", "function () { return 'worked!'; }", nil, "worked!", nil},
		{"ReturnError", "function () { error error; }", nil, nil, fmt.Errorf("missing/invalid action definition for transaction - Uncaught SyntaxError: Unexpected identifier - SyntaxError: Unexpected identifier\n    at new Function (<anonymous>)")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := db.Transaction(nil, testCase.action, testCase.options)
			if !reflect.DeepEqual(testCase.expectResult, result) {
				t.Errorf("expected result %v, got %v", testCase.expectResult, result)
			}
			if testCase.expectError != nil {
				if testCase.expectError.Error() != err.Error() {
					t.Errorf("expected error %v, got %v", testCase.expectError.Error(), err.Error())
				}
			}
		})
	}
}
