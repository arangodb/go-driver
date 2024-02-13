//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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
)

type View interface {
	// Name returns the name of the view.
	Name() string

	// Type returns the type of this view.
	Type() ViewType

	// ArangoSearchView returns this view as an ArangoSearch view.
	// When the type of the view is not ArangoSearch, an error is returned.
	ArangoSearchView() (ArangoSearchView, error)

	// ArangoSearchViewAlias returns this view as an ArangoSearch view alias.
	// When the type of the view is not ArangoSearch alias, an error is returned.
	ArangoSearchViewAlias() (ArangoSearchViewAlias, error)

	// Database returns the database containing the view.
	Database() Database

	// Rename renames the view (SINGLE server only).
	Rename(ctx context.Context, newName string) error

	// Remove removes the entire view.
	// If the view does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// RemoveWithOptions removes the entire view.
	// If the view does not exist, a NotFoundError is returned.
	RemoveWithOptions(ctx context.Context, opts *RemoveViewOptions) error
}

// ViewType is the type of view.
type ViewType string

const (
	// ViewTypeArangoSearch specifies an ArangoSearch view type.
	ViewTypeArangoSearch = ViewType("arangosearch")

	// ViewTypeSearchAlias specifies an ArangoSearch view type alias.
	ViewTypeSearchAlias = ViewType("search-alias")
)

type ViewBase struct {
	ID               string   `json:"id,omitempty"`
	GloballyUniqueId string   `json:"globallyUniqueId,omitempty"`
	Type             ViewType `json:"type,omitempty"`
	Name             string   `json:"name,omitempty"`
}
