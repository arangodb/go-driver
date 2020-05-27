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
	"encoding/json"
	"fmt"
	"github.com/arangodb/go-driver"
	"strings"
	"time"
)

var (
	condTrue  = true  // Do not change!
	condFalse = false // Do not change!
)

type agency struct {
	conn driver.Connection
}

// NewAgency creates an Agency accessor for the given connection.
// The connection must contain the endpoints of one or more agents, and only agents.
func NewAgency(conn driver.Connection) (Agency, error) {
	return &agency{
		conn: conn,
	}, nil
}

// Connection returns the connection used by this api.
func (c *agency) Connection() driver.Connection {
	return c.conn
}

// ReadKey reads the value of a given key in the agency.
func (c *agency) ReadKey(ctx context.Context, key []string, value interface{}) error {
	conn := c.conn
	req, err := conn.NewRequest("POST", "_api/agency/read")
	if err != nil {
		return driver.WithStack(err)
	}
	fullKey := createFullKey(key)
	input := [][]string{{fullKey}}
	req, err = req.SetBody(input)
	if err != nil {
		return driver.WithStack(err)
	}
	//var raw []byte
	//ctx = driver.WithRawResponse(ctx, &raw)
	resp, err := conn.Do(ctx, req)
	if err != nil {
		return driver.WithStack(err)
	}
	if err := resp.CheckStatus(200, 201, 202); err != nil {
		return driver.WithStack(err)
	}
	//fmt.Printf("Agent response: %s\n", string(raw))
	elems, err := resp.ParseArrayBody()
	if err != nil {
		return driver.WithStack(err)
	}
	if len(elems) != 1 {
		return driver.WithStack(fmt.Errorf("Expected 1 element, got %d", len(elems)))
	}
	// If empty key parse directly
	if len(key) == 0 {
		if err := elems[0].ParseBody("", &value); err != nil {
			return driver.WithStack(err)
		}
	} else {
		// Now remove all wrapping objects for each key element
		var rawObject map[string]interface{}
		if err := elems[0].ParseBody("", &rawObject); err != nil {
			return driver.WithStack(err)
		}
		var rawMsg interface{}
		for keyIndex := 0; keyIndex < len(key); keyIndex++ {
			if keyIndex > 0 {
				var ok bool
				rawObject, ok = rawMsg.(map[string]interface{})
				if !ok {
					return driver.WithStack(fmt.Errorf("Data is not an object at key %s", key[:keyIndex+1]))
				}
			}
			var found bool
			rawMsg, found = rawObject[key[keyIndex]]
			if !found {
				return driver.WithStack(KeyNotFoundError{Key: key[:keyIndex+1]})
			}
		}
		// Encode to json ...
		encoded, err := json.Marshal(rawMsg)
		if err != nil {
			return driver.WithStack(err)
		}
		// and decode back into result
		if err := json.Unmarshal(encoded, &value); err != nil {
			return driver.WithStack(err)
		}
	}

	//	fmt.Printf("result as JSON: %s\n", rawResult)
	return nil
}

type writeUpdate struct {
	Operation string      `json:"op,omitempty"`
	New       interface{} `json:"new,omitempty"`
	TTL       int64       `json:"ttl,omitempty"`
	URL       string      `json:"url,omitempty"`
}

type writeCondition struct {
	Old      interface{} `json:"old,omitempty"`      // Require old value to be equal to this
	OldEmpty *bool       `json:"oldEmpty,omitempty"` // Require old value to be empty
	IsArray  *bool       `json:"isArray,omitempty"`  // Require old value to be array
}

type writeTransaction []interface{}

type writeResult struct {
	Results []int64 `json:"results"`
}

// Deprecated: use 'WriteTransaction' instead
// WriteKey writes the given value with the given key with a given TTL (unless TTL is zero).
// If you pass a condition (only 1 allowed), this condition has to be true,
// otherwise the write will fail with a ConditionFailed error.
func (c *agency) WriteKey(ctx context.Context, key []string, value interface{}, ttl time.Duration, condition ...WriteCondition) error {
	var cond WriteCondition
	switch len(condition) {
	case 0:
	// No condition, do nothing
	case 1:
		cond = condition[0]
	default:
		return driver.WithStack(fmt.Errorf("too many conditions"))
	}

	transaction := NewTransaction("")
	transaction.AddKey(NewKeySet(key, value, ttl))
	conditions := ConvertWriteCondition(cond)
	transaction.SetConditions(conditions)

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// Deprecated: use 'WriteTransaction' instead
// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
func (c *agency) WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error {
	transaction := NewTransaction("")
	transaction.AddKey(NewKeySet(key, value, ttl))
	transaction.AddCondition(key, NewConditionOldEmpty(true))

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// Deprecated: use 'WriteTransaction' instead
// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
// to the given old value.
func (c *agency) WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error {
	transaction := NewTransaction("")
	transaction.AddKey(NewKeySet(key, newValue, ttl))
	transaction.AddCondition(key, NewConditionIfEqual(oldValue))

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// WriteTransaction performs transaction in the agency.
// Transaction can have list of operations to perform like e.g. delete, set, observe...
// Transaction can have preconditions which must be fulfilled to perform transaction.
func (c *agency) WriteTransaction(ctx context.Context, transaction Transaction, transient ...bool) error {
	conn := c.conn

	var path string
	if len(transient) > 0 && transient[0] == true {
		path = "_api/agency/transient"
	} else {
		path = "_api/agency/write"
	}

	req, err := conn.NewRequest("POST", path)
	if err != nil {
		return driver.WithStack(err)
	}

	writeTxs := make([]writeTransaction, 0, 1)
	f := make(writeTransaction, 0, 3)
	keysToChange := make(map[string]interface{})
	for _, v := range transaction.keys {
		keysToChange[v.GetKey()] = writeUpdate{
			Operation: v.GetOperation(),
			New:       v.GetValue(),
			TTL:       int64(v.GetTTL().Seconds()),
			URL:       v.GetURL(),
		}
	}

	conditions := make(map[string]interface{})
	if transaction.conditions != nil {
		for key, condition := range transaction.conditions {
			conditions[key] = map[string]interface{}{
				condition.GetName(): condition.GetValue(),
			}
		}
	}

	f = append(f, keysToChange, conditions, transaction.clientID)
	writeTxs = append(writeTxs, f)

	req, err = req.SetBody(writeTxs)
	if err != nil {
		return driver.WithStack(err)
	}
	resp, err := conn.Do(ctx, req)
	if err != nil {
		return driver.WithStack(err)
	}

	var result writeResult
	if err := resp.CheckStatus(200, 201, 202); err != nil {
		return driver.WithStack(err)
	}

	if err := resp.ParseBody("", &result); err != nil {
		return driver.WithStack(err)
	}

	if len(result.Results) != 1 {
		return driver.WithStack(fmt.Errorf("expected results of 1 long, got %d", len(result.Results)))
	}

	if result.Results[0] == 0 {
		// Condition failed
		return driver.WithStack(preconditionFailedError)
	}

	// Success
	return nil
}

// Deprecated: use 'WriteTransaction' instead
// RemoveKey removes the given key.
// If you pass a condition (only 1 allowed), this condition has to be true,
// otherwise the remove will fail with a ConditionFailed error.
func (c *agency) RemoveKey(ctx context.Context, key []string, condition ...WriteCondition) error {
	var cond WriteCondition
	switch len(condition) {
	case 0:
	// No condition, do nothing
	case 1:
		cond = condition[0]
	default:
		return driver.WithStack(fmt.Errorf("too many conditions"))
	}

	transaction := NewTransaction("")
	transaction.AddKey(NewKeyDelete(key))
	conditions := ConvertWriteCondition(cond)
	transaction.SetConditions(conditions)

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// Deprecated: use 'WriteTransaction' instead
// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
// to the given old value.
func (c *agency) RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error {
	transaction := NewTransaction("")
	transaction.AddKey(NewKeyDelete(key))
	transaction.AddCondition(key, NewConditionIfEqual(oldValue))

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// Deprecated: use 'WriteTransaction' instead
// Register a URL to receive notification callbacks when the value of the given key changes
func (c *agency) RegisterChangeCallback(ctx context.Context, key []string, cbURL string) error {
	transaction := NewTransaction("")
	transaction.AddKey(NewKeyObserve(key, cbURL, true))

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	return nil
}

// Deprecated: use 'WriteTransaction' instead
// Register a URL to receive notification callbacks when the value of the given key changes
func (c *agency) UnregisterChangeCallback(ctx context.Context, key []string, cbURL string) error {

	transaction := NewTransaction("")
	transaction.AddKey(NewKeyObserve(key, cbURL, false))

	if err := c.WriteTransaction(ctx, transaction); err != nil {
		return driver.WithStack(err)
	}

	// Success
	return nil
}

func createFullKey(key []string) string {
	return "/" + strings.Join(key, "/")
}
