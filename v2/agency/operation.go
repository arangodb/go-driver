//
// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

package agency

import "time"

type Key []string

// KeyChanger describes how operation should be performed on a key in the agency
type KeyChanger interface {
	// GetKey returns which key must be changed
	GetKey() string
	// GetOperation returns what type of operation must be performed on a key
	GetOperation() string
	// GetURL returns URL address where must be sent callback in case of some changes on key
	GetURL() string
	// GetNew returns new value for a key in the agency
	GetNew() interface{}
	// GetVal returns new value for a key in the agency
	GetVal() interface{}

	// Deprecated: removed since 3.12
	//
	// GetTTL returns how long (in seconds) a key will live in the agency
	GetTTL() time.Duration
}

type keyCommon struct {
	key Key
}

// CreateSubKey creates new key based on receiver key.
// Returns new key with new allocated memory.
func (k Key) CreateSubKey(elements ...string) Key {
	NewKey := make([]string, 0, len(k)+len(elements))

	NewKey = append(NewKey, k...)
	NewKey = append(NewKey, elements...)

	return NewKey
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

type keyArrayPush struct {
	keyCommon
	value interface{}
}

type keyArrayErase struct {
	keyCommon
	value interface{}
}

type keyArrayReplace struct {
	keyCommon
	newValue interface{}
	oldValue interface{}
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
func NewKeySet(key []string, value interface{}) KeyChanger {
	return &keySet{
		keyCommon: keyCommon{
			key: key,
		},
		value: value,
	}
}

// NewKeySetWithTTL returns a new key operation which must be set in the agency with TTL.
// TTL is deprecated since ArangoDB 3.12 but still needed for lock functionality.
func NewKeySetWithTTL(key []string, value interface{}, ttl time.Duration) KeyChanger {
	return &keySet{
		keyCommon: keyCommon{
			key: key,
		},
		TTL:   ttl,
		value: value,
	}
}

// NewKeyArrayPush returns a new key operation for adding elements to the array.
func NewKeyArrayPush(key []string, value interface{}) KeyChanger {
	return &keyArrayPush{
		keyCommon: keyCommon{
			key: key,
		},
		value: value,
	}
}

// NewKeyArrayErase returns a new key operation for removing elements from the array.
func NewKeyArrayErase(key []string, value interface{}) KeyChanger {
	return &keyArrayErase{
		keyCommon: keyCommon{
			key: key,
		},
		value: value,
	}
}

// NewKeyArrayReplace returns a new key operation for replacing element in the array.
func NewKeyArrayReplace(key []string, oldValue, newValue interface{}) KeyChanger {
	return &keyArrayReplace{
		keyCommon: keyCommon{
			key: key,
		},
		newValue: newValue,
		oldValue: oldValue,
	}
}

func (k *keyDelete) GetOperation() string {
	return "delete"
}

func (k *keyDelete) GetTTL() time.Duration {
	return 0
}

func (k *keyDelete) GetNew() interface{} {
	return nil
}

func (k *keyDelete) GetURL() string {
	return ""
}

func (k *keyDelete) GetVal() interface{} {
	return nil
}

func (k *keySet) GetOperation() string {
	return "set"
}

func (k *keySet) GetTTL() time.Duration {
	return k.TTL
}

func (k *keySet) GetNew() interface{} {
	return k.value
}

func (k *keySet) GetURL() string {
	return ""
}

func (k *keySet) GetVal() interface{} {
	return nil
}

func (k *keyArrayPush) GetOperation() string {
	return "push"
}

func (k *keyArrayPush) GetTTL() time.Duration {
	return 0
}

func (k *keyArrayPush) GetNew() interface{} {
	return k.value
}

func (k *keyArrayPush) GetURL() string {
	return ""
}

func (k *keyArrayPush) GetVal() interface{} {
	return nil
}

func (k *keyArrayErase) GetOperation() string {
	return "erase"
}

func (k *keyArrayErase) GetTTL() time.Duration {
	return 0
}

func (k *keyArrayErase) GetNew() interface{} {
	return nil
}

func (k *keyArrayErase) GetURL() string {
	return ""
}

func (k *keyArrayErase) GetVal() interface{} {
	return k.value
}

func (k *keyArrayReplace) GetOperation() string {
	return "replace"
}

func (k *keyArrayReplace) GetTTL() time.Duration {
	return 0
}

func (k *keyArrayReplace) GetNew() interface{} {
	return k.newValue
}

func (k *keyArrayReplace) GetURL() string {
	return ""
}

func (k *keyArrayReplace) GetVal() interface{} {
	return k.oldValue
}
