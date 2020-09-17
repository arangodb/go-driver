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
// Author Adam Janikowski
//

package arangodb

import (
	"context"
)

type Database interface {
	// Name returns the name of the database.
	Name() string

	// Info fetches information about the database.
	Info(ctx context.Context) (DatabaseInfo, error)

	// Remove removes the entire database.
	// If the database does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	DatabaseCollection
	DatabaseTransaction
	DatabaseQuery
}
