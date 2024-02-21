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

package connection

import (
	"errors"
	"sync"
)

// Deprecated: use NewRoundRobinEndpoints
// NewEndpoints returns Endpoint manager which runs round-robin
func NewEndpoints(e ...string) Endpoint {
	return NewRoundRobinEndpoints(e)
}

// NewRoundRobinEndpoints returns Endpoint manager which runs round-robin
func NewRoundRobinEndpoints(e []string) Endpoint {
	return &roundRobinEndpoints{
		endpoints: e,
	}
}

type roundRobinEndpoints struct {
	lock      sync.Mutex
	endpoints []string
	index     int
}

func (e *roundRobinEndpoints) List() []string {
	return e.endpoints
}

func (e *roundRobinEndpoints) Get(providedEp, _, _ string) (string, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if providedEp != "" {
		return providedEp, nil
	}

	if len(e.endpoints) == 0 {
		return "", errors.New("no endpoints known")
	}

	if e.index >= len(e.endpoints) {
		e.index = 0
	}

	r := e.endpoints[e.index]

	e.index++

	return r, nil
}
