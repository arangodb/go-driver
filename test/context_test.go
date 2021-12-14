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

package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

// TestContextParentNil calls all WithXyz context functions with a nil parent context.
// This must not crash.
func TestContextParentNil(t *testing.T) {
	testValue := func(ctx context.Context) {
		ctx.Value("foo")
	}

	testValue(driver.WithRevision(nil, "rev"))
	testValue(driver.WithRevisions(nil, []string{"rev1", "rev2"}))
	testValue(driver.WithReturnNew(nil, make(map[string]interface{})))
	testValue(driver.WithReturnOld(nil, make(map[string]interface{})))
	testValue(driver.WithDetails(nil))
	testValue(driver.WithKeepNull(nil, false))
	testValue(driver.WithMergeObjects(nil, true))
	testValue(driver.WithSilent(nil))
	testValue(driver.WithWaitForSync(nil))
	testValue(driver.WithRawResponse(nil, &[]byte{}))
	testValue(driver.WithArangoQueueTimeout(nil, true))
	testValue(driver.WithArangoQueueTime(nil, time.Second*5))
}

func TestContextWithArangoQueueTimeoutParams(t *testing.T) {
	c := createClientFromEnv(t, true)

	t.Run("without timout", func(t *testing.T) {
		_, err := c.Version(context.Background())
		require.NoError(t, err)
	})

	t.Run("with context deadLine timeout", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Nanosecond))
		defer cancel()

		ctx = driver.WithArangoQueueTimeout(ctx, true)

		_, err := c.Version(ctx)
		require.Error(t, err)
	})

	t.Run("without timeout - if no queue timeout and no context deadline set", func(t *testing.T) {
		ctx := driver.WithArangoQueueTimeout(context.Background(), true)

		_, err := c.Version(ctx)
		require.NoError(t, err)
	})

	t.Run("with queue param timeout", func(t *testing.T) {
		// TODO: verify that this test works under 3.9
		skipBelowVersion(c, "3.9", t)

		ctx := driver.WithArangoQueueTimeout(context.Background(), true)
		ctx = driver.WithArangoQueueTime(context.Background(), time.Nanosecond)

		_, err := c.Version(ctx)
		require.Error(t, err)
	})
}
