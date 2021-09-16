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
