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
	"time"

	"github.com/arangodb/go-driver/v2/agency"
)

// ClientAgency provides API implemented by the ArangoDB agency.
type ClientAgency interface {
	// ReadKey reads the value of a given key in the agency.
	ReadKey(ctx context.Context, key []string, value interface{}) error

	// WriteTransaction performs transaction in the agency.
	// Transaction can have a list of operations to perform like e.g. delete, set, observe...
	// Transaction can have preconditions which must be fulfilled to perform transaction.
	WriteTransaction(ctx context.Context, transaction agency.Transaction) error

	// WriteKeyIfEmpty writes the given value with the given key only if the key was empty before.
	// This is a convenience method for lock functionality.
	// Deprecated: Use WriteTransaction with OldEmpty condition instead.
	WriteKeyIfEmpty(ctx context.Context, key []string, value interface{}, ttl time.Duration) error

	// WriteKeyIfEqualTo writes the given new value with the given key only if the existing value for that key equals
	// to the given old value.
	// This is a convenience method for lock functionality.
	// Deprecated: Use WriteTransaction with IfEqual condition instead.
	WriteKeyIfEqualTo(ctx context.Context, key []string, newValue, oldValue interface{}, ttl time.Duration) error

	// RemoveKeyIfEqualTo removes the given key only if the existing value for that key equals
	// to the given old value.
	// This is a convenience method for lock functionality.
	// Deprecated: Use WriteTransaction with IfEqual condition instead.
	RemoveKeyIfEqualTo(ctx context.Context, key []string, oldValue interface{}) error
}
