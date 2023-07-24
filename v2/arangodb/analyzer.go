//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

import "context"

type Analyzer interface {
	Name() string
	Database() Database

	// Type returns the analyzer type
	Type() ArangoSearchAnalyzerType

	// UniqueName returns the unique name: <database>::<analyzer-name>
	UniqueName() string

	// Definition returns the analyzer definition
	Definition() AnalyzerDefinition

	// Remove the analyzer
	Remove(ctx context.Context, force bool) error
}

type AnalyzerDefinition struct {
	Name       string                         `json:"name,omitempty"`
	Type       ArangoSearchAnalyzerType       `json:"type,omitempty"`
	Properties ArangoSearchAnalyzerProperties `json:"properties,omitempty"`
	Features   []ArangoSearchFeature          `json:"features,omitempty"`
}
