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
// Tomasz Mielech <tomasz@arangodb.com>
//

package agency

import "time"

// KeyChanger describes how operation should be performed on a key in the agency
type KeyChanger interface {
	// GetKey returns which key must be changed
	GetKey() string
	// GetOperation returns what type of operation must be performed on a key
	GetOperation() string
	// GetTTL returns how long (in seconds) a key will live in the agency
	GetTTL() time.Duration
	// GetURL returns URL address where must be sent callback in case of some changes on key
	GetURL() string
	// GetValue returns new value for a key in the agency
	GetValue() interface{}
}

type keyCommon struct {
	key []string
}

func (k *keyCommon) GetKey() string {
	return createFullKey(k.key)
}

type keyDelete struct {
	keyCommon
}

type keySet struct {
	keyCommon
	TTL   time.Duration
	value interface{}
}

type keyObserve struct {
	keyCommon
	URL     string
	observe bool
}

// NewKeyDelete returns a new key operation which must be removed from the agency
func NewKeyDelete(key []string) KeyChanger {
	return &keyDelete{
		keyCommon{
			key: key,
		},
	}
}

// NewKeySet returns a new key operation which must be set in the agency
func NewKeySet(key []string, value interface{}, TTL time.Duration) KeyChanger {
	return &keySet{
		keyCommon: keyCommon{
			key: key,
		},
		TTL:   TTL,
		value: value,
	}
}

// NewKeyObserve returns a new key callback operation which must be written in the agency.
// URL parameter describes where callback must be sent in case of changes on a key.
// When 'observe' is false then we want to stop observing a key.
func NewKeyObserve(key []string, URL string, observe bool) KeyChanger {
	return &keyObserve{
		keyCommon: keyCommon{
			key: key,
		},
		URL:     URL,
		observe: observe,
	}
}

func (k *keyDelete) GetOperation() string {
	return "delete"
}

func (k *keyDelete) GetTTL() time.Duration {
	return 0
}

func (k *keyDelete) GetValue() interface{} {
	return nil
}

func (k *keyDelete) GetURL() string {
	return ""
}

func (k *keySet) GetOperation() string {
	return "set"
}

func (k *keySet) GetTTL() time.Duration {
	return k.TTL
}

func (k *keySet) GetValue() interface{} {
	return k.value
}

func (k *keySet) GetURL() string {
	return ""
}

func (k *keyObserve) GetOperation() string {
	if k.observe {
		return "observe"
	}
	return "unobserve"
}

func (k *keyObserve) GetTTL() time.Duration {
	return 0
}

func (k *keyObserve) GetValue() interface{} {
	return nil
}

func (k *keyObserve) GetURL() string {
	return k.URL
}
