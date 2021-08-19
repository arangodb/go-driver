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
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/agency"
)

// TestAgencyLock tests the agency.Lock interface.
func TestAgencyLock(t *testing.T) {
	t.Fatal("Test failure")

	ctx := context.Background()
	c := createClientFromEnv(t, true)
	if a, err := getAgencyConnection(ctx, t, c); driver.IsPreconditionFailed(err) {
		t.Skipf("Skip agency test: %s", describe(err))
	} else if err != nil {
		t.Fatalf("Cluster failed: %s", describe(err))
	} else {
		key := []string{"go-driver", "TestAgencyLock"}
		l, err := agency.NewLock(t, a, key, "2b2173ae-6684-501c-b8b1-c8b754b7fd40", time.Minute)
		if err != nil {
			t.Fatalf("NewLock failed: %s", describe(err))
		}
		if l.IsLocked() {
			t.Error("IsLocked must be false, got true")
		}
		if err := l.Lock(ctx); err != nil {
			t.Fatalf("Lock failed: %s", describe(err))
		}
		if !l.IsLocked() {
			t.Error("IsLocked must be true, got false")
		}
		if err := l.Lock(ctx); !agency.IsAlreadyLocked(err) {
			t.Fatalf("AlreadyLockedError expected, got %s", describe(err))
		}
		if err := l.Unlock(ctx); err != nil {
			t.Fatalf("Unlock failed: %s", describe(err))
		}
		if l.IsLocked() {
			t.Error("IsLocked must be false, got true")
		}
		if err := l.Unlock(ctx); !agency.IsNotLocked(err) {
			t.Fatalf("NotLockedError expected, got %s", describe(err))
		}
		if err := l.Lock(ctx); err != nil {
			t.Fatalf("Lock failed: %s", describe(err))
		}
		if !l.IsLocked() {
			t.Error("IsLocked must be true, got false")
		}
		if err := l.Unlock(ctx); err != nil {
			t.Fatalf("Unlock failed: %s", describe(err))
		}
		if l.IsLocked() {
			t.Error("IsLocked must be false, got true")
		}
	}
}
