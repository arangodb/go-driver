//
// DISCLAIMER
//
// Copyright 2019 ArangoDB GmbH, Cologne, Germany
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
// Author Max neunhoeffer
//

package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
)

func checkEnabled(t *testing.T, c driver.Client, ctx context.Context) {
	_, err := c.Statistics(ctx)
	if err != nil {
		if driver.IsArangoErrorWithErrorNum(err, 36) {
			t.Skip("Statistics disabled.")
		}
		t.Fatalf("Statistics failed: %s", describe(err))
	}
}

// TestServerStatisticsWorks tests if Client.Statistics works at all
func TestServerStatisticsWorks(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	checkEnabled(t, c, ctx)

	stats, err := c.Statistics(ctx)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}
	b, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Cannot marshal statistics to JSON: %s", describe(err))
	}
  t.Logf("Statistics: %s", string(b))
}

// TestServerStatisticsTraffic tests if Client.Statistics increase
// with traffic in the correct way
func TestServerStatisticsTraffic(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	checkEnabled(t, c, ctx)

	statsBefore, err := c.Statistics(ctx)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	db := ensureDatabase(nil, c, "statistics_test", nil, t)
	col := ensureCollection(nil, db, "statistics_test", nil, t)
	doc := UserDoc{
	  "Max",
		50,
  }
	for i := 0; i < 1000; i++ {
		_, err := col.CreateDocument(nil, doc)
		if err != nil {
		  t.Fatalf("Failed to create new document: %s", describe(err))
		}
	}

	time.Sleep(time.Second)  // Wait until statistics updated

	statsAfter, err := c.Statistics(ctx)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	//b, _ := json.Marshal(statsBefore)
	//t.Logf("statsBefore: %s", string(b))
	//b, _ = json.Marshal(statsAfter)
	//t.Logf("statsAfter: %s", string(b))

  var diff float64
	diff = statsAfter.Client.BytesReceived.Sum - statsBefore.Client.BytesReceived.Sum
  if diff < 40000.0 {
		t.Errorf("Difference in Client.BytesReceived.Sum is too small (< 40000.0): %f", diff)
	}
	diff = statsAfter.Client.BytesSent.Sum - statsBefore.Client.BytesSent.Sum
  if diff < 100000.0 {
		t.Errorf("Difference in Client.BytesSent.Sum is too small (< 100000.0): %f", diff)
	}
	var intdiff int64
	intdiff = statsAfter.Client.BytesReceived.Count - statsBefore.Client.BytesReceived.Count
  if intdiff < 1000 {
		t.Errorf("Difference in Client.BytesReceived.Count is too small (< 1000): %d", intdiff)
	}
	intdiff = statsAfter.Client.BytesSent.Count - statsBefore.Client.BytesSent.Count
  if intdiff < 1000 {
		t.Errorf("Difference in Client.BytesSent.Count is too small (< 1000): %d", intdiff)
	}

	// Now check if user only stats are there and see if they should have increased:
  if statsBefore.ClientUser.BytesReceived.Counts != nil {
	  t.Logf("New user only statistics API is present, testing...")
		auth := os.Getenv("TEST_AUTHENTICATION")
		if auth == "super:testing" {
			t.Logf("Authentication %s is jwt superuser, expecting no user traffic...", auth)
	    // Traffic is superuser, so nothing should be counted in ClientUser
			diff = statsAfter.ClientUser.BytesReceived.Sum - statsBefore.ClientUser.BytesReceived.Sum
			if diff > 1.0 {
				t.Errorf("Difference in ClientUser.BytesReceived.Sum is too large (> 1.0): %f", diff)
			}
			diff = statsAfter.ClientUser.BytesSent.Sum - statsBefore.ClientUser.BytesSent.Sum
			if diff > 1.0 {
				t.Errorf("Difference in ClientUser.BytesSent.Sum is too large (> 1.0): %f", diff)
			}
			var intdiff int64
			intdiff = statsAfter.ClientUser.BytesReceived.Count - statsBefore.ClientUser.BytesReceived.Count
			if intdiff > 0 {
				t.Errorf("Difference in ClientUser.BytesReceived.Count is too large (> 0): %d", intdiff)
			}
			intdiff = statsAfter.ClientUser.BytesSent.Count - statsBefore.ClientUser.BytesSent.Count
			if intdiff > 0 {
				t.Errorf("Difference in ClientUser.BytesSent.Count is too large (> 0): %d", intdiff)
			}
		} else {
			t.Logf("Authentication %s is not jwt superuser, expecting to see user traffic...", auth)
			// Traffic is either unauthenticated or with password, so there should
			// be traffic in ClientUser
			diff = statsAfter.ClientUser.BytesReceived.Sum - statsBefore.ClientUser.BytesReceived.Sum
			if diff < 40000.0 {
				t.Errorf("Difference in ClientUser.BytesReceived.Sum is too small (< 40000.0): %f", diff)
			}
			diff = statsAfter.ClientUser.BytesSent.Sum - statsBefore.ClientUser.BytesSent.Sum
			if diff < 100000.0 {
				t.Errorf("Difference in ClientUser.BytesSent.Sum is too small (< 100000.0): %f", diff)
			}
			var intdiff int64
			intdiff = statsAfter.ClientUser.BytesReceived.Count - statsBefore.ClientUser.BytesReceived.Count
			if intdiff < 1000 {
				t.Errorf("Difference in ClientUser.BytesReceived.Count is too small (< 1000): %d", intdiff)
			}
			intdiff = statsAfter.ClientUser.BytesSent.Count - statsBefore.ClientUser.BytesSent.Count
			if intdiff < 1000 {
				t.Errorf("Difference in ClientUser.BytesSent.Count is too small (< 1000): %d", intdiff)
			}
		}
	} else {
		t.Log("Skipping ClientUser tests for statistics, since API is not present.")
	}
}
