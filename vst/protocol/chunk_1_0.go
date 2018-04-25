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
	"io"

	driver "github.com/arangodb/go-driver"
)

// readChunkVST1_0 reads an entire chunk from the given reader in VST 1.0 format.
func readChunkVST1_0(r io.Reader) (chunk, error) {
	hdr := [maxChunkHeaderSize]byte{}
	if err := readBytes(hdr[:minChunkHeaderSize], r); err != nil {
		return chunk{}, driver.WithStack(err)
	}
	le := binary.LittleEndian
	length := le.Uint32(hdr[0:])
	chunkX := le.Uint32(hdr[4:])
	messageID := le.Uint64(hdr[8:])
	var messageLength uint64
	contentLength := length - minChunkHeaderSize

	if (1 == (chunkX & 0x1)) && ((chunkX >> 1) > 1) {
		// First chunk, numberOfChunks>1 -> read messageLength
		//fmt.Println("Reading maxHdr")
		if err := readBytes(hdr[minChunkHeaderSize:], r); err != nil {
			return chunk{}, driver.WithStack(err)
		}
		messageLength = le.Uint64(hdr[16:])
		contentLength = length - maxChunkHeaderSize
	}

	data := make([]byte, contentLength)
	if err := readBytes(data, r); err != nil {
		return chunk{}, driver.WithStack(err)
	}
	//fmt.Printf("data: " + hex.EncodeToString(data) + "\n")
	return chunk{
		chunkX:        chunkX,
		MessageID:     messageID,
		MessageLength: messageLength,
		Data:          data,
	}, nil
}

// WriteToVST1_0 write the chunk to the given writer in VST 1.0 format.
// An error is returned when less than the entire chunk was written.
func (c chunk) WriteToVST1_0(w io.Writer) (int64, error) {
	le := binary.LittleEndian
	hdrArr := [maxChunkHeaderSize]byte{}
	var hdr []byte
	if c.IsFirst() && c.NumberOfChunks() > 1 {
		// Use extended header
		hdr = hdrArr[:maxChunkHeaderSize]
		le.PutUint64(hdr[16:], c.MessageLength) // message length
	} else {
		// Use minimal header
		hdr = hdrArr[:minChunkHeaderSize]
	}

	le.PutUint32(hdr[0:], uint32(len(c.Data)+len(hdr))) // length
	le.PutUint32(hdr[4:], c.chunkX)                     // chunkX
	le.PutUint64(hdr[8:], c.MessageID)                  // message ID

	// Write header
	//fmt.Printf("Writing hdr: %s\n", hex.EncodeToString(hdr))
	if n, err := w.Write(hdr); err != nil {
		return int64(n), driver.WithStack(err)
	}

	// Write data
	//fmt.Printf("Writing data: %s\n", hex.EncodeToString(c.Data))
	n, err := w.Write(c.Data)
	result := int64(n) + int64(len(hdr))
	if err != nil {
		return result, driver.WithStack(err)
	}
	return result, nil
}
