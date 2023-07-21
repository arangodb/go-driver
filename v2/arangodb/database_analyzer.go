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

type DatabaseAnalyzer interface {
	// Analyzer returns the analyzer definition for the given analyzer
	Analyzer(ctx context.Context, name string) (Analyzer, error)

	// Analyzers returns an iterator to read all analyzers
	Analyzers(ctx context.Context) (AnalyzersResponseReader, error)
}

// TODO: consider renaming XXXResponseReader types to XXXCursor or XXXIterator
type AnalyzersResponseReader interface {
	Read() (Analyzer, error)
}
