package main

import (
	"encoding/json"
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
		return fmt.Errorf("key 'configuration' not found or not an object")
	}
	pool, ok := configuration["pool"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("key 'pool' not found or not an array")
	}

	leaderId, ok := data["leaderId"].(string)
	if !ok {
		return fmt.Errorf("key 'leaderId' not found or not a str")
	}
	if leaderId == "" {
		return fmt.Errorf("key 'leaderId' not set")
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
