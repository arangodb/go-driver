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
	"sort"
	"sync/atomic"
)

// Message is what is send back to the client in response to a request.
type Message struct {
	ID   uint64
	Data []byte

	chunks             []chunk
	numberOfChunks     uint32
	responseChanClosed int32
	responseChan       chan Message
}

// closes the response channel if needed.
func (m *Message) closeResponseChan() {
	if atomic.CompareAndSwapInt32(&m.responseChanClosed, 0, 1) {
		if ch := m.responseChan; ch != nil {
			m.responseChan = nil
			close(ch)
		}
	}
}

// notifyListener pushes itself onto its response channel and closes the response channel afterwards.
func (m *Message) notifyListener() {
	if atomic.CompareAndSwapInt32(&m.responseChanClosed, 0, 1) {
		if ch := m.responseChan; ch != nil {
			m.responseChan = nil
			ch <- *m
			close(ch)
		}
	}
}

// addChunk adds the given chunks to the list of chunks of the message.
// If the given chunk is the first chunk, the expected number of chunks is recorded.
func (m *Message) addChunk(c chunk) {
	m.chunks = append(m.chunks, c)
	if c.IsFirst() {
		m.numberOfChunks = c.NumberOfChunks()
	}
}

// assemble tries to assemble the message data from all chunks.
// If not all chunks are available yet, nothing is done and false
// is returned.
// If all chunks are available, the Data field is build and set and true is returned.
func (m *Message) assemble() bool {
	if m.Data != nil {
		// Already assembled
		return true
	}
	if m.numberOfChunks == 0 {
		// We don't have the first chunk yet
		return false
	}
	if len(m.chunks) < int(m.numberOfChunks) {
		// Not all chunks have arrived yet
		return false
	}

	// Fast path, only 1 chunk
	if m.numberOfChunks == 1 {
		m.Data = m.chunks[0].Data
		return true
	}

	// Sort chunks by index
	sort.Sort(chunkByIndex(m.chunks))

	// Build data buffer and copy chunks into it
	data := make([]byte, m.chunks[0].MessageLength)
	offset := 0
	for _, c := range m.chunks {
		copy(data[offset:], c.Data)
		offset += len(c.Data)
	}
	m.Data = data
	return true
}

type chunkByIndex []chunk

// Len is the number of elements in the collection.
func (l chunkByIndex) Len() int { return len(l) }

// Less reports whether the element with
// index i should sort before the element with index j.
func (l chunkByIndex) Less(i, j int) bool {
	ii := l[i].Index()
	ij := l[j].Index()
	return ii < ij
}

// Swap swaps the elements with indexes i and j.
func (l chunkByIndex) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
