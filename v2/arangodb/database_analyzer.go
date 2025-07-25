//
// DISCLAIMER
//
// Copyright 2024-2025 ArangoDB GmbH, Cologne, Germany
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

type DatabaseAnalyzer interface {
	// EnsureCreatedAnalyzer creates an Analyzer for the database, if it does not already exist.
	// It returns the Analyser object together with a boolean indicating if the Analyzer was newly created (true) or pre-existing (false).
	EnsureCreatedAnalyzer(ctx context.Context, analyzer *AnalyzerDefinition) (Analyzer, bool, error)

	// Analyzer returns the analyzer definition for the given analyzer
	Analyzer(ctx context.Context, name string) (Analyzer, error)

	// Analyzers return an iterator to read all analyzers
	Analyzers(ctx context.Context) (AnalyzersResponseReader, error)
}

type AnalyzersResponseReader interface {
	// Read returns next Analyzer. If no Analyzers left, shared.NoMoreDocumentsError returned
	Read() (Analyzer, error)
}
