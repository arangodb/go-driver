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
	"encoding/json"
	"io"

	"github.com/arangodb/go-velocypack"
)

type Decoder interface {
	Decode(reader io.Reader, obj interface{}) error
	Encode(writer io.Writer, obj interface{}) error
	Reencode(in, out interface{}) error
}

func getJsonDecoder() Decoder {
	return jsonDecoderObj
}

var jsonDecoderObj Decoder = &jsonDecoder{}

type jsonDecoder struct {
}

func (j jsonDecoder) Decode(reader io.Reader, obj interface{}) error {
	return json.NewDecoder(reader).Decode(obj)
}

func (j jsonDecoder) Encode(writer io.Writer, obj interface{}) error {
	return json.NewEncoder(writer).Encode(obj)
}

func (j jsonDecoder) Reencode(in, out interface{}) error {
	d, err := json.Marshal(in)
	if err != nil {
		return err
	}

	return json.Unmarshal(d, out)
}

func getVPackDecoder() Decoder {
	return vpackDecoderObj
}

var vpackDecoderObj Decoder = &vpackDecoder{}

type vpackDecoder struct {
}

func (v vpackDecoder) Decode(reader io.Reader, obj interface{}) error {
	return velocypack.NewDecoder(reader).Decode(obj)
}

func (v vpackDecoder) Encode(writer io.Writer, obj interface{}) error {
	return velocypack.NewEncoder(writer).Encode(obj)
}

func (v vpackDecoder) Reencode(in, out interface{}) error {
	d, err := velocypack.Marshal(in)
	if err != nil {
		return err
	}

	return velocypack.Unmarshal(d, out)
}
