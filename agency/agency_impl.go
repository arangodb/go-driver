//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

package agency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/arangodb/go-driver"
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

type writeTransaction []map[string]interface{}
type writeTransactions []writeTransaction

type writeResult struct {
	Results []int64 `json:"results"`
}

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
	if err := c.write(ctx, "set", key, value, cond, ttl); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
func (c *agency) WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error {
	var cond WriteCondition
	cond = cond.IfEmpty(key)
	if err := c.write(ctx, "set", key, value, cond, ttl); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
// to the given old value.
func (c *agency) WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error {
	var cond WriteCondition
	cond = cond.IfEqualTo(key, oldValue)
	if err := c.write(ctx, "set", key, newValue, cond, ttl); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// write writes the given value with the given key only if the given condition is fullfilled.
func (c *agency) write(ctx context.Context, operation string, key []string, value interface{}, condition WriteCondition, ttl time.Duration) error {
	conn := c.conn
	req, err := conn.NewRequest("POST", "_api/agency/write")
	if err != nil {
		return driver.WithStack(err)
	}

	fullKey := createFullKey(key)
	writeTxs := writeTransactions{
		writeTransaction{
			// Update
			map[string]interface{}{
				fullKey: writeUpdate{
					Operation: operation,
					New:       value,
					TTL:       int64(ttl.Seconds()),
				},
			},
			// Condition
			condition.toMap(),
		},
	}
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

	// "results" should be 1 long
	if len(result.Results) != 1 {
		return driver.WithStack(fmt.Errorf("Expected results of 1 long, got %d", len(result.Results)))
	}

	// If results[0] == 0, condition failed, otherwise success
	if result.Results[0] == 0 {
		// Condition failed
		return driver.WithStack(preconditionFailedError)
	}

	// Success
	return nil
}

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
	if err := c.write(ctx, "delete", key, nil, cond, 0); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
// to the given old value.
func (c *agency) RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error {
	var cond WriteCondition
	cond = cond.IfEqualTo(key, oldValue)
	if err := c.write(ctx, "delete", key, nil, cond, 0); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Register a URL to receive notification callbacks when the value of the given key changes
func (c *agency) RegisterChangeCallback(ctx context.Context, key []string, cbURL string) error {
	conn := c.conn
	req, err := conn.NewRequest("POST", "_api/agency/write")
	if err != nil {
		return driver.WithStack(err)
	}

	fullKey := createFullKey(key)
	writeTxs := writeTransactions{
		writeTransaction{
			// Update
			map[string]interface{}{
				fullKey: writeUpdate{
					Operation: "observe",
					URL:       cbURL,
				},
			},
		},
	}

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

	// "results" should be 1 long
	if len(result.Results) != 1 {
		return driver.WithStack(fmt.Errorf("Expected results of 1 long, got %d", len(result.Results)))
	}

	// Success
	return nil
}

// Register a URL to receive notification callbacks when the value of the given key changes
func (c *agency) UnregisterChangeCallback(ctx context.Context, key []string, cbURL string) error {
	conn := c.conn
	req, err := conn.NewRequest("POST", "_api/agency/write")
	if err != nil {
		return driver.WithStack(err)
	}

	fullKey := createFullKey(key)
	writeTxs := writeTransactions{
		writeTransaction{
			// Update
			map[string]interface{}{
				fullKey: writeUpdate{
					Operation: "unobserve",
					URL:       cbURL,
				},
			},
		},
	}

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

	// "results" should be 1 long
	if len(result.Results) != 1 {
		return driver.WithStack(fmt.Errorf("Expected results of 1 long, got %d", len(result.Results)))
	}

	// Success
	return nil
}

func createFullKey(key []string) string {
	return "/" + strings.Join(key, "/")
}
