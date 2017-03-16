//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

// +build failover

package test

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/coreos/go-iptables/iptables"
)

const (
	filterTable = "filter"
	chainName   = "ARANGOGODRIVER"
)

// TestFailoverDrop performs various tests while DROP'ng traffic to 1 coordinator.
func TestFailoverDrop(t *testing.T) {
	failoverTest("DROP", t)
}

// TestFailoverReject performs various tests while REJECT'ng traffic to 1 coordinator.
func TestFailoverReject(t *testing.T) {
	failoverTest("REJECT", t)
}

func failoverTest(action string, t *testing.T) {
	iptc, err := iptables.New()
	if err != nil {
		t.Fatalf("Failed to create iptables client: %s", describe(err))
	}
	createChains(iptc, t)
	defer cleanupChains(iptc, t)

	coordinatorPorts := []int{7002, 7007, 7012}
	var conn driver.Connection
	c := createClientFromEnv(t, true, &conn)
	db := ensureDatabase(nil, c, "failover_test", nil, t)
	col := ensureCollection(nil, db, strings.ToLower(action)+"_test", nil, t)

	lastEndpoint := ""
	endpointChanges := 0
	for i := 0; i < 1000 && endpointChanges < 10; i++ {
		port := coordinatorPorts[rand.Intn(len(coordinatorPorts))]
		ruleSpec := blockPort(iptc, port, action, t)

		// Perform low lever request and check handling endpoint
		for {
			req, err := conn.NewRequest("GET", "/_api/version")
			if err != nil {
				t.Fatalf("Cannot create request: %s", describe(err))
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*9)
			resp, err := conn.Do(ctx, req)
			cancel()
			if driver.IsResponseError(err) {
				t.Logf("ResponseError in version request")
				continue
			} else if err != nil {
				t.Fatalf("Cannot execute request: %s", describe(err))
			}
			ep := resp.Endpoint()
			if ep != lastEndpoint {
				lastEndpoint = ep
				endpointChanges++
				t.Logf("New server detected: %s", ep)
			}
			break
		}

		// Create document & read it
		doc := UserDoc{
			"Jan",
			40,
		}
		meta, err := col.CreateDocument(nil, doc)
		if err != nil {
			t.Fatalf("Failed to create new document: %s", describe(err))
		}
		// Document must exists now
		var readDoc UserDoc
		if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
			t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
		}
		if !reflect.DeepEqual(doc, readDoc) {
			t.Errorf("Got wrong document. Expected %+v, got %+v", doc, readDoc)
		}

		removeRuleSpec(iptc, ruleSpec, t)
	}
}

func blockPort(client *iptables.IPTables, port int, action string, t *testing.T) []string {
	ruleSpec := []string{
		"-p", "tcp",
		"-m", "tcp", "--dport", strconv.Itoa(port),
		"-j", action,
	}
	t.Logf("Denying traffic to TCP port %d", port)
	if found, err := client.Exists(filterTable, chainName, ruleSpec...); err != nil {
		t.Fatalf("Failed to check existance of rulespec %q: %v", ruleSpec, err)
	} else if !found {
		if err := client.Insert(filterTable, chainName, 1, ruleSpec...); err != nil {
			t.Fatalf("Failed to deny traffic to TCP port %d: %v", port, err)
		}
	}
	return ruleSpec
}

func removeRuleSpec(client *iptables.IPTables, ruleSpec []string, t *testing.T) {
	if found, err := client.Exists(filterTable, chainName, ruleSpec...); err != nil {
		t.Fatalf("Failed to check existance of rulespec %q: %v", ruleSpec, err)
	} else if found {
		if err := client.Delete(filterTable, chainName, ruleSpec...); err != nil {
			t.Fatalf("Failed to remove ruleSpec %q: %v", ruleSpec, err)
		}
	}
}

func createChains(client *iptables.IPTables, t *testing.T) {
	if err := client.ClearChain(filterTable, chainName); err != nil {
		t.Fatalf("Failed to create chain: %s", describe(err))
	}
	if err := client.Append(filterTable, chainName, "-j", "RETURN"); err != nil {
		t.Fatalf("Failed to append RETURN to chain: %s", describe(err))
	}
	if err := client.Insert(filterTable, "INPUT", 1, "-j", chainName); err != nil {
		t.Fatalf("Failed to insert INPUT chain: %s", describe(err))
	}
	if err := client.Insert(filterTable, "FORWARD", 1, "-j", chainName); err != nil {
		t.Fatalf("Failed to insert FORWARD OUTPUT chain: %s", describe(err))
	}
	if err := client.Insert(filterTable, "OUTPUT", 1, "-j", chainName); err != nil {
		t.Fatalf("Failed to insert OUTPUT chain: %s", describe(err))
	}
}

// cleanupChains removes all generated iptables chain & rules made by createChains.
func cleanupChains(client *iptables.IPTables, t *testing.T) {
	if err := client.Delete(filterTable, "INPUT", "-j", chainName); err != nil {
		t.Logf("Failed to remove INPUT chain rule: %v", err)
	}
	if err := client.Delete(filterTable, "FORWARD", "-j", chainName); err != nil {
		t.Logf("Failed to remove FORWARD chain rule: %v", err)
	}
	if err := client.Delete(filterTable, "OUTPUT", "-j", chainName); err != nil {
		t.Logf("Failed to remove OUTPUT chain rule: %v", err)
	}
	if err := client.ClearChain(filterTable, chainName); err != nil {
		t.Logf("Failed to clear '%s' chain: %v", chainName, err)
	}
	if err := client.DeleteChain(filterTable, chainName); err != nil {
		t.Logf("Failed to remove '%s' chain: %v", chainName, err)
	}
}
