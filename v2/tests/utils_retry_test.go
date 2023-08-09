//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// defaultTestTimeout is the default timeout for context use in tests
// less than 2 minutes is causing problems on CI
const defaultTestTimeout = 2 * time.Minute

type Timeout func() error

func NewTimeout(f func() error) Timeout {
	return f
}

func (t Timeout) TimeoutT(test testing.TB, timeout, interval time.Duration) {
	require.NoError(test, t.Timeout(timeout, interval))
}

func (t Timeout) Timeout(timeout, interval time.Duration) error {
	timeoutT := time.NewTimer(timeout)
	defer timeoutT.Stop()
	intervalT := time.NewTicker(interval)
	defer intervalT.Stop()

	for {
		err := t()
		if err != nil {
			if _, ok := err.(Interrupt); ok {
				return nil
			}

			return err
		}

		select {
		case <-timeoutT.C:
			return errors.Errorf("Timeouted")
		case <-intervalT.C:
			continue
		}
	}
}

type Interrupt struct {
}

func (i Interrupt) Error() string {
	return "interrupt"
}
