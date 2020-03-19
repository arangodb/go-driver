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

package vst

import (
	"bytes"
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-velocypack"
	"reflect"
)

type vstBody struct {
	body []byte
}

func NewVstBodyBuilder() *vstBody {
	return &vstBody{}
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBody(body ...interface{}) error {
	switch len(body) {
	case 0:
		return driver.WithStack(fmt.Errorf("Must provide at least 1 body"))
	case 1:
		if data, err := velocypack.Marshal(body[0]); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	default:
		slices := make([]velocypack.Slice, len(body))
		for i, b := range body {
			var err error
			slices[i], err = velocypack.Marshal(b)
			if err != nil {
				return driver.WithStack(err)
			}
		}
		merged, err := velocypack.Merge(slices...)
		if err != nil {
			return driver.WithStack(err)
		}
		b.body = merged
		return nil
	}
	return nil
}

// SetBodyArray sets the content of the request as an array.
// If the given mergeArray is not nil, its elements are merged with the elements in the body array (mergeArray data overrides bodyArray data).
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	if mergeArray == nil {
		// Simple case; just marshal bodyArray directly.
		if data, err := velocypack.Marshal(bodyArray); err != nil {
			return driver.WithStack(err)
		} else {
			b.body = data
		}
		return nil
	}

	// Complex case, mergeArray is not nil
	builder := velocypack.Builder{}
	// Start array
	if err := builder.OpenArray(); err != nil {
		return driver.WithStack(err)
	}

	elementCount := bodyArrayVal.Len()
	for i := 0; i < elementCount; i++ {
		// Marshal body element
		bodySlice, err := velocypack.Marshal(bodyArrayVal.Index(i).Interface())
		if err != nil {
			return driver.WithStack(err)
		}
		var sliceToAdd velocypack.Slice
		if maElem := mergeArray[i]; maElem != nil {
			// Marshal merge array element
			elemSlice, err := velocypack.Marshal(maElem)
			if err != nil {
				return driver.WithStack(err)
			}
			// Merge elemSlice with bodySlice
			sliceToAdd, err = velocypack.Merge(elemSlice, bodySlice)
			if err != nil {
				return driver.WithStack(err)
			}
		} else {
			// Just use bodySlice
			sliceToAdd = bodySlice
		}

		// Add resulting slice
		if err := builder.AddValue(velocypack.NewSliceValue(sliceToAdd)); err != nil {
			return driver.WithStack(err)
		}
	}

	// Close array
	if err := builder.Close(); err != nil {
		return driver.WithStack(err)
	}

	// Get resulting slice
	arraySlice, err := builder.Slice()
	if err != nil {
		return driver.WithStack(err)
	}

	b.body = arraySlice
	return nil
}

// SetBodyImportArray sets the content of the request as an array formatted for importing documents.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *vstBody) SetBodyImportArray(bodyArray interface{}) error {
	bodyArrayVal := reflect.ValueOf(bodyArray)
	switch bodyArrayVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return driver.WithStack(driver.InvalidArgumentError{Message: fmt.Sprintf("bodyArray must be slice, got %s", bodyArrayVal.Kind())})
	}
	// Render elements
	buf := &bytes.Buffer{}
	encoder := velocypack.NewEncoder(buf)
	if err := encoder.Encode(bodyArray); err != nil {
		return driver.WithStack(err)
	}

	b.body = buf.Bytes()
	return nil
}

func (b *vstBody) GetBody() []byte {
	return b.body
}

func (b *vstBody) GetContentType() string {
	return ""
}

func (b *vstBody) Clone() driver.BodyBuilder {
	return &vstBody{
		body: b.GetBody(),
	}
}
