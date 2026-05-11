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

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

var _ json.Unmarshaler = &Array{}

type Array struct {
	decoder *json.Decoder

	lock sync.Mutex
}

func (a *Array) UnmarshalJSON(d []byte) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	data := make([]byte, len(d))
	for id, b := range d {
		data[id] = b
	}

	in := bytes.NewReader(data)

	decoder := json.NewDecoder(in)

	if _, err := decoder.Token(); err != nil {
		return err
	}

	a.decoder = decoder

	return nil
}

func (a *Array) Unmarshal(i interface{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a == nil {
		return io.EOF
	}

	return a.decoder.Decode(i)
}

func (a *Array) More() bool {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.decoder.More()
}
