//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

package utils

import "reflect"

func IsListPtr(i interface{}) bool {
	t := reflect.ValueOf(i)
	switch t.Kind() {
	case reflect.Ptr:
		return IsList(t.Elem().Interface())
	default:
		return false
	}
}

func IsList(i interface{}) bool {
	switch reflect.ValueOf(i).Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		return true
	default:
		return false
	}
}

func NewType[T any](val T) *T {
	return &val
}
