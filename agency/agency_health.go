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

package agency

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
)

const (
	maxAgentResponseTime                               = time.Second * 10
	keyAllowNoLeader                 driver.ContextKey = "arangodb-agency-allow-no-leader"
	keyAllowDifferentLeaderEndpoints driver.ContextKey = "arangodb-agency-allow-different-leader-endpoints"
)

// agentStatus is a helper structure used in AreAgentsHealthy.
type agentStatus struct {
	IsLeader       bool
	LeaderEndpoint string
	IsResponding   bool
}

// WithAllowNoLeader is used to configure a context to make AreAgentsHealthy
// accept the situation where it finds 0 leaders.
func WithAllowNoLeader(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, keyAllowNoLeader, true)
}

// hasAllowNoLeader returns true when the given context was
// prepared with WithAllowNoLeader.
func hasAllowNoLeader(ctx context.Context) bool {
	return ctx != nil && ctx.Value(keyAllowNoLeader) != nil
}

// WithAllowNoLeader is used to configure a context to make AreAgentsHealthy
// accept the situation where leader endpoint is different (during agency endpoint update).
func WithAllowDifferentLeaderEndpoints(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, keyAllowDifferentLeaderEndpoints, true)
}

// hasAllowNoLeader returns true when the given context was
// prepared with WithAllowDifferentLeaderEndpoints.
func hasAllowDifferentLeaderEndpoints(ctx context.Context) bool {
	return ctx != nil && ctx.Value(keyAllowDifferentLeaderEndpoints) != nil
}

// AreAgentsHealthy performs a health check on all given agents.
// Of the given agents, 1 must respond as leader and all others must redirect to the leader.
// The function returns nil when all agents are healthy or an error when something is wrong.
func AreAgentsHealthy(ctx context.Context, clients []driver.Connection) error {
	wg := sync.WaitGroup{}
	invalidKey := []string{"does-not-exist-70ddb948-59ea-52f3-9a19-baaca18de7ae"}
	statuses := make([]agentStatus, len(clients))
	for i, c := range clients {
		wg.Add(1)
		go func(i int, c driver.Connection) {
			defer wg.Done()
			lctx, cancel := context.WithTimeout(ctx, maxAgentResponseTime)
			defer cancel()
			var result interface{}
			a, err := NewAgency(c)
			if err == nil {
				var resp driver.Response
				lctx = driver.WithResponse(lctx, &resp)
				if err := a.ReadKey(lctx, invalidKey, &result); err == nil || IsKeyNotFound(err) {
					// We got a valid read from the leader
					statuses[i].IsLeader = true
					statuses[i].LeaderEndpoint = strings.Join(c.Endpoints(), ",")
					statuses[i].IsResponding = true
				} else {
					if driver.IsArangoErrorWithCode(err, http.StatusTemporaryRedirect) && resp != nil {
						location := resp.Header("Location")
						// Valid response from a follower
						statuses[i].IsLeader = false
						statuses[i].LeaderEndpoint = location
						statuses[i].IsResponding = true
					} else {
						// Unexpected / invalid response
						statuses[i].IsResponding = false
					}
				}
			}
		}(i, c)
	}
	wg.Wait()

	// Check the results
	noLeaders := 0
	for i, status := range statuses {
		if !status.IsResponding {
			return driver.WithStack(fmt.Errorf("Agent %s is not responding", strings.Join(clients[i].Endpoints(), ",")))
		}
		if status.IsLeader {
			noLeaders++
		}
		if i > 0 {
			if hasAllowDifferentLeaderEndpoints(ctx) {
				continue
			}

			// Compare leader endpoint with previous
			prev := statuses[i-1].LeaderEndpoint
			if !IsSameEndpoint(prev, status.LeaderEndpoint) {
				return driver.WithStack(fmt.Errorf("Not all agents report the same leader endpoint"))
			}
		}
	}
	if noLeaders != 1 && !hasAllowNoLeader(ctx) {
		return driver.WithStack(fmt.Errorf("Unexpected number of agency leaders: %d", noLeaders))
	}
	return nil
}

// IsSameEndpoint returns true when the 2 given endpoints
// refer to the same server.
func IsSameEndpoint(a, b string) bool {
	if a == b {
		return true
	}
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Hostname() == ub.Hostname()
}
