//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/arangodb/go-driver"
	"reflect"
)

type jsonBody struct {
	body []byte
}

func NewJsonBodyBuilder() *jsonBody {
	return &jsonBody{}
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *jsonBody) SetBody(body ...interface{}) error {
	switch len(body) {
	case 0:
		return driver.WithStack(fmt.Errorf("Must provide at least 1 body"))
	case 1:
		if data, err := json.Marshal(body[0]); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	case 2:
		mo := mergeObject{Object: body[1], Merge: body[0]}
		if data, err := json.Marshal(mo); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	default:
		return driver.WithStack(fmt.Errorf("Must provide at most 2 bodies"))
	}

}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *jsonBody) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	if mergeArray == nil {
		// Simple case; just marshal bodyArray directly.
		if data, err := json.Marshal(bodyArray); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	}
	// Complex case, mergeArray is not nil
	elementCount := bodyArrayVal.Len()
	mergeObjects := make([]mergeObject, elementCount)
	for i := 0; i < elementCount; i++ {
		mergeObjects[i] = mergeObject{
			Object: bodyArrayVal.Index(i).Interface(),
			Merge:  mergeArray[i],
		}
	}
	// Now marshal merged array
	if data, err := json.Marshal(mergeObjects); err != nil {
		return driver.WithStack(err)
	} else {
		b.body = data
	}
	return nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *jsonBody) SetBodyImportArray(bodyArray interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	// Render elements
	elementCount := bodyArrayVal.Len()
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	for i := 0; i < elementCount; i++ {
		entryVal := bodyArrayVal.Index(i)
		if isNil(entryVal) {
			buf.WriteString("\n")
		} else {
			if err := encoder.Encode(entryVal.Interface()); err != nil {
				return driver.WithStack(err)
			}
		}
	}
	b.body = buf.Bytes()
	return nil
}

func (b *jsonBody) GetBody() []byte {
	return b.body
}

func (b *jsonBody) GetContentType() string {
	return "application/json"
}

func (b *jsonBody) Clone() driver.BodyBuilder {
	return &jsonBody{
		body: b.GetBody(),
	}
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
