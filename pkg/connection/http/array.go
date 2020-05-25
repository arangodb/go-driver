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
	"encoding/json"
	"io"
	"sync"

	"github.com/arangodb/go-driver/pkg/connection"
)

func newArray(in io.ReadCloser) (connection.Array, error) {
	d := json.NewDecoder(in)

	if _, err := d.Token(); err != nil {
		return nil, err
	}

	return &array{
		body:    in,
		decoder: d,
	}, nil
}

var _ connection.Array = &array{}

type array struct {
	body    io.ReadCloser
	decoder *json.Decoder

	lock sync.Mutex
}

func (a *array) Unmarshal(i interface{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.decoder.Decode(i)
}

func (a *array) More() bool {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.decoder.More()
}

func (a *array) Close() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return dropBodyData(a.body)
}
