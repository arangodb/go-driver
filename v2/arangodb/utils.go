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

package arangodb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// AlternativeError informs that at least one of the errors must be resolved before continueing.
func newAlternativeError(errs ...*error) error {
	return &AlternativeError{
		errs: errs,
	}
}

type AlternativeError struct {
	errs []*error
}

// Error message for AlternativeError
func (e AlternativeError) Error() string {
	var errorMessages []string
	for _, err := range e.errs {
		errorMessages = append(errorMessages, (*err).Error())
	}
	return fmt.Sprintf("One of the following must be correct:\n- %s", strings.Join(errorMessages, "\n- "))
}

// OrderedMultiUnmarshaller runs all unmarshalers given as the input skipping the UnmarshallerTypeErrors (See GetError)
func newOrderedMultiUnmarshaller(obj ...json.Unmarshaler) *orderedMultiUnmarshaller {
	return &orderedMultiUnmarshaller{
		obj: obj,
	}
}

type orderedMultiUnmarshaller struct {
	obj  []json.Unmarshaler
	errs []*json.UnmarshalTypeError
}

var _ json.Unmarshaler = &orderedMultiUnmarshaller{}

func (m *orderedMultiUnmarshaller) UnmarshalJSON(d []byte) error {
	m.errs = []*json.UnmarshalTypeError{}
	noneSuccessful := true

	for _, o := range m.obj {
		switch err := o.UnmarshalJSON(d); err.(type) {
		case nil:
			m.errs = append(m.errs, nil)
			noneSuccessful = false
		case *json.UnmarshalTypeError:
			m.errs = append(m.errs, err.(*json.UnmarshalTypeError))
		default:
			m.errs = nil
			return err
		}
	}
	if noneSuccessful {
		errs := []*error{}
		for _, e := range m.errs {
			if e != nil {
				eAsError := error(e)
				errs = append(errs, &eAsError)
			}
		}
		return newAlternativeError(errs...)
	}
	return nil
}

// GetError recovers error triggered by the index-th unmarshaller
func (m *orderedMultiUnmarshaller) GetError(index int) *json.UnmarshalTypeError {
	if m.errs == nil {
		return nil
	}
	return m.errs[index]
}

var _ json.Unmarshaler = &multiUnmarshaller{}
var _ json.Marshaler = &multiUnmarshaller{}

func newMultiUnmarshaller(obj ...interface{}) json.Unmarshaler {
	return &multiUnmarshaller{
		obj: obj,
	}
}

type multiUnmarshaller struct {
	obj []interface{}
}

func (m multiUnmarshaller) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{}
	for _, o := range m.obj {
		z := map[string]interface{}{}
		if d, err := json.Marshal(o); err != nil {
			return nil, err
		} else {
			if err := json.Unmarshal(d, &z); err != nil {
				return nil, err
			}
		}

		for k, v := range z {
			r[k] = v
		}
	}

	return json.Marshal(r)
}

func (m multiUnmarshaller) UnmarshalJSON(d []byte) error {
	for _, o := range m.obj {
		if err := json.Unmarshal(d, o); err != nil {
			return err
		}
	}

	return nil
}

type byteDecoder struct {
	data []byte
}

func (b *byteDecoder) UnmarshalJSON(d []byte) error {
	b.data = make([]byte, len(d))

	copy(b.data, d)

	return nil
}

func (b *byteDecoder) Unmarshal(i interface{}) error {
	return json.Unmarshal(b.data, i)
}

func newUnmarshalInto(obj interface{}) *UnmarshalInto {
	return &UnmarshalInto{obj}
}

var _ json.Unmarshaler = &UnmarshalInto{}

type UnmarshalInto struct {
	obj interface{}
}

func (u *UnmarshalInto) UnmarshalJSON(d []byte) error {
	if u.obj == nil {
		return nil
	}

	if reflect.TypeOf(u.obj).Kind() != reflect.Ptr {
		return errors.Errorf("Unable to unmarshal into non ptr")
	}

	return json.Unmarshal(d, u.obj)
}

var _ json.Unmarshaler = &jsonReader{}

type jsonReader struct {
	in       []byte
	inStream *json.Decoder
}

func (j *jsonReader) UnmarshalJSON(d []byte) error {
	j.in = make([]byte, len(d))

	copy(j.in, d)

	j.inStream = json.NewDecoder(bytes.NewReader(j.in))

	if _, err := j.inStream.Token(); err != nil {
		return err
	}

	return nil
}

func (j *jsonReader) Read(i interface{}) error {
	if !j.inStream.More() {
		return io.EOF
	}

	return j.inStream.Decode(i)
}

func (j *jsonReader) HasMore() bool {
	if len(j.in) == 0 {
		return false
	}
	return j.inStream.More()
}

// CreateDocuments creates given number of documents for the provided collection.
func CreateDocuments(ctx context.Context, col Collection, docCount int, generator func(i int) any) error {
	if generator == nil {
		return errors.New("document generator can not be nil")
	}
	if col == nil {
		return errors.New("collection can not be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	docs := make([]any, 0, docCount)
	for i := 0; i < docCount; i++ {
		docs = append(docs, generator(i))
	}

	_, err := col.CreateDocuments(ctx, docs)
	return err
}
