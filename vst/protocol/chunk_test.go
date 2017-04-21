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
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"
)

type readChunksTest struct {
	ChunkHex       string
	MessageID      uint64
	MessageLength  uint64
	IsFirst        bool
	Index          uint32
	NumberOfChunks uint32
	Data           []byte
}

func TestReadChunk(t *testing.T) {
	tests := []readChunksTest{
		readChunksTest{
			ChunkHex:       "1b0000000900000037020000000000000c00000000000000010203",
			MessageID:      567,
			MessageLength:  12,
			IsFirst:        true,
			Index:          0,
			NumberOfChunks: 4,
			Data:           []byte{1, 2, 3},
		},
		readChunksTest{
			ChunkHex:       "1b0000000200000037020000000000000c00000000000000040506",
			MessageID:      567,
			MessageLength:  12,
			IsFirst:        false,
			Index:          1,
			NumberOfChunks: 0,
			Data:           []byte{4, 5, 6},
		},
	}

	for _, test := range tests {
		raw, err := hex.DecodeString(test.ChunkHex)
		if err != nil {
			t.Fatalf("Hex decode failed: %#v", err)
		}
		r := bytes.NewReader(raw)
		c, err := readChunk(r)
		if err != nil {
			t.Errorf("ReadChunk for '%s' failed: %#v", test.ChunkHex, err)
		}
		if c.IsFirst() != test.IsFirst {
			t.Errorf("IsFirst for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.IsFirst(), test.IsFirst)
		}
		if c.Index() != test.Index {
			t.Errorf("Index for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.Index(), test.Index)
		}
		if c.NumberOfChunks() != test.NumberOfChunks {
			t.Errorf("NumberOfChunks for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.NumberOfChunks(), test.NumberOfChunks)
		}
		if c.MessageID != test.MessageID {
			t.Errorf("MessageID for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.MessageID, test.MessageID)
		}
		if c.MessageLength != test.MessageLength {
			t.Errorf("MessageLength for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.MessageLength, test.MessageLength)
		}
		if !reflect.DeepEqual(c.Data, test.Data) {
			t.Errorf("Data for '%s' is invalid. \nGot '%v'\nExpected '%v'", test.ChunkHex, c.Data, test.Data)
		}
	}
}

type buildChunksTest struct {
	MessageID         uint64
	MaxChunkSize      uint32
	MessageParts      [][]byte
	ExpectedChunksHex []string
}

func TestBuildChunks(t *testing.T) {
	tests := []buildChunksTest{
		buildChunksTest{
			MessageID:    567,
			MaxChunkSize: 24 + 3,
			MessageParts: [][]byte{
				[]byte{1, 2, 3},
				[]byte{4, 5, 6},
				[]byte{7, 8, 9, 10, 11, 12},
			},
			ExpectedChunksHex: []string{
				"1b0000000900000037020000000000000c00000000000000010203",
				"1b0000000200000037020000000000000c00000000000000040506",
				"1b0000000400000037020000000000000c00000000000000070809",
				"1b0000000600000037020000000000000c000000000000000a0b0c",
			},
		},
		buildChunksTest{
			MessageID:    567,
			MaxChunkSize: 24 + 6,
			MessageParts: [][]byte{
				[]byte{1, 2, 3},
				[]byte{4, 5, 6},
				[]byte{7, 8, 9, 10, 11, 12},
			},
			ExpectedChunksHex: []string{
				"1b0000000700000037020000000000000c00000000000000010203",
				"1b0000000200000037020000000000000c00000000000000040506",
				"1e0000000400000037020000000000000c000000000000000708090a0b0c",
			},
		},
	}

	for _, test := range tests {
		chunks, err := buildChunks(test.MessageID, test.MaxChunkSize, test.MessageParts...)
		if err != nil {
			t.Fatalf("BuildChunks failed: %#v", err)
		}

		if len(chunks) != len(test.ExpectedChunksHex) {
			t.Errorf("Expected %d chunks, got %d", len(test.ExpectedChunksHex), len(chunks))
		}
		for i, expected := range test.ExpectedChunksHex {
			if i >= len(chunks) {
				continue
			}
			var buf bytes.Buffer
			if _, err := chunks[i].WriteTo(&buf); err != nil {
				t.Errorf("Failed to write chunk %d: %#v", i, err)
			}
			actual := hex.EncodeToString(buf.Bytes())
			if expected != actual {
				t.Errorf("Chunk %d is invalid. \nGot '%s'\nExpected '%s'", i, actual, expected)
			}
		}
	}
}
