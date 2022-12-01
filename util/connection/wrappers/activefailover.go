//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package wrappers

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
)

const (
	timeout  = time.Second * 60
	interval = time.Second * 2
)

type activeFailoverWrapper struct {
	testEnv
	driver.Connection
}

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

func NewActiveFailoverWrapper(t testEnv, conn driver.Connection) driver.Connection {
	return &activeFailoverWrapper{t, conn}
}

// TODO: with Go 1.18 we can extract this method to utils using generics
func (af *activeFailoverWrapper) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	timeoutT := time.NewTimer(timeout)
	defer timeoutT.Stop()

	intervalT := time.NewTicker(interval)
	defer intervalT.Stop()

	for {
		resp, err := af.Connection.Do(ctx, req)
		if err != nil {
			if driver.IsNoLeaderOrOngoing(err) {
				af.Logf("RETRYING (ERROR) - there is no Leader or Leader change is ongoing: %v", err)
			} else {
				return resp, err
			}
		} else {
			if errBody := resp.CheckStatus(); errBody != nil && driver.IsNoLeaderOrOngoing(errBody) {
				af.Logf("RETRYING (ERROR IN BODY CASE) - there is no Leader or Leader change is ongoing: %v", errBody)
			} else {
				return resp, err
			}
		}

		select {
		case <-timeoutT.C:
			return nil, fmt.Errorf("activeFailoverWrapper function time out (waiting for the Leader)")
		case <-intervalT.C:
			continue
		}
	}
}
