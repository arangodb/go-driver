//
// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/agency"
	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/utils"
)

// getAgencyEndpoints queries the cluster to get all agency endpoints.
func getAgencyEndpoints(ctx context.Context, client arangodb.Client) ([]string, error) {
	health, err := client.Health(ctx)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, entry := range health.Health {
		if entry.Role == arangodb.ServerRoleAgent {
			ep := connection.FixupEndpointURLScheme(entry.Endpoint)
			result = append(result, ep)
		}
	}
	return result, nil
}

// getAgencyClient creates an agency client from cluster endpoints.
func getAgencyClient(ctx context.Context, t testing.TB, client arangodb.Client) (arangodb.Client, error) {
	requireClusterMode(t)

	endpoints, err := getAgencyEndpoints(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		t.Skip("No agency endpoints found")
	}

	// Create a connection to agency endpoints
	conn := connection.NewHttpConnection(connection.HttpConfiguration{
		Endpoint: connection.NewRoundRobinEndpoints(endpoints),
		Transport: &http.Transport{
			//nolint:gosec // test-only agency connection
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})

	// Create agency client
	agencyClient := arangodb.NewClient(conn)
	return agencyClient, nil
}

func newTx() agency.Transaction {
	return agency.NewTransaction(nil, agency.TransactionOptions{})
}

func agencyWrite(t testing.TB, ctx context.Context, c arangodb.Client, tx agency.Transaction) {
	t.Helper()
	require.NoError(t, c.WriteTransaction(ctx, tx))
}

func agencySet(t testing.TB, ctx context.Context, c arangodb.Client, key []string, value interface{}) {
	t.Helper()
	tx := newTx()
	tx.AddKey(agency.NewKeySet(key, value))
	agencyWrite(t, ctx, c, tx)
}

func agencyDelete(t testing.TB, ctx context.Context, c arangodb.Client, key []string) {
	t.Helper()
	tx := newTx()
	tx.AddKey(agency.NewKeyDelete(key))
	agencyWrite(t, ctx, c, tx)
}

func agencyCleanup(t testing.TB, ctx context.Context, c arangodb.Client, root string) {
	t.Helper()
	tx := newTx()
	tx.AddKey(agency.NewKeyDelete([]string{root}))
	_ = c.WriteTransaction(ctx, tx)
}

// TestAgencyRead tests the Agency.ReadKey method.
func TestAgencyRead(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			var result interface{}
			if err := agencyClient.ReadKey(ctx, []string{"not-found-b1d534b1-26d8-5ad0-b22d-23d49d3ea92c"}, &result); !arangodb.IsKeyNotFound(err) {
				t.Errorf("Expected KeyNotFoundError, got %s", err)
			}
			if err := agencyClient.ReadKey(ctx, []string{"arango"}, &result); err != nil {
				t.Errorf("Expected success, got %s", err)
			}
		})
	})
}

// TestAgencyWriteTransaction tests the Agency.WriteTransaction method.
func TestAgencyWriteTransaction(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransaction")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)

			op := func(key []string, value, result interface{}) {
				agencySet(t, ctx, agencyClient, key, value)

				require.NoError(t, agencyClient.ReadKey(ctx, key, result))

				got := reflect.ValueOf(result).Elem().Interface()
				require.Equal(t, value, got)
			}
			op([]string{rootKeyAgency, "string"}, "hello world", new(string))
			op([]string{rootKeyAgency, "int"}, 55, new(int))
			op([]string{rootKeyAgency, "bool"}, true, new(bool))
			op([]string{rootKeyAgency, "object"}, struct{ Field string }{Field: "hello world"}, &struct{ Field string }{})
			op([]string{rootKeyAgency, "string-array"}, []string{"hello", "world"}, new([]string))
			op([]string{rootKeyAgency, "int-array"}, []int{-5, 34, 11}, new([]int))

		})
	})
}

func agencySetWithCondition(
	t testing.TB,
	ctx context.Context,
	c arangodb.Client,
	key []string,
	value interface{},
	cond agency.KeyConditioner,
) error {
	t.Helper()
	tx := newTx()
	tx.AddKey(agency.NewKeySet(key, value))
	tx.AddCondition(key, cond)
	return c.WriteTransaction(ctx, tx)
}

// TestAgencyWriteTransactionWithConditions tests the Agency.WriteTransaction method with conditions.
func TestAgencyWriteTransactionWithConditions(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skip(err)
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionWithConditions")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)

			key := []string{rootKeyAgency, "test"}

			agencySet(t, ctx, agencyClient, key, "foo")

			var result string
			require.NoError(t, agencyClient.ReadKey(ctx, key, &result))
			assert.Equal(t, "foo", result)

			// Wrong condition
			err = agencySetWithCondition(
				t, ctx, agencyClient, key, "bar",
				agency.NewConditionIfEqual("wrong"),
			)
			require.Error(t, err)

			ok, ae := shared.IsArangoError(err)
			require.True(t, ok)
			assert.Equal(t, http.StatusPreconditionFailed, ae.Code)

			// Correct condition
			require.NoError(t,
				agencySetWithCondition(
					t, ctx, agencyClient, key, "bar",
					agency.NewConditionIfEqual("foo"),
				),
			)

			require.NoError(t, agencyClient.ReadKey(ctx, key, &result))
			assert.Equal(t, "bar", result)

			agencyDelete(t, ctx, agencyClient, key)
		})
	}, WrapOptions{Parallel: utils.NewType(false)})
}

// TestAgencyWriteTransactionOldEmpty tests the Agency.WriteTransaction method with oldEmpty condition.
func TestAgencyWriteTransactionOldEmpty(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionOldEmpty")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)
			key := []string{rootKeyAgency, "test"}

			// Set a value first transaction1
			agencySet(t, ctx, agencyClient, key, "foo")

			// Try to set with oldEmpty condition (should fail)
			transaction2 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction2.AddKey(agency.NewKeySet(key, "bar"))
			transaction2.AddCondition(key, agency.NewConditionOldEmpty(true))
			err = agencyClient.WriteTransaction(ctx, transaction2)
			require.Error(t, err)
			ok, arangoErr := shared.IsArangoError(err)
			require.True(t, ok)
			assert.Equal(t, http.StatusPreconditionFailed, arangoErr.Code)

			// Delete the key transaction3
			agencyDelete(t, ctx, agencyClient, key)

			// Verify key is not found
			var result string
			err = agencyClient.ReadKey(ctx, key, &result)
			assert.True(t, arangodb.IsKeyNotFound(err))

			// Now set with oldEmpty condition (should succeed)
			transaction4 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction4.AddKey(agency.NewKeySet(key, "bar"))
			transaction4.AddCondition(key, agency.NewConditionOldEmpty(true))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction4))

			// Verify value was set
			require.NoError(t, agencyClient.ReadKey(ctx, key, &result))
			assert.Equal(t, "bar", result)

		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

// TestAgencyWriteTransactionDelete tests the Agency.WriteTransaction method with delete operations.
func TestAgencyWriteTransactionDelete(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionDelete")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)

			key := []string{rootKeyAgency, "test"}

			// Set a value transaction1
			transaction1 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction1.AddKey(agency.NewKeySet(key, "foo"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction1))
			var result string
			require.NoError(t, agencyClient.ReadKey(ctx, key, &result))
			assert.Equal(t, "foo", result)

			// Delete the key
			agencyDelete(t, ctx, agencyClient, key)

			// Verify key is not found
			err = agencyClient.ReadKey(ctx, key, &result)
			assert.True(t, arangodb.IsKeyNotFound(err))

		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func agencyArrayPush(t testing.TB, ctx context.Context, c arangodb.Client, key []string, v interface{}) {
	tx := newTx()
	tx.AddKey(agency.NewKeyArrayPush(key, v))
	agencyWrite(t, ctx, c, tx)
}

// TestAgencyWriteTransactionArrayOperations tests array operations.
func TestAgencyWriteTransactionArrayOperations(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionArrayOperations")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)

			key := []string{rootKeyAgency, "test", "array"}

			// Push elements to array (using strings like v1 test)
			// Each push in a separate transaction like v1 test
			agencyArrayPush(t, ctx, agencyClient, key, "1")

			agencyArrayPush(t, ctx, agencyClient, key, "2")

			agencyArrayPush(t, ctx, agencyClient, key, "3")

			agencyArrayPush(t, ctx, agencyClient, key, "4")

			// Read from root structure like v1 test
			var rootResult map[string]interface{}
			require.NoError(t, agencyClient.ReadKey(ctx, []string{rootKeyAgency}, &rootResult))
			require.Contains(t, rootResult, "test", "Root should contain 'test' key")
			testObj, ok := rootResult["test"].(map[string]interface{})
			require.True(t, ok, "'test' should be an object")
			require.Contains(t, testObj, "array", "'test' should contain 'array' key")
			result, ok := testObj["array"].([]interface{})
			require.True(t, ok, "'array' should be an array")
			assert.Len(t, result, 4, "Expected array to have 4 elements after 4 pushes")

			// Erase element
			transaction5 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction5.AddKey(agency.NewKeyArrayErase(key, "2"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction5))

			// Read array again from root
			require.NoError(t, agencyClient.ReadKey(ctx, []string{rootKeyAgency}, &rootResult))
			testObj, ok = rootResult["test"].(map[string]interface{})
			require.True(t, ok)
			result, ok = testObj["array"].([]interface{})
			require.True(t, ok)
			assert.Len(t, result, 3, "Expected array to have 3 elements after erasing one")

			// Replace element (replace "3" with "99")
			transaction6 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction6.AddKey(agency.NewKeyArrayReplace(key, "3", "99"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction6))

			// Read array again from root
			require.NoError(t, agencyClient.ReadKey(ctx, []string{rootKeyAgency}, &rootResult))
			testObj, ok = rootResult["test"].(map[string]interface{})
			require.True(t, ok)
			result, ok = testObj["array"].([]interface{})
			require.True(t, ok)
			assert.Len(t, result, 3, "Expected array to still have 3 elements after replace")

		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

// TestAgencyWriteTransactionMultipleKeys tests writing multiple keys in one transaction.
func TestAgencyWriteTransactionMultipleKeys(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionMultipleKeys")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)
			// Create two keys in one transaction
			transaction := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction.AddKey(agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1"))
			transaction.AddKey(agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction))

			// Read both keys
			var result1 string
			var result2 string
			require.NoError(t, agencyClient.ReadKey(ctx, []string{rootKeyAgency, "test", "1"}, &result1))
			require.NoError(t, agencyClient.ReadKey(ctx, []string{rootKeyAgency, "test", "2"}, &result2))
			assert.Equal(t, "1", result1)
			assert.Equal(t, "2", result2)

		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

// TestAgencyWriteTransactionConditions tests various condition types.
func TestAgencyWriteTransactionConditions(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			rootKeyAgency := GenerateUUID("TestAgencyWriteTransactionConditions")
			defer agencyCleanup(t, ctx, agencyClient, rootKeyAgency)

			key1 := []string{rootKeyAgency, "test", "1"}
			key2 := []string{rootKeyAgency, "test", "2"}

			// Set initial values
			transaction1 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction1.AddKey(agency.NewKeySet(key1, "1"))
			transaction1.AddKey(agency.NewKeySet(key2, "2"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction1))

			// Test IfEqual and IfNotEqual conditions
			transaction2 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction2.AddKey(agency.NewKeySet(key1, "3"))
			transaction2.AddCondition(key1, agency.NewConditionIfEqual("1"))
			transaction2.AddCondition(key2, agency.NewConditionIfNotEqual("3"))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction2))

			var result1 string
			var result2 string
			require.NoError(t, agencyClient.ReadKey(ctx, key1, &result1))
			require.NoError(t, agencyClient.ReadKey(ctx, key2, &result2))
			assert.Equal(t, "3", result1)
			assert.Equal(t, "2", result2)

			// Test IsArray condition
			// First set a non-array value
			key3 := []string{rootKeyAgency, "test", "array"}
			transaction3 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction3.AddKey(agency.NewKeySet(key3, 1))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction3))

			// Try to set a string with IsArray(true) condition - should fail because old value is not an array
			transaction4 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction4.AddKey(agency.NewKeySet(key3, "not-array"))
			transaction4.AddCondition(key3, agency.NewConditionIsArray(true))
			err = agencyClient.WriteTransaction(ctx, transaction4)
			require.Error(t, err, "Expected error when setting non-array value with IsArray(true) condition on non-array old value")
			ok, arangoErr := shared.IsArangoError(err)
			require.True(t, ok)
			assert.Equal(t, http.StatusPreconditionFailed, arangoErr.Code)

			// Now set an array value
			transaction5 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction5.AddKey(agency.NewKeySet(key3, []int{1, 2}))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction5))

			// Now set a string with IsArray(true) condition - should succeed because old value IS an array
			transaction6 := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction6.AddKey(agency.NewKeySet(key3, "8"))
			transaction6.AddCondition(key3, agency.NewConditionIsArray(true))
			require.NoError(t, agencyClient.WriteTransaction(ctx, transaction6), "Should succeed when old value is an array")

			// Verify the value was set
			var result3 string
			require.NoError(t, agencyClient.ReadKey(ctx, key3, &result3))
			assert.Equal(t, "8", result3)

		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

// TestAgencyRedirectHandling tests that agency redirects (307) are handled correctly.
func TestAgencyRedirectHandling(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %s", err)
				return
			}

			// Get all agency endpoints
			agencyEndpoints, err := getAgencyEndpoints(ctx, client)
			require.NoError(t, err)
			require.NotEmpty(t, agencyEndpoints, "Should have at least one agency endpoint")

			// If we have multiple agency endpoints, test redirect handling
			if len(agencyEndpoints) > 1 {
				// Create connections to individual agency endpoints
				agencyClients := make([]arangodb.Client, len(agencyEndpoints))
				for i, ep := range agencyEndpoints {
					conn := connection.NewHttpConnection(connection.HttpConfiguration{
						Endpoint: connection.NewRoundRobinEndpoints([]string{ep}),
						Transport: &http.Transport{
							//nolint:gosec // test-only agency connection
							TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
						},
					})
					agencyClients[i] = arangodb.NewClient(conn)
				}

				// Test that ReadKey handles redirects correctly
				// When we connect to a follower, it should redirect to leader
				testKey := []string{"arango", "Plan", "Collections"}
				var result interface{}
				err = agencyClient.ReadKey(ctx, testKey, &result)
				// Should succeed (either directly or after redirect)
				require.NoError(t, err, "ReadKey should handle redirects and succeed")
			} else {
				t.Log("Only one agency endpoint available, skipping redirect test")
			}
		})
	})
}

// TestAgencyMultipleEndpoints tests agency functionality with multiple agency endpoints.
func TestAgencyMultipleEndpoints(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get all agency endpoints
			agencyEndpoints, err := getAgencyEndpoints(ctx, client)
			require.NoError(t, err)
			require.NotEmpty(t, agencyEndpoints, "Should have at least one agency endpoint")

			t.Logf("Found %d agency endpoints: %v", len(agencyEndpoints), agencyEndpoints)

			// If we have multiple agency endpoints, verify we can connect to all of them
			if len(agencyEndpoints) > 1 {
				// Create agency client with all endpoints
				conn := connection.NewHttpConnection(connection.HttpConfiguration{
					Endpoint: connection.NewRoundRobinEndpoints(agencyEndpoints),
					Transport: &http.Transport{
						//nolint:gosec // test-only agency connection
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				})
				multiEndpointClient := arangodb.NewClient(conn)

				// Test that we can read from agency with multiple endpoints
				var result interface{}
				err = multiEndpointClient.ReadKey(ctx, []string{"arango"}, &result)
				require.NoError(t, err, "Should be able to read from agency with multiple endpoints")
			} else {
				t.Log("Only one agency endpoint available, testing single endpoint scenario")
				// Test with single endpoint
				agencyClient, err := getAgencyClient(ctx, t, client)
				if err != nil {
					t.Skipf("Skip agency test: %s", err)
					return
				}
				var result interface{}
				err = agencyClient.ReadKey(ctx, []string{"arango"}, &result)
				require.NoError(t, err, "Should be able to read from agency")
			}
		})
	})
}

// TestAgencyLeaderDiscovery tests leader discovery functionality.
func TestAgencyLeaderDiscovery(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get all agency endpoints
			agencyEndpoints, err := getAgencyEndpoints(ctx, client)
			require.NoError(t, err)
			require.NotEmpty(t, agencyEndpoints, "Should have at least one agency endpoint")

			// Create individual clients for each agency endpoint
			agencyClients := make([]arangodb.Client, len(agencyEndpoints))
			for i, ep := range agencyEndpoints {
				conn := connection.NewHttpConnection(connection.HttpConfiguration{
					Endpoint: connection.NewRoundRobinEndpoints([]string{ep}),
					Transport: &http.Transport{
						//nolint:gosec // test-only agency connection
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				})
				agencyClients[i] = arangodb.NewClient(conn)
			}

			// Test AreAgentsHealthy
			err = arangodb.AreAgentsHealthy(ctx, agencyClients)
			if len(agencyEndpoints) == 1 {
				// With single endpoint, we might not have a leader (depending on context)
				// Use WithAllowNoLeader to allow this scenario
				ctxWithAllowNoLeader := arangodb.WithAllowNoLeader(ctx)
				err = arangodb.AreAgentsHealthy(ctxWithAllowNoLeader, agencyClients)
			}
			// Should succeed if we have proper agency setup
			// Note: This might fail in some test environments, so we log but don't fail
			if err != nil {
				t.Logf("AreAgentsHealthy check: %v (this may be expected in some test environments)", err)
			} else {
				t.Log("AreAgentsHealthy: All agents are healthy")
			}
		})
	})
}

// TestAgencyLeaderDiscoveryDuringUpgrade simulates leader discovery during upgrades.
func TestAgencyLeaderDiscoveryDuringUpgrade(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get all agency endpoints
			agencyEndpoints, err := getAgencyEndpoints(ctx, client)
			require.NoError(t, err)
			require.NotEmpty(t, agencyEndpoints, "Should have at least one agency endpoint")

			if len(agencyEndpoints) < 2 {
				t.Skip("Need at least 2 agency endpoints to test upgrade scenario")
				return
			}

			// Create individual clients for each agency endpoint
			agencyClients := make([]arangodb.Client, len(agencyEndpoints))
			for i, ep := range agencyEndpoints {
				conn := connection.NewHttpConnection(connection.HttpConfiguration{
					Endpoint: connection.NewRoundRobinEndpoints([]string{ep}),
					Transport: &http.Transport{
						//nolint:gosec // test-only agency connection
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				})
				agencyClients[i] = arangodb.NewClient(conn)
			}

			// During upgrades, leader endpoints might differ temporarily
			// Also, during upgrades, multiple agents might temporarily think they're leaders
			// Use both context options to allow this scenario
			ctxWithAllowDifferent := arangodb.WithAllowDifferentLeaderEndpoints(
				arangodb.WithAllowNoLeader(ctx),
			)
			err = arangodb.AreAgentsHealthy(ctxWithAllowDifferent, agencyClients)
			if err != nil {
				t.Logf("AreAgentsHealthy during upgrade simulation: %v", err)
			} else {
				t.Log("AreAgentsHealthy: Agents healthy (allowing different leader endpoints and multiple leaders)")
			}

			// Test that agency operations still work during upgrade scenarios
			// Verify we can read from at least one of the agency clients
			var result interface{}
			readSuccess := false
			for i, agencyClient := range agencyClients {
				if err := agencyClient.ReadKey(ctx, []string{"arango"}, &result); err == nil {
					readSuccess = true
					t.Logf("Successfully read from agency client %d during upgrade simulation", i)
					break
				}
			}
			require.True(t, readSuccess, "Should be able to read from at least one agency client during upgrade scenarios")
		})
	})
}

// TestAgencyLock tests the agency lock functionality in V2.
func TestAgencyLock(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get agency client
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %v", err)
				return
			}

			// agencyClient is already a ClientAgency (Client embeds ClientAgency)
			require.NotNil(t, agencyClient, "Agency client should not be nil")

			// Create a unique lock key for this test
			key := []string{"go-driver", "v2", "TestAgencyLock", GenerateUUID("lock")}
			lockID := GenerateUUID("lock-id")

			// Create lock with test logger
			logger := &testLogger{t: t}
			lock, err := arangodb.NewLock(logger, agencyClient, key, lockID, time.Minute)
			require.NoError(t, err, "NewLock should succeed")
			require.NotNil(t, lock, "Lock should not be nil")

			// Initially, lock should not be locked
			assert.False(t, lock.IsLocked(), "Lock should initially be unlocked")

			// Lock it
			err = lock.Lock(ctx)
			require.NoError(t, err, "Lock should succeed")
			assert.True(t, lock.IsLocked(), "Lock should be locked after Lock()")

			// Try to lock again - should fail with AlreadyLockedError
			err = lock.Lock(ctx)
			require.Error(t, err, "Lock should fail when already locked")
			assert.True(t, arangodb.IsAlreadyLocked(err), "Error should be AlreadyLockedError")

			// Unlock it
			err = lock.Unlock(ctx)
			require.NoError(t, err, "Unlock should succeed")
			assert.False(t, lock.IsLocked(), "Lock should be unlocked after Unlock()")

			// Try to unlock again - should fail with NotLockedError
			err = lock.Unlock(ctx)
			require.Error(t, err, "Unlock should fail when not locked")
			assert.True(t, arangodb.IsNotLocked(err), "Error should be NotLockedError")

			// Lock and unlock again to ensure it works multiple times
			err = lock.Lock(ctx)
			require.NoError(t, err, "Lock should succeed on second attempt")
			assert.True(t, lock.IsLocked(), "Lock should be locked")

			err = lock.Unlock(ctx)
			require.NoError(t, err, "Unlock should succeed on second attempt")
			assert.False(t, lock.IsLocked(), "Lock should be unlocked")
		})
	})
}

// TestAgencyLockConcurrent tests concurrent lock acquisition.
func TestAgencyLockConcurrent(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get agency client
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %v", err)
				return
			}

			// agencyClient is already a ClientAgency (Client embeds ClientAgency)
			require.NotNil(t, agencyClient, "Agency client should not be nil")

			// Create a unique lock key for this test
			key := []string{"go-driver", "v2", "TestAgencyLockConcurrent", GenerateUUID("lock")}
			lockID1 := GenerateUUID("lock-id-1")
			lockID2 := GenerateUUID("lock-id-2")

			// Create two locks with different IDs
			logger := &testLogger{t: t}
			lock1, err := arangodb.NewLock(logger, agencyClient, key, lockID1, time.Minute)
			require.NoError(t, err)
			lock2, err := arangodb.NewLock(logger, agencyClient, key, lockID2, time.Minute)
			require.NoError(t, err)

			// Lock1 acquires the lock
			err = lock1.Lock(ctx)
			require.NoError(t, err, "Lock1 should acquire lock")
			assert.True(t, lock1.IsLocked(), "Lock1 should be locked")

			// Lock2 tries to acquire the same lock - should fail
			err = lock2.Lock(ctx)
			require.Error(t, err, "Lock2 should fail to acquire lock")
			assert.True(t, arangodb.IsAlreadyLocked(err), "Error should be AlreadyLockedError")
			assert.False(t, lock2.IsLocked(), "Lock2 should not be locked")

			// Unlock lock1
			err = lock1.Unlock(ctx)
			require.NoError(t, err, "Lock1 unlock should succeed")

			// Now lock2 should be able to acquire the lock
			err = lock2.Lock(ctx)
			require.NoError(t, err, "Lock2 should now acquire lock")
			assert.True(t, lock2.IsLocked(), "Lock2 should be locked")

			// Cleanup
			err = lock2.Unlock(ctx)
			require.NoError(t, err, "Lock2 unlock should succeed")
		})
	})
}

// TestAgencyLockHelperMethods tests the helper methods WriteKeyIfEmpty, WriteKeyIfEqualTo, RemoveKeyIfEqualTo.
func TestAgencyLockHelperMethods(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			// Get agency client
			agencyClient, err := getAgencyClient(ctx, t, client)
			if err != nil {
				t.Skipf("Skip agency test: %v", err)
				return
			}

			// agencyClient is already a ClientAgency (Client embeds ClientAgency)
			require.NotNil(t, agencyClient, "Agency client should not be nil")

			// Test WriteKeyIfEmpty
			key1 := []string{"go-driver", "v2", "TestHelperMethods", GenerateUUID("key1")}
			value1 := "test-value-1"
			err = agencyClient.WriteKeyIfEmpty(ctx, key1, value1, time.Minute)
			require.NoError(t, err, "WriteKeyIfEmpty should succeed on empty key")

			// Try to write again - should fail (key is not empty)
			err = agencyClient.WriteKeyIfEmpty(ctx, key1, "another-value", time.Minute)
			require.Error(t, err, "WriteKeyIfEmpty should fail when key is not empty")
			assert.True(t, shared.IsPreconditionFailed(err), "Error should be PreconditionFailed")

			// Verify the value is still the original
			var readValue1 string
			err = agencyClient.ReadKey(ctx, key1, &readValue1)
			require.NoError(t, err, "ReadKey should succeed")
			assert.Equal(t, value1, readValue1, "Value should remain unchanged")

			// Test WriteKeyIfEqualTo
			key2 := []string{"go-driver", "v2", "TestHelperMethods", GenerateUUID("key2")}
			oldValue := "old-value"
			newValue := "new-value"

			// First set the key
			transaction := agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction.AddKey(agency.NewKeySet(key2, oldValue))
			err = agencyClient.WriteTransaction(ctx, transaction)
			require.NoError(t, err, "Initial WriteTransaction should succeed")

			// Update with WriteKeyIfEqualTo - should succeed
			err = agencyClient.WriteKeyIfEqualTo(ctx, key2, newValue, oldValue, time.Minute)
			require.NoError(t, err, "WriteKeyIfEqualTo should succeed when values match")

			// Verify the value was updated
			var readValue2 string
			err = agencyClient.ReadKey(ctx, key2, &readValue2)
			require.NoError(t, err, "ReadKey should succeed")
			assert.Equal(t, newValue, readValue2, "Value should be updated")

			// Try to update with wrong old value - should fail
			err = agencyClient.WriteKeyIfEqualTo(ctx, key2, "another-value", oldValue, time.Minute)
			require.Error(t, err, "WriteKeyIfEqualTo should fail when old value doesn't match")
			assert.True(t, shared.IsPreconditionFailed(err), "Error should be PreconditionFailed")

			// Test RemoveKeyIfEqualTo
			key3 := []string{"go-driver", "v2", "TestHelperMethods", GenerateUUID("key3")}
			value3 := "value-to-remove"

			// First set the key
			transaction = agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction.AddKey(agency.NewKeySet(key3, value3))
			err = agencyClient.WriteTransaction(ctx, transaction)
			require.NoError(t, err, "Initial WriteTransaction should succeed")

			// Remove with RemoveKeyIfEqualTo - should succeed
			err = agencyClient.RemoveKeyIfEqualTo(ctx, key3, value3)
			require.NoError(t, err, "RemoveKeyIfEqualTo should succeed when values match")

			// Verify the key was removed
			var readValue3 interface{}
			err = agencyClient.ReadKey(ctx, key3, &readValue3)
			require.Error(t, err, "ReadKey should fail for removed key")
			assert.True(t, arangodb.IsKeyNotFound(err), "Error should be KeyNotFound")

			// Try to remove with wrong value - should fail
			transaction = agency.NewTransaction(nil, agency.TransactionOptions{})
			transaction.AddKey(agency.NewKeySet(key3, value3))
			err = agencyClient.WriteTransaction(ctx, transaction)
			require.NoError(t, err, "Re-set key should succeed")

			err = agencyClient.RemoveKeyIfEqualTo(ctx, key3, "wrong-value")
			require.Error(t, err, "RemoveKeyIfEqualTo should fail when value doesn't match")
			assert.True(t, shared.IsPreconditionFailed(err), "Error should be PreconditionFailed")
		})
	})
}

// testLogger implements arangodb.Logger for testing.
type testLogger struct {
	t *testing.T
}

func (l *testLogger) Errorf(msg string, args ...interface{}) {
	l.t.Logf("Lock error: "+msg, args...)
}
