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
	"sync"
)

type messageStore struct {
	mutex    sync.RWMutex
	messages map[uint64]*Message
}

// Get returns the message with given id, or nil if not found
func (s *messageStore) Get(id uint64) *Message {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	m, ok := s.messages[id]
	if ok {
		return m
	}
	return nil
}

// Add adds a new message to the store with given ID.
// If the ID is not unique this function will panic.
func (s *messageStore) Add(id uint64) *Message {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.messages == nil {
		s.messages = make(map[uint64]*Message)
	}
	if _, ok := s.messages[id]; ok {
		panic(fmt.Sprintf("ID %v is not unique", id))
	}

	m := &Message{
		ID:       id,
		response: make(chan Message),
	}
	s.messages[id] = m
	return m
}

// Remove removes the message with given ID from the store.
func (s *messageStore) Remove(id uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.messages, id)
}

// ForEach calls the given function for each message in the store.
func (s *messageStore) ForEach(cb func(*Message)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, m := range s.messages {
		cb(m)
	}
}
