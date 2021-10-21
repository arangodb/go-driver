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
	"context"
	"fmt"
)

func NewHeaderAuth(key, value string, args ...interface{}) Authentication {
	return &headerAuth{
		key:   key,
		value: fmt.Sprintf(value, args...),
	}
}

type headerAuth struct {
	key, value string
}

func (b headerAuth) Init(ctx context.Context, c Connection) error {
	return nil
}

func (b headerAuth) Refresh(c Connection) error {
	return nil
}

func (b headerAuth) RequestModifier(r Request) error {
	r.AddHeader(b.key, b.value)

	return nil
}
