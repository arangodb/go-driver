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

package arangodb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

const (
	maxAgentResponseTime = time.Second * 10
)

type contextKey string

const (
	keyAllowNoLeader                 contextKey = "arangodb-agency-allow-no-leader"
	keyAllowDifferentLeaderEndpoints contextKey = "arangodb-agency-allow-different-leader-endpoints"
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

// WithAllowDifferentLeaderEndpoints is used to configure a context to make AreAgentsHealthy
// accept the situation where leader endpoint is different (during agency endpoint update).
func WithAllowDifferentLeaderEndpoints(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, keyAllowDifferentLeaderEndpoints, true)
}

// hasAllowDifferentLeaderEndpoints returns true when the given context was
// prepared with WithAllowDifferentLeaderEndpoints.
func hasAllowDifferentLeaderEndpoints(ctx context.Context) bool {
	return ctx != nil && ctx.Value(keyAllowDifferentLeaderEndpoints) != nil
}

// AreAgentsHealthy performs a health check on all given agency connections.
// Of the given connections, 1 must respond as leader and all others must redirect to the leader.
// The function returns nil when all agents are healthy or an error when something is wrong.
func AreAgentsHealthy(ctx context.Context, clients []Client) error {
	wg := sync.WaitGroup{}
	invalidKey := []string{"does-not-exist-70ddb948-59ea-52f3-9a19-baaca18de7ae"}
	statuses := make([]agentStatus, len(clients))
	for i, c := range clients {
		wg.Add(1)
		go func(i int, c Client) {
			defer wg.Done()
			lctx, cancel := context.WithTimeout(ctx, maxAgentResponseTime)
			defer cancel()

			// Store original endpoint before ReadKey call
			originalEndpoints := c.Connection().GetEndpoint().List()
			originalEndpointStr := strings.Join(originalEndpoints, ",")

			// Use ReadKey - it will handle redirects automatically
			var result interface{}
			err := c.ReadKey(lctx, invalidKey, &result)

			// Check endpoint after ReadKey call to detect if redirect occurred
			currentEndpoints := c.Connection().GetEndpoint().List()
			currentEndpointStr := strings.Join(currentEndpoints, ",")

			if err == nil || IsKeyNotFound(err) {
				// ReadKey succeeded - check if endpoint changed (indicating redirect)
				if originalEndpointStr != currentEndpointStr {
					// Endpoint changed - this was a redirect (follower)
					statuses[i].IsLeader = false
					statuses[i].IsResponding = true
					statuses[i].LeaderEndpoint = currentEndpointStr
				} else {
					// Endpoint unchanged - this is the leader
					statuses[i].IsLeader = true
					statuses[i].IsResponding = true
					statuses[i].LeaderEndpoint = originalEndpointStr
				}
			} else {
				// ReadKey failed - check if it's a redirect error
				if shared.IsArangoErrorWithCode(err, http.StatusTemporaryRedirect) {
					// Redirect error - this is a follower
					statuses[i].IsLeader = false
					statuses[i].IsResponding = true
					// Try to get leader endpoint from current endpoint (if it was updated)
					if currentEndpointStr != originalEndpointStr {
						statuses[i].LeaderEndpoint = currentEndpointStr
					} else {
						statuses[i].LeaderEndpoint = ""
					}
				} else {
					// Other error - not responding
					statuses[i].IsResponding = false
				}
			}
		}(i, c)
	}
	wg.Wait()

	// Check the results
	noLeaders := 0
	for i, status := range statuses {
		if !status.IsResponding {
			endpoints := clients[i].Connection().GetEndpoint().List()
			return fmt.Errorf("Agent %s is not responding", strings.Join(endpoints, ","))
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
				return errors.New("Not all agents report the same leader endpoint")
			}
		}
	}
	// During upgrades, multiple leaders might temporarily exist
	// If WithAllowDifferentLeaderEndpoints is set, we're more lenient about leader count
	if noLeaders != 1 {
		if hasAllowNoLeader(ctx) {
			// Allow 0 leaders
			if noLeaders == 0 {
				return nil
			}
		}
		// If we're allowing different leader endpoints (upgrade scenario),
		// also allow multiple leaders temporarily
		if hasAllowDifferentLeaderEndpoints(ctx) && noLeaders > 1 {
			// During upgrades, multiple leaders might exist temporarily
			// Log but don't fail - this is expected during upgrades
			return nil
		}
		return fmt.Errorf("Unexpected number of agency leaders: %d", noLeaders)
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
