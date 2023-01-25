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
// Author Jakub Wierzbowski
//

package arangodb

// AnalyzerFeature specifies a feature to an analyzer
type AnalyzerFeature string

const (
	// AnalyzerFeatureFrequency how often a term is seen, required for PHRASE()
	AnalyzerFeatureFrequency AnalyzerFeature = "frequency"

	// AnalyzerFeatureNorm the field normalization factor
	AnalyzerFeatureNorm AnalyzerFeature = "norm"

	// AnalyzerFeaturePosition sequentially increasing term position, required for PHRASE(). If present then the frequency feature is also required
	AnalyzerFeaturePosition AnalyzerFeature = "position"

	// AnalyzerFeatureOffset can be specified if 'position' feature is set
	AnalyzerFeatureOffset AnalyzerFeature = "offset"
)
