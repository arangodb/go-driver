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

import (
	"strings"

	"github.com/pkg/errors"
)

// TransactionOptions defines options how transaction should behave.
type TransactionOptions struct {
	Transient bool
}

// Transaction stores information about operations which must be performed for particular keys with some conditions
type Transaction struct {
	keys       []KeyChanger
	conditions ConditionsMap
	clientID   *string
	options    TransactionOptions
}

// NewTransaction creates new transaction.
// The argument 'clientID' should be used to mark that transaction sender uniquely.
func NewTransaction(clientID *string, options TransactionOptions) Transaction {
	if clientID == nil {
		defaultClientID := "go-driver/v2"
		clientID = &defaultClientID
	}

	return Transaction{
		clientID: clientID,
		options:  options,
	}
}

// AddConditionByFullKey adds new condition to the list of keys which must be changed in one transaction
func (k *Transaction) AddConditionByFullKey(fullKey string, condition KeyConditioner) error {
	if k.conditions == nil {
		k.conditions = make(map[string]KeyConditioner)
	}
	if _, ok := k.conditions[fullKey]; ok {
		// For the time being one key can have only one condition. It is a limitation in agency
		return errors.WithStack(errors.New("too many conditions"))
	}

	k.conditions[fullKey] = condition
	return nil
}

// AddCondition adds new condition to the list of keys which must be changed in one transaction
func (k *Transaction) AddCondition(key []string, condition KeyConditioner) error {
	fullKey := createFullKey(key)
	return k.AddConditionByFullKey(fullKey, condition)
}

// AddKey adds new key which must be changed in one transaction
func (k *Transaction) AddKey(key KeyChanger) {
	k.keys = append(k.keys, key)
}

// Keys returns the list of keys that will be changed in this transaction
func (k *Transaction) Keys() []KeyChanger {
	return k.keys
}

// Conditions returns the map of conditions for this transaction
func (k *Transaction) Conditions() ConditionsMap {
	return k.conditions
}

// ClientID returns the client ID for this transaction
func (k *Transaction) ClientID() *string {
	return k.clientID
}

// Options returns the transaction options
func (k *Transaction) Options() TransactionOptions {
	return k.options
}

func createFullKey(key []string) string {
	return "/" + strings.Join(key, "/")
}
