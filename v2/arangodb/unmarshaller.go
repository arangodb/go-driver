//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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

package arangodb

import "encoding/json"

var _ json.Unmarshaler = &Unmarshal[string, int]{}

type Unmarshal[C, T any] struct {
	Current *C

	Object T
}

func (u *Unmarshal[C, T]) UnmarshalJSON(bytes []byte) error {
	u.Current = nil

	var q C

	if err := json.Unmarshal(bytes, &q); err == nil {
		u.Current = &q
	}

	return json.Unmarshal(bytes, &u.Object)
}

type UnmarshalData []byte

func (u UnmarshalData) Inject(object any) error {
	return json.Unmarshal(u, object)
}

func (u *UnmarshalData) UnmarshalJSON(bytes []byte) error {
	z := make([]byte, len(bytes))

	copy(z, bytes)

	*u = z
	return nil
}
