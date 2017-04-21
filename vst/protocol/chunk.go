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
// Author Ewout Prangsma
//

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	driver "github.com/arangodb/go-driver"
)

// Chunk is a part of a larger message.
type Chunk struct {
	chunkX        uint32
	MessageID     uint64
	MessageLength uint64
	Data          []byte
}

const (
	chunkHeaderSize = 24
)

// ReadChunk reads an entire chunk from the given reader.
func ReadChunk(r io.Reader) (Chunk, error) {
	hdr := [chunkHeaderSize]byte{}
	if err := readBytes(hdr[:], r); err != nil {
		return Chunk{}, driver.WithStack(err)
	}
	le := binary.LittleEndian
	length := le.Uint32(hdr[0:])
	chunkX := le.Uint32(hdr[4:])
	messageID := le.Uint64(hdr[8:])
	messageLength := le.Uint64(hdr[16:])

	data := make([]byte, length-chunkHeaderSize)
	if err := readBytes(data, r); err != nil {
		return Chunk{}, driver.WithStack(err)
	}
	return Chunk{
		chunkX:        chunkX,
		MessageID:     messageID,
		MessageLength: messageLength,
		Data:          data,
	}, nil
}

// BuildChunks splits a message consisting of 1 or more parts into chunks.
func BuildChunks(messageID uint64, maxChunkSize uint32, messageParts ...[]byte) ([]Chunk, error) {
	if maxChunkSize <= chunkHeaderSize {
		return nil, fmt.Errorf("maxChunkSize is too small (%d)", maxChunkSize)
	}
	messageLength := uint64(0)
	for _, m := range messageParts {
		messageLength += uint64(len(m))
	}
	minChunkCount := int(messageLength / uint64(maxChunkSize))
	maxDataLength := int(maxChunkSize - chunkHeaderSize)
	chunks := make([]Chunk, 0, minChunkCount+len(messageParts))
	chunkIndex := uint32(0)
	for _, m := range messageParts {
		offset := 0
		remaining := len(m)
		for remaining > 0 {
			dataLength := remaining
			if dataLength > maxDataLength {
				dataLength = maxDataLength
			}
			var chunkX uint32
			if chunkIndex > 0 {
				chunkX = chunkIndex << 1
			}
			c := Chunk{
				chunkX:        chunkX,
				MessageID:     messageID,
				MessageLength: messageLength,
				Data:          m[offset : offset+dataLength],
			}
			chunks = append(chunks, c)
			remaining -= dataLength
			offset += dataLength
			chunkIndex++
		}
	}
	// Set chunkX of first chunk
	chunks[0].chunkX = uint32((len(chunks) << 1) + 1)
	return chunks, nil
}

// readBytes tries to read len(dst) bytes into dst.
func readBytes(dst []byte, r io.Reader) error {
	offset := 0
	remaining := len(dst)
	for {
		n, err := r.Read(dst[offset:])
		offset += n
		remaining -= n
		if remaining == 0 {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// IsFirst returns true when the "first chunk" flag has been set.
func (c Chunk) IsFirst() bool {
	return (c.chunkX & 0x01) == 1
}

// Index return the index of this chunk in the message.
func (c Chunk) Index() uint32 {
	if (c.chunkX & 0x01) == 1 {
		return 0
	}
	return c.chunkX >> 1
}

// NumberOfChunks return the number of chunks that make up the entire message.
// This function is only valid for first chunks.
func (c Chunk) NumberOfChunks() uint32 {
	if (c.chunkX & 0x01) == 1 {
		return c.chunkX >> 1
	}
	return 0 // Not known
}

// WriteTo write the chunk to the given writer.
// An error is returned when less than the entire chunk was written.
func (c Chunk) WriteTo(w io.Writer) (int64, error) {
	hdr := [chunkHeaderSize]byte{}

	le := binary.LittleEndian
	le.PutUint32(hdr[0:], uint32(len(c.Data)+chunkHeaderSize)) // length
	le.PutUint32(hdr[4:], c.chunkX)                            // chunkX
	le.PutUint64(hdr[8:], c.MessageID)                         // message ID
	le.PutUint64(hdr[16:], c.MessageLength)                    // message length

	// Write header
	if n, err := w.Write(hdr[:]); err != nil {
		return int64(n), driver.WithStack(err)
	}

	// Write data
	n, err := w.Write(c.Data)
	result := int64(n) + chunkHeaderSize
	if err != nil {
		return result, driver.WithStack(err)
	}
	return result, nil
}
