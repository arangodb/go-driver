//
// DISCLAIMER
//
// Copyright 2017-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dchest/uniuri"
	"github.com/google/uuid"
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
	Name() string
	FailNow()
}

func NewUUID() string {
	return uuid.New().String()
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

type retryFunc func() error

func (r retryFunc) RetryT(t *testing.T, interval, timeout time.Duration) {
	require.NoError(t, r.Retry(interval, timeout))
}

func (r retryFunc) Retry(interval, timeout time.Duration) error {
	timeoutT := time.NewTimer(timeout)
	defer timeoutT.Stop()

	intervalT := time.NewTicker(interval)
	defer intervalT.Stop()

	for {
		if err := r(); err != nil {
			if _, ok := err.(interrupt); ok {
				return nil
			}

			return err
		}

		select {
		case <-timeoutT.C:
			return fmt.Errorf("function timeouted")
		case <-intervalT.C:
			continue
		}
	}
}

func newRetryFunc(f func() error) retryFunc {
	return f
}

func retry(interval, timeout time.Duration, f func() error) error {
	return newRetryFunc(f).Retry(interval, timeout)
}

const bulkSize = 1000

func sendBulks(t *testing.T, col driver.Collection, ctx context.Context, creator func(t *testing.T, i int) interface{}, size int) {
	current := 0
	t.Logf("Creating %d documents", size)

	for {
		t.Logf("Created %d/%d documents", current, size)
		stepSize := min(bulkSize, size-current)
		if stepSize == 0 {
			return
		}

		objs := make([]interface{}, min(bulkSize, stepSize))
		for i := 0; i < stepSize; i++ {
			objs[i] = creator(t, current+i)
		}

		_, _, err := col.CreateDocuments(ctx, objs)
		t.Logf("Creating %d documents", len(objs))
		require.NoError(t, err)

		current += stepSize
	}
}

func min(max int, ints ...int) int {
	z := max

	for _, i := range ints {
		if z > i {
			z = i
		}
	}

	return z
}

// getThisFunctionName returns the name of the function of the caller.
func getThisFunctionName() string {
	programCounters := make([]uintptr, 10)
	// skip this function and 'runtime.Callers' function
	runtime.Callers(2, programCounters)
	functionPackage := runtime.FuncForPC(programCounters[0])

	function := strings.Split(functionPackage.Name(), ".")
	if len(function) > 1 {
		return function[len(function)-1] + "_" + uniuri.NewLen(6)
	}

	return function[0] + "_" + uniuri.NewLen(6)
}
