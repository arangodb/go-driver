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

// doSomeWrites does some writes
func doSomeWrites(t *testing.T, ctx context.Context, c driver.Client) {
	db := ensureDatabase(ctx, c, "statistics_test", nil, t)
	col := ensureCollection(ctx, db, "statistics_test", nil, t)
	doc := UserDoc{
	  "Max",
		50,
  }
	for i := 0; i < 1000; i++ {
		_, err := col.CreateDocument(ctx, doc)
		if err != nil {
		  t.Fatalf("Failed to create new document: %s", describe(err))
		}
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
	_, err = json.Marshal(stats)
	if err != nil {
		t.Fatalf("Cannot marshal statistics to JSON: %s", describe(err))
	}
  //t.Logf("Statistics: %s", string(b))
}

type source int

const (
	user source = iota
	all source  = iota
)

type limits struct {
	Received  float64
  Sent      float64
	RecCount  int64
	SentCount int64
}


// checkTrafficAtLeast compares stats before and after some operation and
// checks that at least some amount of traffic has happened.
func checkTrafficAtLeast(t *testing.T, statsBefore *driver.ServerStatistics, statsAfter *driver.ServerStatistics, which source, lim *limits) {
	var before *driver.ClientStats
	var after *driver.ClientStats
	var name string
	if which == user {
		before = &statsBefore.ClientUser
		after = &statsAfter.ClientUser
		name = "ClientUser"
	} else {
		before = &statsBefore.Client
    after = &statsAfter.Client
		name = "Client"
	}
	diff := after.BytesReceived.Sum - before.BytesReceived.Sum
	if diff < lim.Received {
		t.Errorf("Difference in %s.BytesReceived.Sum is too small (< %f): %f",
		         name, lim.Received, diff)
	}
	diff = after.BytesSent.Sum - before.BytesSent.Sum
	if diff < lim.Sent {
		t.Errorf("Difference in %s.BytesSent.Sum is too small (< %f): %f",
		         name, lim.Sent, diff)
	}
	intDiff := after.BytesReceived.Count - before.BytesReceived.Count
	if intDiff < lim.RecCount {
		t.Errorf("Difference in %s.BytesReceived.Count is too small (< %d): %d",
		         name, lim.RecCount, intDiff)
	}
	intDiff = after.BytesSent.Count - before.BytesSent.Count
	if intDiff < lim.SentCount {
		t.Errorf("Difference in %s.BytesSent.Count is too small (< %d): %d",
		         name, lim.SentCount, intDiff)
	}
}

// checkTrafficAtMost compares stats before and after some operation and
// checks that at most some amount of traffic has happened.
func checkTrafficAtMost(t *testing.T, statsBefore *driver.ServerStatistics, statsAfter *driver.ServerStatistics, which source, lim *limits) {
	var before *driver.ClientStats
	var after *driver.ClientStats
	var name string
	if which == user {
		before = &statsBefore.ClientUser
		after = &statsAfter.ClientUser
		name = "ClientUser"
	} else {
		before = &statsBefore.Client
    after = &statsAfter.Client
		name = "Client"
	}
	diff := after.BytesReceived.Sum - before.BytesReceived.Sum
	if diff > lim.Received {
		t.Errorf("Difference in %s.BytesReceived.Sum is too large (> %f): %f",
		         name, lim.Received, diff)
	}
	diff = after.BytesSent.Sum - before.BytesSent.Sum
	if diff > lim.Sent {
		t.Errorf("Difference in %s.BytesSent.Sum is too large (> %f): %f",
		         name, lim.Sent, diff)
	}
	intDiff := after.BytesReceived.Count - before.BytesReceived.Count
	if intDiff > lim.RecCount {
		t.Errorf("Difference in %s.BytesReceived.Count is too large (> %d): %d",
		         name, lim.RecCount, intDiff)
	}
	intDiff = after.BytesSent.Count - before.BytesSent.Count
	if intDiff > lim.SentCount {
		t.Errorf("Difference in %s.BytesSent.Count is too large (> %d): %d",
		         name, lim.SentCount, intDiff)
	}
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

	doSomeWrites(t, nil, c)

	time.Sleep(time.Second)  // Wait until statistics updated

	statsAfter, err := c.Statistics(ctx)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	checkTrafficAtLeast(t, &statsBefore, &statsAfter, all,
											&limits{Sent: 100000.0, Received: 40000.0,
															SentCount: 1000, RecCount: 1000});

	// Now check if user only stats are there and see if they should have increased:
  if statsBefore.ClientUser.BytesReceived.Counts != nil {
	  t.Logf("New user only statistics API is present, testing...")
		auth := os.Getenv("TEST_AUTHENTICATION")
		if auth == "super:testing" {
			t.Logf("Authentication %s is jwt superuser, expecting no user traffic...", auth)
	    // Traffic is superuser, so nothing should be counted in ClientUser,
			// not even the statistics calls.
			checkTrafficAtMost(t, &statsBefore, &statsAfter, user,
												 &limits{Sent: 0.1, Received: 0.1,
												         SentCount: 0, RecCount: 0});
		} else {
			t.Logf("Authentication %s is not jwt superuser, expecting to see user traffic...", auth)
			// Traffic is either unauthenticated or with password, so there should
			// be traffic in ClientUser
			checkTrafficAtLeast(t, &statsBefore, &statsAfter, user,
												  &limits{Sent: 100000.0, Received: 40000.0,
												          SentCount: 1000, RecCount: 1000});
		}
	} else {
		t.Log("Skipping ClientUser tests for statistics, since API is not present.")
	}
}

// myQueryRequest is used below for a special query test for forwarding.
type myQueryRequest struct {
  Query string  `json:"query"`
	BatchSize int `json:"batchSize,omitempty"`
}

// cursorData is used to dig out the ID of the cursor
type myCursorData struct {
	ID      string `json:"id"`
	HasMore bool   `json:"hasMore,omitempty"`
}

// TestServerStatisticsForwarding tests if Client.Statistics increase
// with traffic in the correct way if queries are forwarded between
// coordinators.
func TestServerStatisticsForwarding(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	_, err := c.Cluster(ctx)
	if driver.IsPreconditionFailed(err) {
		t.Skip("Not a cluster")
	} else if err != nil {
		t.Fatalf("Health failed: %s", describe(err))
	}

	checkEnabled(t, c, ctx)

	conn := c.Connection()
	endpoints := conn.Endpoints()

	if len(endpoints) < 2 {
		t.Fatalf("Did not have at least two endpoints. Giving up.")
	}

	// Do a preliminary test to see if we can do some traffic on one coordinator
	// and not see it on the second one.

	ctx1 := driver.WithEndpoint(context.Background(), endpoints[0])
	ctx2 := driver.WithEndpoint(context.Background(), endpoints[1])

	statsBefore, err := c.Statistics(ctx2)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	// At least 5000 documents in the collection:
	doSomeWrites(t, ctx1, c)
	doSomeWrites(t, ctx1, c)
	doSomeWrites(t, ctx1, c)
	doSomeWrites(t, ctx1, c)
	doSomeWrites(t, ctx1, c)

	time.Sleep(time.Second)  // wait for statistics to settle

	statsAfter, err := c.Statistics(ctx2)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	// No traffic on second coordinator (besides statistics calls):
  checkTrafficAtMost(t, &statsBefore, &statsAfter, all,
	                   &limits{Received: 400, Sent: 4000,
								             RecCount: 2, SentCount: 2})

  	if statsAfter.ClientUser.BytesReceived.Counts == nil {
		t.Skip("Skipping ClientUser tests for statistics, since API is not present.")
	}

	// First ask for a cursor on coordinator 1:
	req, err := conn.NewRequest("POST", "_db/statistics_test/_api/cursor")
	if err != nil {
		t.Fatalf("Error in NewRequest call for cursor: %s", describe(err))
	}
	query := myQueryRequest{
	  Query: "FOR x IN statistics_test RETURN x",
		BatchSize: 1000,
	}
	if _, err := req.SetBody(query); err != nil {
		t.Fatalf("Error in SetBody call for cursor: %s", describe(err))
	}
	resp, err := conn.Do(ctx1, req)
	if err != nil {
		t.Fatalf("Error in Do call for cursor: %s", describe(err))
	}
	var cursorBody myCursorData
	err = resp.ParseBody("", &cursorBody)
	if err != nil || !cursorBody.HasMore {
		t.Fatalf("Error in cursor call: %s", describe(err))
	}

	time.Sleep(time.Second)

	statsBefore1, err := c.Statistics(ctx1)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}
	statsBefore2, err := c.Statistics(ctx2)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	// Now issue a cursor continuation call to the second coordinator:
	req, err = conn.NewRequest("PUT", "_db/statistics_test/_api/cursor/" + cursorBody.ID)
	if err != nil {
		t.Fatalf("Error in NewRequest call for cursor cont: %s", describe(err))
	}
	_, err = conn.Do(ctx2, req)
	if err != nil {
		t.Fatalf("Error in Do call for cursor cont: %s", describe(err))
	}

	time.Sleep(time.Second)  // wait until statistics settled

	statsAfter1, err := c.Statistics(ctx1)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}
	statsAfter2, err := c.Statistics(ctx2)
	if err != nil {
		t.Fatalf("Error in statistics call: %s", describe(err))
	}

	// Second coordinator should not count as user traffic (besides maybe
	// the statistics calls):
	t.Logf("Checking user traffic on coordinator2...")
	checkTrafficAtMost(t, &statsBefore2, &statsAfter2, user,
										 &limits{Received: 400, Sent: 4000,
														 RecCount: 2, SentCount: 2})
	// However, first coordinator should have counted the user traffic,
	// note: it was just a single request with nearly no upload but quite
	// some download:
	auth := os.Getenv("TEST_AUTHENTICATION")
	if auth != "super:testing" {
	  t.Logf("Checking user traffic on coordinator1...")
		checkTrafficAtLeast(t, &statsBefore1, &statsAfter1, user,
												&limits{Received: 0, Sent: 40000,
																RecCount: 1, SentCount: 1})
	} else {
	  t.Logf("Checking traffic on coordinator1...")
		checkTrafficAtLeast(t, &statsBefore1, &statsAfter1, all,
												&limits{Received: 0, Sent: 40000,
																RecCount: 1, SentCount: 1})
		checkTrafficAtMost(t, &statsBefore1, &statsAfter1, user,
												&limits{Received: 0.1, Sent: 0.1,
																RecCount: 0, SentCount: 0})
	}
}

