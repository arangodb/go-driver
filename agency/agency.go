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
// Author Ewout Prangsma
// Author Tomasz Mielech <tomasz@arangodb.com>
//

package agency

import (
	"context"
	"time"

	driver "github.com/arangodb/go-driver"
)

// Agency provides API implemented by the ArangoDB agency.
type Agency interface {
	// Connection returns the connection used by this api.
	Connection() driver.Connection

	// ReadKey reads the value of a given key in the agency.
	ReadKey(ctx context.Context, key []string, value interface{}) error

	// WriteTransaction performs transaction in the agency.
	// Transaction can have list of operations to perform like e.g. delete, set, observe...
	// Transaction can have preconditions which must be fulfilled to perform transaction.
	WriteTransaction(ctx context.Context, transactions Transaction, transient ...bool) error

	// WriteKey writes the given value with the given key with a given TTL (unless TTL is zero).
	// If you pass a condition (only 1 allowed), this condition has to be true,
	// otherwise the write will fail with a ConditionFailed error.
	WriteKey(ctx context.Context, key []string, value interface{}, ttl time.Duration, condition ...WriteCondition) error

	// Deprecated: use 'WriteTransaction' instead
	// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
	WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error

	// Deprecated: use 'WriteTransaction' instead
	// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
	// to the given old value.
	WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error

	// RemoveKey removes the given key.
	// If you pass a condition (only 1 allowed), this condition has to be true,
	// otherwise the remove will fail with a ConditionFailed error.
	RemoveKey(ctx context.Context, key []string, condition ...WriteCondition) error

	// Deprecated: use 'WriteTransaction' instead
	// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
	// to the given old value.
	RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error

	// Deprecated: use 'WriteTransaction' instead
	// Register a URL to receive notification callbacks when the value of the given key changes
	RegisterChangeCallback(ctx context.Context, key []string, cbURL string) error

	// Deprecated: use 'WriteTransaction' instead
	// Register a URL to receive notification callbacks when the value of the given key changes
	UnregisterChangeCallback(ctx context.Context, key []string, cbURL string) error
}

// Deprecated: use 'agency.KeyConditioner' instead
// WriteCondition is a precondition before a write is accepted.
type WriteCondition struct {
	conditions map[string]writeCondition
}

// IfEmpty adds an empty check on the given key to the given condition
// and returns the updated condition.
func (c WriteCondition) add(key []string, updater func(wc *writeCondition)) WriteCondition {
	if c.conditions == nil {
		c.conditions = make(map[string]writeCondition)
	}
	fullKey := createFullKey(key)
	wc := c.conditions[fullKey]
	updater(&wc)
	c.conditions[fullKey] = wc
	return c
}

// Deprecated: use 'agency.KeyConditioner' instead
// IfEmpty adds an "is empty" check on the given key to the given condition
// and returns the updated condition.
func (c WriteCondition) IfEmpty(key []string) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.OldEmpty = &condTrue
	})
}

// Deprecated: use 'agency.KeyConditioner' instead
// IfIsArray adds an "is-array" check on the given key to the given condition
// and returns the updated condition.
func (c WriteCondition) IfIsArray(key []string) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.IsArray = &condTrue
	})
}

// Deprecated: use 'agency.KeyConditioner' instead
// IfEqualTo adds an "value equals oldValue" check to given old value on the
// given key to the given condition and returns the updated condition.
func (c WriteCondition) IfEqualTo(key []string, oldValue interface{}) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.Old = oldValue
	})
}
