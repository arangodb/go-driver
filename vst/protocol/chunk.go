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
	"fmt"
	"io"
)

// chunk is a part of a larger message.
type chunk struct {
	chunkX        uint32
	MessageID     uint64
	MessageLength uint64
	Data          []byte
}

const (
	minChunkHeaderSize = 16
	maxChunkHeaderSize = 24
)

// buildChunks splits a message consisting of 1 or more parts into chunks.
func buildChunks(messageID uint64, maxChunkSize uint32, messageParts ...[]byte) ([]chunk, error) {
	if maxChunkSize <= maxChunkHeaderSize {
		return nil, fmt.Errorf("maxChunkSize is too small (%d)", maxChunkSize)
	}
	messageLength := uint64(0)
	for _, m := range messageParts {
		messageLength += uint64(len(m))
	}
	minChunkCount := int(messageLength / uint64(maxChunkSize))
	maxDataLength := int(maxChunkSize - maxChunkHeaderSize)
	chunks := make([]chunk, 0, minChunkCount+len(messageParts))
	chunkIndex := uint32(0)
	for _, m := range messageParts {
		offset := 0
		remaining := len(m)
		for remaining > 0 {
			dataLength := remaining
			if dataLength > maxDataLength {
				dataLength = maxDataLength
			}
			chunkX := chunkIndex << 1
			c := chunk{
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
	if len(chunks) == 1 {
		chunks[0].chunkX = 3
	} else {
		chunks[0].chunkX = uint32((len(chunks) << 1) + 1)
	}
	return chunks, nil
}

// readBytes tries to read len(dst) bytes into dst.
func readBytes(dst []byte, r io.Reader) error {
	offset := 0
	remaining := len(dst)
	if remaining == 0 {
		return nil
	}
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
func (c chunk) IsFirst() bool {
	return (c.chunkX & 0x01) == 1
}

// Index return the index of this chunk in the message.
func (c chunk) Index() uint32 {
	if (c.chunkX & 0x01) == 1 {
		return 0
	}
	return c.chunkX >> 1
}

// NumberOfChunks return the number of chunks that make up the entire message.
// This function is only valid for first chunks.
func (c chunk) NumberOfChunks() uint32 {
	if (c.chunkX & 0x01) == 1 {
		return c.chunkX >> 1
	}
	return 0 // Not known
}
