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
//
// Author Adam Janikowski
//

package connection

import "sync"

type Endpoint interface {
	// Get return one of endpoints if is valid, if no default one is returned
	Get(endpoints ...string) (string, bool)

	List() []string
}

func NewEndpoints(e ...string) Endpoint {
	return &endpoints{
		endpoints: e,
	}
}

type endpoints struct {
	lock sync.Mutex

	endpoints []string

	index int
}

func (e *endpoints) List() []string {
	return e.endpoints
}

func (e *endpoints) Get(endpoints ...string) (string, bool) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if len(e.endpoints) == 0 {
		return "", false
	}

	//return e.endpoints[0], true

	if e.index >= len(e.endpoints) {
		e.index = 0
	}

	r := e.endpoints[e.index]

	e.index++

	return r, true
}
