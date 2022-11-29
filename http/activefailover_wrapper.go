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

package http

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
)

type activeFailoverWrapper struct {
	driver.Connection
}

const (
	timeout  = time.Second * 60
	interval = time.Second * 2
)

func NewActiveFailoverWrapper(conn driver.Connection) driver.Connection {
	return &activeFailoverWrapper{conn}
}

// TODO: with Go 1.18 we can extract this method to utils using generics
func (c *activeFailoverWrapper) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	timeoutT := time.NewTimer(timeout)
	defer timeoutT.Stop()

	intervalT := time.NewTicker(interval)
	defer intervalT.Stop()

	for {
		resp, err := c.Connection.Do(ctx, req)
		if err != nil && driver.IsNoLeaderOrOngoing(err) {
			fmt.Printf("there is no Leader or Leader change is ongoing - %v", err)
		} else {
			return resp, err
		}

		select {
		case <-timeoutT.C:
			return nil, fmt.Errorf("function timeouted")
		case <-intervalT.C:
			continue
		}
	}
}
