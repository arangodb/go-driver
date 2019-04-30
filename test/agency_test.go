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
	"os"
	"reflect"
	"testing"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/agency"
	"github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/util"
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
	conn, err := agency.NewAgencyConnection(http.ConnectionConfig{
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
		conn, err := http.NewConnection(http.ConnectionConfig{
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
