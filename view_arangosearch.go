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

package driver

import (
	"context"
)

// ArangoSearchView provides access to the information of a view.
// Views are only available in ArangoDB 3.4 and higher.
type ArangoSearchView interface {
	// Include generic View functions
	View

	// Properties fetches extended information about the view.
	Properties(ctx context.Context) (ArangoSearchViewProperties, error)

	// SetProperties changes properties of the view.
	SetProperties(ctx context.Context, options ArangoSearchViewProperties) error
}

// ArangoSearchViewProperties contains properties an an ArangoSearch view.
type ArangoSearchViewProperties struct {
	// TODO include locale, commit, ...

	// Links contains the properties for how individual collections
	// are indexed in thie view.
	// The key of the map are collection names.
	Links ArangoSearchLinks `json:"links,omitempty"`
}

// ArangoSearchLinks is a strongly typed map containing links between a
// collection and a view.
// The keys in the map are collection names.
type ArangoSearchLinks map[string]ArangoSearchElementProperties

// ArangoSearchFields is a strongly typed map containing properties per field.
// The keys in the map are field names.
type ArangoSearchFields map[string]ArangoSearchElementProperties

// ArangoSearchElementProperties contains properties that specify how an element
// is indexed in an ArangoSearch view.
// Note that this structure is recursive. Settings not specified (nil)
// at a given level will inherit their setting from a lower level.
type ArangoSearchElementProperties struct {
	// The list of analyzers to be used for indexing of string values. Defaults to ["identify"].
	Analyzers []string `json:"analyzers,omitempty"`
	// If set to true, all fields of this element will be indexed. Defaults to false.
	IncludeAllFields *bool `json:"includeAllFields,omitempty"`
	// If set to true, values in a listed are treated as separate values. Defaults to false.
	TrackListPositions *bool `json:"trackListPositions,omitempty"`
	// This values specifies how the view should track values.
	StoreValues ArangoSearchStoreValues `json:"storeValues,omitempty"`
	// Fields contains the properties for individual fields of the element.
	// The key of the map are field names.
	Fields ArangoSearchFields `json:"fields,omitempty"`
}

// ArangoSearchStoreValues is the type of the StoreValues option of an ArangoSearch element.
type ArangoSearchStoreValues string

const (
	// ArangoSearchStoreValuesNone specifies that a view should not store values.
	ArangoSearchStoreValuesNone ArangoSearchStoreValues = "none"
	// ArangoSearchStoreValuesID specifies that a view should only store
	// information about value presence, to allow use of the EXISTS() function.
	ArangoSearchStoreValuesID ArangoSearchStoreValues = "id"
)
