//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/agency"
	httpdriver "github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/jwt"
	"github.com/arangodb/go-driver/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getAgencyEndpoints queries the cluster to get all agency endpoints.
func getAgencyEndpoints(ctx context.Context, c driver.Client) ([]string, error) {
	cl, err := c.Cluster(ctx)
	if err != nil {
		return nil, err
	}
	h, err := cl.Health(ctx)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, entry := range h.Health {
		if entry.Role == driver.ServerRoleAgent {
			ep := util.FixupEndpointURLScheme(entry.Endpoint)
			result = append(result, ep)
		}
	}
	return result, nil
}

// getJWTSecretAuth return auth with superjwt
func getJWTSecretAuth(t testEnv) driver.Authentication {
	value := os.Getenv("TEST_JWTSECRET")
	if value == "" {
		return nil
	}

	header, err := jwt.CreateArangodJwtAuthorizationHeader(value, "arangodb")
	if err != nil {
		t.Fatalf("Could not create JWT authentication header: %s", describe(err))
	}
	return driver.RawAuthentication(header)
}

// getHttpAuthAgencyConnection queries the cluster and creates an agency accessor using an agency.AgencyConnection for the entire agency using HTTP for all protocols.
func getHttpAuthAgencyConnection(ctx context.Context, t testEnv, c driver.Client) (agency.Agency, error) {
	endpoints, err := getAgencyEndpoints(ctx, c)
	if err != nil {
		return nil, err
	}
	conn, err := agency.NewAgencyConnection(httpdriver.ConnectionConfig{
		Endpoints: endpoints,
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return nil, err
	}

	if clusterAuth := createAuthenticationFromEnv(t); clusterAuth != nil {
		if auth := getJWTSecretAuth(t); auth != nil {
			conn, err = conn.SetAuthentication(auth)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("auth is required in agency")
		}
	}

	result, err := agency.NewAgency(conn)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// getAgencyConnection queries the cluster and creates an agency accessor using an agency.AgencyConnection for the entire agency.
func getAgencyConnection(ctx context.Context, t testEnv, c driver.Client) (agency.Agency, error) {
	if os.Getenv("TEST_CONNECTION") == "vst" {
		// These tests assume an HTTP connetion, so we skip under this condition
		return nil, driver.ArangoError{HasError: true, Code: 412, ErrorMessage: "Using vst is not supported in agency tests"}
	}
	endpoints, err := getAgencyEndpoints(ctx, c)
	if err != nil {
		return nil, err
	}
	conn, err := agency.NewAgencyConnection(httpdriver.ConnectionConfig{
		Endpoints: endpoints,
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if auth := createAuthenticationFromEnv(t); auth != nil {
		// This requires a JWT token, which we not always have in this test, so we skip under this condition
		return nil, driver.ArangoError{HasError: true, Code: 412, ErrorMessage: "Authentication required but not supported in agency tests"}
	}

	result, err := agency.NewAgency(conn)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// getIndividualAgencyConnections queries the cluster and creates an agency accessor using a single http.Connection for each agent.
func getIndividualAgencyConnections(ctx context.Context, t testEnv, c driver.Client) ([]agency.Agency, error) {
	if os.Getenv("TEST_CONNECTION") == "vst" {
		// These tests assume an HTTP connetion, so we skip under this condition
		return nil, driver.ArangoError{HasError: true, Code: 412}
	}
	endpoints, err := getAgencyEndpoints(ctx, c)
	if err != nil {
		return nil, err
	}
	if auth := createAuthenticationFromEnv(t); auth != nil {
		// This requires a JWT token, which we not always have in this test, so we skip under this condition
		return nil, driver.ArangoError{HasError: true, Code: 412}
	}
	result := make([]agency.Agency, len(endpoints))
	for i, ep := range endpoints {
		conn, err := httpdriver.NewConnection(httpdriver.ConnectionConfig{
			Endpoints:          []string{ep},
			TLSConfig:          &tls.Config{InsecureSkipVerify: true},
			DontFollowRedirect: true,
		})
		if err != nil {
			return nil, err
		}
		result[i], err = agency.NewAgency(conn)
	}
	return result, nil
}

// TestAgencyRead tests the Agency.ReadKey method.
func TestAgencyRead(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		var result interface{}
		if err := a.ReadKey(ctx, []string{"not-found-b1d534b1-26d8-5ad0-b22d-23d49d3ea92c"}, &result); !agency.IsKeyNotFound(err) {
			t.Errorf("Expected KeyNotFoundError, got %s", describe(err))
		}
		if err := a.ReadKey(ctx, []string{"arango"}, &result); err != nil {
			t.Errorf("Expected success, got %s", describe(err))
		}
	}
}

// TestAgencyWrite tests the Agency.WriteKey method.
func TestAgencyWrite(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		op := func(key []string, value, result interface{}) {
			if err := a.WriteKey(ctx, key, value, 0); err != nil {
				t.Fatalf("WriteKey failed: %s", describe(err))
			}
			if err := a.ReadKey(ctx, key, result); err != nil {
				t.Fatalf("ReadKey failed: %s", describe(err))
			}
			if !reflect.DeepEqual(value, reflect.ValueOf(result).Elem().Interface()) {
				t.Errorf("Expected '%v', got '%v'", value, result)
			}
		}
		op([]string{"go-driver", "TestAgencyWrite", "string"}, "hello world", new(string))
		op([]string{"go-driver", "TestAgencyWrite", "int"}, 55, new(int))
		op([]string{"go-driver", "TestAgencyWrite", "bool"}, true, new(bool))
		op([]string{"go-driver", "TestAgencyWrite", "object"}, struct{ Field string }{Field: "hello world"}, &struct{ Field string }{})
		op([]string{"go-driver", "TestAgencyWrite", "string-array"}, []string{"hello", "world"}, new([]string))
		op([]string{"go-driver", "TestAgencyWrite", "int-array"}, []int{-5, 34, 11}, new([]int))
	}
}

// TestAgencyWriteIfEmpty tests the Agency.WriteKeyIfEmpty method.
func TestAgencyWriteIfEmpty(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		key := []string{"go-driver", "TestAgencyWriteIfEmpty"}
		if err := a.WriteKey(ctx, key, "foo", 0); err != nil {
			t.Fatalf("WriteKey failed: %s", describe(err))
		}
		var result string
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if err := a.WriteKeyIfEmpty(ctx, key, "not-foo", 0); !driver.IsPreconditionFailed(err) {
			t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
		}
		if err := a.RemoveKey(ctx, key); err != nil {
			t.Fatalf("RemoveKey failed: %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); !agency.IsKeyNotFound(err) {
			t.Errorf("Expected KeyNotFoundError, got %s", describe(err))
		}
		if err := a.WriteKeyIfEmpty(ctx, key, "again-foo", 0); err != nil {
			t.Errorf("WriteKeyIfEmpty failed: %s", describe(err))
		}
	}
}

// TestAgencyWriteIfEqualTo tests the Agency.WriteIfEqualTo method.
func TestAgencyWriteIfEqualTo(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		key := []string{"go-driver", "TestAgencyWriteIfEqualTo"}
		if err := a.WriteKey(ctx, key, "foo", 0); err != nil {
			t.Fatalf("WriteKey failed: %s", describe(err))
		}
		var result string
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if result != "foo" {
			t.Errorf("Expected 'foo', got '%s", result)
		}
		if err := a.WriteKeyIfEqualTo(ctx, key, "not-foo", "incorrect", 0); !driver.IsPreconditionFailed(err) {
			t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if result != "foo" {
			t.Errorf("Expected 'foo', got '%s", result)
		}
		if err := a.WriteKeyIfEqualTo(ctx, key, "not-foo", "foo", 0); err != nil {
			t.Fatalf("WriteKeyIfEqualTo failed: %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if result != "not-foo" {
			t.Errorf("Expected 'not-foo', got '%s", result)
		}
	}
}

// TestAgencyRemove tests the Agency.RemoveKey method.
func TestAgencyRemove(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		key := []string{"go-driver", "TestAgencyRemove"}
		if err := a.WriteKey(ctx, key, "foo", 0); err != nil {
			t.Fatalf("WriteKey failed: %s", describe(err))
		}
		var result string
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if err := a.RemoveKey(ctx, key); err != nil {
			t.Fatalf("RemoveKey failed: %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); !agency.IsKeyNotFound(err) {
			t.Errorf("Expected KeyNotFoundError, got %s", describe(err))
		}
	}
}

// TestAgencyRemoveIfEqualTo tests the Agency.RemoveKeyIfEqualTo method.
func TestAgencyRemoveIfEqualTo(t *testing.T) {
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		key := []string{"go-driver", "RemoveKeyIfEqualTo"}
		if err := a.WriteKey(ctx, key, "foo", 0); err != nil {
			t.Fatalf("WriteKey failed: %s", describe(err))
		}
		var result string
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if err := a.RemoveKeyIfEqualTo(ctx, key, "incorrect"); !driver.IsPreconditionFailed(err) {
			t.Errorf("Expected PreconditionFailedError, got %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); err != nil {
			t.Errorf("ReadKey failed: %s", describe(err))
		}
		if err := a.RemoveKeyIfEqualTo(ctx, key, "foo"); err != nil {
			t.Fatalf("RemoveKeyIfEqualTo failed: %s", describe(err))
		}
		if err := a.ReadKey(ctx, key, &result); !agency.IsKeyNotFound(err) {
			t.Errorf("Expected KeyNotFoundError, got %s", describe(err))
		}
	}
}

func TestAgencyCallbacks(t *testing.T) {
	if getTestMode() != testModeCluster {
		t.Skipf("Not a cluster mode")
	}

	rootKeyAgency := "TestAgencyCallbacks"
	ctx := context.Background()
	c := createClientFromEnv(t, true)

	a, err := getAgencyConnection(ctx, t, c)
	if driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	}
	require.NoError(t, err)

	type callback struct {
		key     string
		URL     string
		observe bool
	}

	testCases := []struct {
		name      string
		callbacks []callback
	}{
		{
			name: "Register callback",
			callbacks: []callback{
				{
					key:     rootKeyAgency + "/test/1",
					URL:     "",
					observe: true,
				},
			},
		},
		{
			name: "Register and unregister callback",
			callbacks: []callback{
				{
					key:     rootKeyAgency + "/test/1",
					URL:     "localhost:1234",
					observe: true,
				},
				{
					key:     rootKeyAgency + "/test/1",
					URL:     "localhost:1234",
					observe: false,
				},
			},
		},
		{
			name: "Unregister non-existing callback",
			callbacks: []callback{
				{
					key:     rootKeyAgency + "/test/1",
					URL:     "localhost:1234",
					observe: false,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var err error

			for _, v := range testCase.callbacks {
				key := strings.Split(v.key, "/")
				if v.observe {
					err = a.RegisterChangeCallback(ctx, key, v.URL)
				} else {
					err = a.UnregisterChangeCallback(ctx, key, v.URL)
				}
				require.NoError(t, err)
			}
		})
	}
}

func TestAgencyTransactionWrite(t *testing.T) {
	writeTransaction(t, false)
}

func TestAgencyTransactionTransient(t *testing.T) {
	t.Skip("Currently it does not work because we can not parse the response " +
		"using ParseBody or ParseArrayBody functions")
	writeTransaction(t, true)
}

func writeTransaction(t *testing.T, transient bool) {
	if getTestMode() != testModeCluster {
		t.Skipf("Not a cluster mode")
	}

	rootKeyAgency := "TestAgencyWriteTransaction"
	ctx := context.Background()
	c := createClientFromEnv(t, true)

	a, err := getAgencyConnection(ctx, t, c)
	if driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	}
	require.NoError(t, err)

	type TransactionTest struct {
		keys       []agency.KeyChanger
		conditions map[string]agency.KeyConditioner
	}

	type Request struct {
		transaction   TransactionTest
		expectedError error
	}

	testCases := []struct {
		name           string
		requests       []Request
		expectedResult map[string]interface{}
		sleep          time.Duration
	}{
		{
			name: "Create two keys within one request",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
							agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2", 0),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "1",
						"2": "2",
					},
				},
			},
		},
		{
			name: "Create two keys with two requests",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2", 0),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "1",
						"2": "2",
					},
				},
			},
		},
		{
			name: "Create two keys in one requests and then remove one of them in next request",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
							agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2", 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyDelete([]string{rootKeyAgency, "test", "2"}),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "1",
					},
				},
			},
		},
		{
			name: "Register and unregister callback",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyObserve([]string{rootKeyAgency, "test", "1"}, "localhost:2345", true),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyObserve([]string{rootKeyAgency, "test", "1"}, "localhost:2345", false),
						},
					},
				},
			},
		},
		{
			name: "Delete non-existing keys",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyDelete([]string{rootKeyAgency, "test", "1"}),
							agency.NewKeyDelete([]string{rootKeyAgency, "test", "2"}),
							agency.NewKeyDelete([]string{rootKeyAgency, "test", "3"}),
							agency.NewKeySet([]string{rootKeyAgency, "test", "4"}, "2", 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyDelete([]string{rootKeyAgency, "test", "4"}),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{},
				},
			},
		},
		{
			name:  "TTL of Key is expired",
			sleep: time.Second * 2,
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "4", time.Second),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{},
				},
			},
		},
		{
			name: "Conditions 'ifEqual' and 'ifNotEqual' allow to crate two keys",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
							agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2", 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "3", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionIfEqual("1"),
							rootKeyAgency + "/test/2": agency.NewConditionIfNotEqual("3"),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "3",
						"2": "2",
					},
				},
			},
		},
		{
			name: "Condition 'IfNotEqual' does not allow to create two keys",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
							agency.NewKeySet([]string{rootKeyAgency, "test", "2"}, "2", 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "3", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionIfEqual("1"),
							rootKeyAgency + "/test/2": agency.NewConditionIfNotEqual("2"),
						},
					},
					expectedError: driver.ArangoError{
						HasError: true,
						Code:     http.StatusPreconditionFailed,
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "4", 0),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "4",
						"2": "2",
					},
				},
			},
		},
		{
			name: "Testing condition 'oldEmpty'",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "1", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionOldEmpty(true),
						},
					},
				},
				{
					transaction: TransactionTest{

						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "3", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionOldEmpty(true),
						},
					},
					expectedError: driver.ArangoError{
						HasError: true,
						Code:     http.StatusPreconditionFailed,
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "1",
					},
				},
			},
		},
		{
			name: "Testing condition 'isArray'",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, 1, 0),
						},
					},
				},
				{
					transaction: TransactionTest{

						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "3", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionIsArray(true),
						},
					},
					expectedError: driver.ArangoError{
						HasError: true,
						Code:     http.StatusPreconditionFailed,
					},
				},
				{
					transaction: TransactionTest{

						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, []int{1, 2}, 0),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeySet([]string{rootKeyAgency, "test", "1"}, "8", 0),
						},
						conditions: map[string]agency.KeyConditioner{
							rootKeyAgency + "/test/1": agency.NewConditionIsArray(true),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"1": "8",
					},
				},
			},
		},
		{
			name: "Adding and removing elements from array",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, "1"),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, "2"),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, "3"),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, "4"),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayErase([]string{rootKeyAgency, "test", "array"}, "2"),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"array": []interface{}{
							"1",
							"3",
							"4",
						},
					},
				},
			},
		},
		{
			name: "Replace element in array",
			requests: []Request{
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, map[string]interface{}{
								"database":   "db",
								"collection": "col",
								"shard":      0,
							}),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayPush([]string{rootKeyAgency, "test", "array"}, map[string]interface{}{
								"database":   "db",
								"collection": "col",
								"shard":      1,
							}),
						},
					},
				},
				{
					transaction: TransactionTest{
						keys: []agency.KeyChanger{
							agency.NewKeyArrayReplace([]string{rootKeyAgency, "test", "array"},
								map[string]interface{}{
									"database":   "db",
									"collection": "col",
									"shard":      0,
								}, map[string]interface{}{
									"database":   "db",
									"collection": "col",
									"shard":      2,
								}),
						},
					},
				},
			},
			expectedResult: map[string]interface{}{
				rootKeyAgency: map[string]interface{}{
					"test": map[string]interface{}{
						"array": []interface{}{
							map[string]interface{}{
								"database":   "db",
								"collection": "col",
								"shard":      float64(2),
							},
							map[string]interface{}{
								"database":   "db",
								"collection": "col",
								"shard":      float64(1),
							},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, requestTest := range testCase.requests {
				transaction := agency.NewTransaction("", agency.TransactionOptions{Transient: transient})
				for _, v := range requestTest.transaction.keys {
					transaction.AddKey(v)
				}

				if requestTest.transaction.conditions != nil {
					for conditionTestKey, conditionTest := range requestTest.transaction.conditions {
						transaction.AddCondition(strings.Split(conditionTestKey, "/"), conditionTest)
					}
				}

				err := a.WriteTransaction(ctx, transaction)
				if requestTest.expectedError != nil {
					require.EqualError(t, err, requestTest.expectedError.Error())
				} else {
					require.NoError(t, err)
				}
			}

			if testCase.sleep > 0 {
				time.Sleep(testCase.sleep)
			}
			var result map[string]interface{}
			err = a.ReadKey(ctx, []string{rootKeyAgency}, &result)
			if agency.IsKeyNotFound(err) {
				if testCase.expectedResult == nil || len(testCase.expectedResult) == 0 {

				} else {
					require.NoError(t, err)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedResult[rootKeyAgency], result)
			}

			cleanUpTransaction := agency.NewTransaction("", agency.TransactionOptions{Transient: transient})
			cleanUpTransaction.AddKey(agency.NewKeyDelete([]string{rootKeyAgency}))
			err := a.WriteTransaction(ctx, cleanUpTransaction)
			require.NoError(t, err)
		})
	}
}
