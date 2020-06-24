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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	driver "github.com/arangodb/go-driver"
)

type testEnv interface {
	Error(message ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(message ...interface{})
	Fatalf(format string, args ...interface{})
	Log(message ...interface{})
	Logf(format string, args ...interface{})
}

// boolRef returns a reference to a given boolean
func boolRef(v bool) *bool {
	return &v
}

// assertOK fails the test if the given error is not nil.
func assertOK(err error, t *testing.T) {
	if err != nil {
		t.Fatalf("Assertion failed: %s", describe(err))
	}
}

// describe returns a string description of the given error.
func describe(err error) string {
	if err == nil {
		return "nil"
	}
	cause := driver.Cause(err)
	var msg string
	if re, ok := cause.(*driver.ResponseError); ok {
		msg = re.Error()
	} else {
		c, _ := json.Marshal(cause)
		msg = string(c)
	}
	if cause.Error() != err.Error() {
		return fmt.Sprintf("%v caused by %v (%v)", err, cause, msg)
	}
	return fmt.Sprintf("%v (%v)", err, msg)
}

func formatRawResponse(raw []byte) string {
	l := len(raw)
	if l < 2 {
		return hex.EncodeToString(raw)
	}
	if (raw[0] == '{' && raw[l-1] == '}') || (raw[0] == '[' && raw[l-1] == ']') {
		return string(raw)
	}
	return hex.EncodeToString(raw)
}

// getIntFromEnv looks for an environment variable with given key.
// If found, it parses the value to an int, if success that value is returned.
// In all other cases, the given default value is returned.
func getIntFromEnv(envKey string, defaultValue int) int {
	v := strings.TrimSpace(os.Getenv(envKey))
	if v != "" {
		if result, err := strconv.Atoi(v); err == nil {
			return result
		}
	}
	return defaultValue
}

const (
	testModeCluster         = "cluster"
	testModeResilientSingle = "resilientsingle"
	testModeSingle          = "single"
)

func getTestMode() string {
	return strings.TrimSpace(os.Getenv("TEST_MODE"))
}

func skipNoEnterprise(t *testing.T) {
	c := createClientFromEnv(t, true)
	if v, err := c.Version(nil); err != nil {
		t.Errorf("Failed to get version: %s", describe(err))
	} else if !v.IsEnterprise() {
		t.Skipf("Enterprise only")
	}
}

type interrupt struct {
}

func (i interrupt) Error() string {
	return "interrupted"
}

func retry(interval, timeout time.Duration, f func() error) error {
	timeoutT := time.NewTimer(timeout)
	defer timeoutT.Stop()

	intervalT := time.NewTicker(interval)
	defer intervalT.Stop()

	for {
		select {
		case <-timeoutT.C:
			return fmt.Errorf("function timeouted")
		case <-intervalT.C:
			if err := f(); err != nil {
				if _, ok := err.(interrupt); ok {
					return nil
				}

				return err
			}
		}
	}
}

const bulkSize = 1000

func sendBulks(t *testing.T, col driver.Collection, ctx context.Context, creator func(t *testing.T, i int) interface{}, size int) {
	current := 0

	for {
		stepSize := min(bulkSize, size-current)
		if stepSize == 0 {
			return
		}

		objs := make([]interface{}, min(bulkSize, stepSize))
		for i := 0; i < stepSize; i++ {
			objs[i] = creator(t, current+i)
		}

		_, _, err := col.CreateDocuments(ctx, objs)
		require.NoError(t, err)

		current += stepSize
	}
}

func min(ints ...int) int {
	z := 0

	for _, i := range ints {
		if z > i {
			z = i
		}
	}

	return z
}
