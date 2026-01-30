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

package arangodb

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/agency"
	"github.com/arangodb/go-driver/v2/connection"
)

// AgencyCompatible defines an interface compatible with the agency.Agency interface.
// This allows V2's ClientAgency to be used with code that expects the Agency interface,
// such as the election package in starter, without V2 depending on V1.
//
// IMPORTANT MIGRATION NOTE FOR STARTER:
// =====================================
// V1's agency.Agency interface returns driver.Connection (V1 type).
// This AgencyCompatible interface returns connection.Connection (V2 type).
// These are different types and cannot be directly substituted.
//
// To migrate starter's election package from V1 to V2:
//
// Option 1 (Recommended): Update starter's election package to accept AgencyCompatible
//   - Change election package function signatures from agency.Agency to AgencyCompatible
//   - Update imports to use V2's arangodb package
//   - Use: adapter := arangodb.NewAgencyAdapter(clientAgency, v2Client.Connection())
//
// Option 2: Create a connection wrapper in starter
//   - Create a wrapper that implements driver.Connection using V2's connection.Connection
//   - This is more complex but allows gradual migration
//
// Option 3: Use V2's lock package directly (if election only needs locks)
//   - Use: lock, err := arangodb.NewLock(logger, clientAgency, key, id, ttl)
//   - This avoids the adapter entirely if locks are the only requirement
//
// The adapter is ready to use, but starter's election package needs to be updated
// to accept AgencyCompatible instead of agency.Agency for a clean migration.
type AgencyCompatible interface {
	// Connection returns the connection used by this api.
	// Note: Returns V2's connection.Connection, not V1's driver.Connection.
	// This is the main difference from V1's agency.Agency interface.
	Connection() connection.Connection

	// ReadKey reads the value of a given key in the agency.
	ReadKey(ctx context.Context, key []string, value interface{}) error

	// WriteTransaction performs transaction in the agency.
	// Transaction can have a list of operations to perform like e.g. delete, set, observe...
	// Transaction can have preconditions which must be fulfilled to perform transaction.
	// Accepts both TransactionCompatible (for V1 compatibility) and agency.Transaction (V2 direct).
	// When using V2's NewTransactionCompat, pass the transaction directly - it will be handled automatically.
	WriteTransaction(ctx context.Context, transaction interface{}) error

	// WriteKey writes the given value with the given key with a given TTL (unless TTL is zero).
	// If you pass a condition (only 1 allowed), this condition has to be true,
	// otherwise the write will fail with a ConditionFailed error.
	WriteKey(ctx context.Context, key []string, value interface{}, ttl time.Duration, condition ...WriteCondition) error

	// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
	WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error

	// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
	// to the given old value.
	WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error

	// RemoveKey removes the given key.
	// If you pass a condition (only 1 allowed), this condition has to be true,
	// otherwise the remove will fail with a ConditionFailed error.
	RemoveKey(ctx context.Context, key []string, condition ...WriteCondition) error

	// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
	// to the given old value.
	RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error
}

// TransactionCompatible defines an interface compatible with the agency.Transaction.
type TransactionCompatible interface {
	Keys() []agency.KeyChanger
	Conditions() agency.ConditionsMap
	ClientID() *string
	Options() agency.TransactionOptions
}

// WriteCondition is compatible with the agency.WriteCondition.
type WriteCondition struct {
	conditions map[string]writeCondition
}

type writeCondition struct {
	Old      interface{}
	OldEmpty *bool
	IsArray  *bool
}

var condTrue = true

// IfEmpty adds an "is empty" check on the given key to the given condition.
func (c WriteCondition) IfEmpty(key []string) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.OldEmpty = &condTrue
	})
}

// IfIsArray adds an "is-array" check on the given key to the given condition.
func (c WriteCondition) IfIsArray(key []string) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.IsArray = &condTrue
	})
}

// IfEqualTo adds a "value equals oldValue" check to given old value on the given key.
func (c WriteCondition) IfEqualTo(key []string, oldValue interface{}) WriteCondition {
	return c.add(key, func(wc *writeCondition) {
		wc.Old = oldValue
	})
}

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

// convertWriteCondition converts WriteCondition to agency conditions map.
func convertWriteCondition(cond WriteCondition) map[string]agency.KeyConditioner {
	keyConditions := make(map[string]agency.KeyConditioner)

	for key, v := range cond.conditions {
		if v.Old != nil {
			keyConditions[key] = agency.NewConditionIfEqual(v.Old)
		} else if v.OldEmpty != nil {
			keyConditions[key] = agency.NewConditionOldEmpty(*v.OldEmpty)
		} else if v.IsArray != nil {
			keyConditions[key] = agency.NewConditionIsArray(*v.IsArray)
		}
	}

	return keyConditions
}

// AgencyAdapter wraps V2's ClientAgency to implement AgencyCompatible interface.
// This allows V2 ClientAgency to be used with code that expects the Agency interface,
// such as the election package in starter.
type AgencyAdapter struct {
	clientAgency ClientAgency
	conn         connection.Connection
}

// NewAgencyAdapter creates a new adapter that wraps V2's ClientAgency to implement AgencyCompatible interface.
func NewAgencyAdapter(clientAgency ClientAgency, conn connection.Connection) AgencyCompatible {
	return &AgencyAdapter{
		clientAgency: clientAgency,
		conn:         conn,
	}
}

// Connection returns the connection used by this api.
func (a *AgencyAdapter) Connection() connection.Connection {
	return a.conn
}

// ReadKey reads the value of a given key in the agency.
func (a *AgencyAdapter) ReadKey(ctx context.Context, key []string, value interface{}) error {
	return a.clientAgency.ReadKey(ctx, key, value)
}

// WriteTransaction performs transaction in the agency.
// It accepts both TransactionCompatible (for V1 compatibility) and agency.Transaction or *agency.Transaction (V2 direct).
func (a *AgencyAdapter) WriteTransaction(ctx context.Context, transaction interface{}) error {
	var v2Tx agency.Transaction

	// Check if it's a V2 transaction directly (value or pointer)
	if v2TransactionPtr, ok := transaction.(*agency.Transaction); ok {
		v2Tx = *v2TransactionPtr
	} else if v2Transaction, ok := transaction.(agency.Transaction); ok {
		v2Tx = v2Transaction
	} else if compatTx, ok := transaction.(TransactionCompatible); ok {
		// Convert V1-compatible transaction to V2 transaction
		clientID := compatTx.ClientID()
		v2Tx = agency.NewTransaction(clientID, compatTx.Options())

		// Add keys - V1 and V2 KeyChanger interfaces are compatible
		for _, k := range compatTx.Keys() {
			v2Tx.AddKey(k)
		}

		// Add conditions - V1 and V2 KeyConditioner interfaces are compatible
		for key, cond := range compatTx.Conditions() {
			if err := v2Tx.AddConditionByFullKey(key, cond); err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		return errors.New("transaction must be either TransactionCompatible or agency.Transaction")
	}

	return a.clientAgency.WriteTransaction(ctx, v2Tx)
}

// WriteKey writes the given value with the given key with a given TTL.
func (a *AgencyAdapter) WriteKey(ctx context.Context, key []string, value interface{}, ttl time.Duration, condition ...WriteCondition) error {
	clientID := ""
	v2Tx := agency.NewTransaction(&clientID, agency.TransactionOptions{})
	v2Tx.AddKey(agency.NewKeySetWithTTL(key, value, ttl))
	if len(condition) > 0 {
		conditions := convertWriteCondition(condition[0])
		for k, v := range conditions {
			if err := v2Tx.AddConditionByFullKey(k, v); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return a.clientAgency.WriteTransaction(ctx, v2Tx)
}

// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
func (a *AgencyAdapter) WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error {
	return a.clientAgency.WriteKeyIfEmpty(ctx, key, value, ttl)
}

// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value equals the old value.
func (a *AgencyAdapter) WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error {
	return a.clientAgency.WriteKeyIfEqualTo(ctx, key, newValue, oldValue, ttl)
}

// RemoveKey removes the given key.
func (a *AgencyAdapter) RemoveKey(ctx context.Context, key []string, condition ...WriteCondition) error {
	clientID := ""
	v2Tx := agency.NewTransaction(&clientID, agency.TransactionOptions{})
	v2Tx.AddKey(agency.NewKeyDelete(key))
	if len(condition) > 0 {
		conditions := convertWriteCondition(condition[0])
		for k, v := range conditions {
			if err := v2Tx.AddConditionByFullKey(k, v); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return a.clientAgency.WriteTransaction(ctx, v2Tx)
}

// RemoveKeyIfEqualTo removes the given key only if the existing value equals the old value.
func (a *AgencyAdapter) RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error {
	return a.clientAgency.RemoveKeyIfEqualTo(ctx, key, oldValue)
}

// createFullKey creates a full key path from key segments.
func createFullKey(key []string) string {
	return "/" + strings.Join(key, "/")
}

// Helper functions for easier migration from V1 to V2

// NewTransactionCompat creates a new transaction compatible with V1's NewTransaction signature.
// This helper makes it easier to migrate from V1's agency.NewTransaction("", ...) to V2.
// Returns a pointer to the transaction for easier method chaining.
func NewTransactionCompat(clientID string, options agency.TransactionOptions) *agency.Transaction {
	tx := agency.NewTransaction(&clientID, options)
	return &tx
}

// NewKeySetCompat creates a new key set operation compatible with V1's NewKeySet signature.
// This helper makes it easier to migrate from V1's agency.NewKeySet(key, value, ttl) to V2.
// If ttl is 0, it uses NewKeySet, otherwise it uses NewKeySetWithTTL.
func NewKeySetCompat(key []string, value interface{}, ttl time.Duration) agency.KeyChanger {
	if ttl == 0 {
		return agency.NewKeySet(key, value)
	}
	return agency.NewKeySetWithTTL(key, value, ttl)
}
