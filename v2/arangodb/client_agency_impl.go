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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/agency"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/arangodb/go-driver/v2/log"
)

type clientAgency struct {
	client *client
}

func newClientAgency(client *client) *clientAgency {
	return &clientAgency{
		client: client,
	}
}

var _ ClientAgency = &clientAgency{}

// KeyNotFoundError indicates that a key was not found.
type KeyNotFoundError struct {
	Key []string
}

// Error returns a human readable error string
func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("Key '%s' not found", strings.Join(e.Key, "/"))
}

// IsKeyNotFound returns true if the given error is (or is caused by) a KeyNotFoundError.
func IsKeyNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(KeyNotFoundError)
	if ok {
		return true
	}
	// Check if wrapped
	var keyErr KeyNotFoundError
	return errors.As(err, &keyErr)
}

// ReadKey reads the value of a given key in the agency.
// If a 307 Temporary Redirect is received, it extracts the leader endpoint from the Location header
// and updates the connection to use the leader endpoint, then retries the request.
func (c *clientAgency) ReadKey(ctx context.Context, key []string, value interface{}) error {
	fullKey := createFullKey(key)
	input := [][]string{{fullKey}}

	url := connection.NewUrl("_api", "agency", "read")

	var response []interface{}
	allowedStatusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusTemporaryRedirect, // Allow 307 for redirect handling
	}

	resp, err := connection.CallWithChecks(
		ctx,
		c.client.connection,
		http.MethodPost,
		url,
		&response,
		allowedStatusCodes,
		connection.WithBody(input),
	)
	if err != nil {
		log.Errorf(err, "Agency ReadKey failed: key=%s", fullKey)
		return errors.WithStack(err)
	}

	switch resp.Code() {
	case http.StatusTemporaryRedirect:
		// Handle agency redirect - extract leader endpoint from Location header
		location := resp.Header("Location")
		if location != "" {
			log.Debugf("Agency redirect detected: key=%s, location=%s", fullKey, location)
			// Update connection to use leader endpoint
			leaderEndpoint := connection.FixupEndpointURLScheme(location)
			newEndpoint := connection.NewRoundRobinEndpoints([]string{leaderEndpoint})
			if err := c.client.connection.SetEndpoint(newEndpoint); err != nil {
				log.Errorf(err, "Failed to update endpoint after redirect: location=%s", location)
				return errors.WithStack(err)
			}
			// Retry the request with the leader endpoint
			return c.ReadKey(ctx, key, value)
		}
		// No Location header, return error
		log.Errorf(nil, "Agency ReadKey redirect without Location header: key=%s", fullKey)
		return (&shared.ResponseStruct{}).AsArangoErrorWithCode(http.StatusTemporaryRedirect)

	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		if len(response) != 1 {
			return errors.WithStack(
				fmt.Errorf("Agency ReadKey: expected 1 element, got %d", len(response)),
			)
		}
		// Start from the root object
		current := response[0]

		// Traverse key path if provided
		for i, k := range key {
			obj, ok := current.(map[string]interface{})
			if !ok {
				return errors.WithStack(
					fmt.Errorf("data is not an object at key %s", strings.Join(key[:i], "/")),
				)
			}
			var found bool
			current, found = obj[k]
			if !found {
				return errors.WithStack(KeyNotFoundError{Key: key[:i+1]})
			}
		}

		// Convert extracted value into typed result
		data, err := json.Marshal(current)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := json.Unmarshal(data, value); err != nil {
			return errors.WithStack(err)
		}
		return nil

	default:
		log.Errorf(nil, "Agency ReadKey failed: key=%s, status=%d", fullKey, resp.Code())
		return (&shared.ResponseStruct{}).AsArangoErrorWithCode(resp.Code())
	}
}

type writeUpdate struct {
	Operation string      `json:"op,omitempty"`
	New       interface{} `json:"new,omitempty"`
	URL       string      `json:"url,omitempty"`
	Val       interface{} `json:"val,omitempty"`
}

type writeTransaction []interface{}

type writeResult struct {
	shared.ResponseStruct `json:",inline"`
	Results               []int64 `json:"results"`
}

// WriteTransaction performs transaction in the agency.
// Transaction can have list of operations to perform like e.g. delete, set, observe...
// Transaction can have preconditions which must be fulfilled to perform transaction.
func (c *clientAgency) WriteTransaction(ctx context.Context, transaction agency.Transaction) error {
	var url string
	options := transaction.Options()
	if options.Transient {
		url = connection.NewUrl("_api", "agency", "transient")
	} else {
		url = connection.NewUrl("_api", "agency", "write")
	}

	writeTxs := make([]writeTransaction, 0, 1)
	f := make(writeTransaction, 0, 3)

	keysToChange := make(map[string]interface{})
	keys := transaction.Keys()
	for _, v := range keys {
		opStr := v.GetOperation()
		urlStr := v.GetURL()
		key := v.GetKey()
		update := writeUpdate{
			Operation: opStr,
			New:       v.GetNew(),
			URL:       urlStr,
			Val:       v.GetVal(),
		}
		keysToChange[key] = update
	}

	conditions := make(map[string]interface{})
	transactionConditions := transaction.Conditions()

	for key, condition := range transactionConditions {
		conditions[key] = map[string]interface{}{
			condition.GetName(): condition.GetValue(),
		}
	}

	if len(conditions) > 0 {
		log.Debugf("Agency WriteTransaction: num_conditions=%d", len(conditions))
	} else {
		log.Debugf("Agency WriteTransaction: no conditions found")
	}

	// operations (keys to change) must be first parameter
	// conditions must be second parameter
	f = append(f, keysToChange, conditions)

	// clientID must be third parameter
	clientID := transaction.ClientID()
	if len(*clientID) > 0 {
		f = append(f, *clientID)
	}
	writeTxs = append(writeTxs, f)

	var response writeResult
	allowed := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusPreconditionFailed,
	}

	_, err := connection.CallWithChecks(
		ctx,
		c.client.connection,
		http.MethodPost,
		url,
		&response,
		allowed,
		connection.WithBody(writeTxs),
	)
	if err != nil {
		// Don't log context cancellation errors - they're expected during cleanup
		if !shared.IsCanceled(err) {
			log.Errorf(err, "Agency WriteTransaction failed: url=%s", url)
		}
		return errors.WithStack(err)
	}

	// Results FIRST (authoritative)
	if len(response.Results) == 0 {
		// Valid condition failure case - empty Results array
		return (&shared.ResponseStruct{}).
			AsArangoErrorWithCode(http.StatusPreconditionFailed)
	}

	if len(response.Results) != 1 {
		return errors.Errorf("expected 1 result, got %d", len(response.Results))
	}

	if response.Results[0] == 0 {
		// Condition failed - Results[0] = 0 indicates condition failure
		return (&shared.ResponseStruct{}).
			AsArangoErrorWithCode(http.StatusPreconditionFailed)
	}

	// Success - Results[0] != 0 means transaction succeeded
	return nil
}

// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
// This is a convenience method for lock functionality.
func (c *clientAgency) WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error {
	transaction := agency.NewTransaction(nil, agency.TransactionOptions{})
	transaction.AddKey(agency.NewKeySetWithTTL(key, value, ttl))
	transaction.AddCondition(key, agency.NewConditionOldEmpty(true))
	return c.WriteTransaction(ctx, transaction)
}

// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
// to the given old value.
// This is a convenience method for lock functionality.
func (c *clientAgency) WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error {
	transaction := agency.NewTransaction(nil, agency.TransactionOptions{})
	transaction.AddKey(agency.NewKeySetWithTTL(key, newValue, ttl))
	transaction.AddCondition(key, agency.NewConditionIfEqual(oldValue))
	return c.WriteTransaction(ctx, transaction)
}

// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
// to the given old value.
// This is a convenience method for lock functionality.
func (c *clientAgency) RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error {
	transaction := agency.NewTransaction(nil, agency.TransactionOptions{})
	transaction.AddKey(agency.NewKeyDelete(key))
	transaction.AddCondition(key, agency.NewConditionIfEqual(oldValue))
	return c.WriteTransaction(ctx, transaction)
}
