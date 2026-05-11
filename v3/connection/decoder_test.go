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

package connection

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_bytesDecoder_Decode(t *testing.T) {
	t.Run("got response", func(t *testing.T) {
		response := "the response"
		var output []byte

		err := bytesDecoder{}.Decode(strings.NewReader(response), &output)
		require.NoError(t, err)
		require.NotNilf(t, output, "the output should be allocated in the decoder")
		assert.Equal(t, string(output), response)
	})

	t.Run("invalid output argument", func(t *testing.T) {
		var output string

		err := bytesDecoder{}.Decode(strings.NewReader("the response"), &output)
		require.EqualError(t, err, ErrReaderOutputBytes.Error())
		assert.Equal(t, "", output, "the output should be empty")
	})
}

func Test_bytesDecoder_Encode(t *testing.T) {
	t.Run("send request", func(t *testing.T) {
		var buf bytes.Buffer
		request := "the request"

		err := bytesDecoder{}.Encode(&buf, []byte(request))

		require.NoError(t, err)
		assert.Equal(t, request, string(buf.Bytes()))
	})

	t.Run("invalid input argument", func(t *testing.T) {
		var buf bytes.Buffer

		err := bytesDecoder{}.Encode(&buf, "string")
		require.EqualError(t, err, ErrWriterInputBytes.Error())
	})
}
