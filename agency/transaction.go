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

import (
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/dchest/uniuri"
)

// Transaction stores information about operations which must be performed for particular keys with some conditions
type Transaction struct {
	keys       []KeyChanger
	conditions ConditionsMap
	clientID   string
}

// NewTransaction creates new transaction
func NewTransaction(clientID string) Transaction {
	if len(clientID) == 0 {
		clientID = uniuri.New()
	}

	return Transaction{
		clientID: clientID,
	}
}

// SetConditions sets new conditions for the transaction
func (k *Transaction) SetConditions(conditions ConditionsMap) {
	k.conditions = conditions
}

// AddCondition adds new condition to the list of keys which must be changed in one transaction
func (k *Transaction) AddCondition(key []string, condition KeyConditioner) error {
	if k.conditions == nil {
		k.conditions = make(map[string]KeyConditioner)
	}

	fullKey := createFullKey(key)
	if _, ok := k.conditions[fullKey]; ok {
		// For the time being one key can have only one condition. It is a limitation in agency
		return driver.WithStack(fmt.Errorf("too many conditions"))
	}

	k.conditions[fullKey] = condition
	return nil
}

// AddKey adds new key which must be changed in one transaction
func (k *Transaction) AddKey(key KeyChanger) {
	k.keys = append(k.keys, key)
}
