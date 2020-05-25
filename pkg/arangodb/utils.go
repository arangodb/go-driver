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

package arangodb

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"

	"github.com/pkg/errors"
)

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

func newUnmarshalInto(obj interface{}) *unmarshalInto {
	return &unmarshalInto{obj}
}

var _ json.Unmarshaler = &unmarshalInto{}

type unmarshalInto struct {
	obj interface{}
}

func (u *unmarshalInto) UnmarshalJSON(d []byte) error {
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
