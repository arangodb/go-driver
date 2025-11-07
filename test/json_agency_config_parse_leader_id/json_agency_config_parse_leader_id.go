//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func ExtractValue(input io.Reader) error {
	var data map[string]interface{}

	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	configuration, ok := data["configuration"].(map[string]interface{})
	if !ok {
		return errors.New("key 'configuration' not found or not an object")
	}
	pool, ok := configuration["pool"].(map[string]interface{})
	if !ok {
		return errors.New("key 'pool' not found or not an array")
	}

	leaderId, ok := data["leaderId"].(string)
	if !ok {
		return errors.New("key 'leaderId' not found or not a str")
	}
	if leaderId == "" {
		return errors.New("key 'leaderId' not set")
	}

	endpoint, ok := pool[leaderId].(string)
	if !ok {
		return fmt.Errorf("key '%s' not found or not a str", leaderId)
	}
	fmt.Println(endpoint)

	return nil
}

func main() {
	if err := ExtractValue(os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "JsonAgencyConfigParseError: %v\n and as a result the agency dump could not be created.\n", err)

	}
}
