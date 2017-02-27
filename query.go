//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

type queryRequest struct {
	// indicates whether the number of documents in the result set should be returned in the "count" attribute of the result.
	// Calculating the "count" attribute might have a performance impact for some queries in the future so this option is turned off by default, and "count" is only returned when requested.
	Count bool `json:"count,omitempty"`
	// maximum number of result documents to be transferred from the server to the client in one roundtrip. If this attribute is not set, a server-controlled default value will be used. A batchSize value of 0 is disallowed.
	BatchSize int `json:"batchSize,omitempty"`
	// flag to determine whether the AQL query cache shall be used. If set to false, then any query cache lookup will be skipped for the query. If set to true, it will lead to the query cache being checked for the query if the query cache mode is either on or demand.
	Cache bool `json:"cache,omitempty"`
	// the maximum number of memory (measured in bytes) that the query is allowed to use. If set, then the query will fail with error "resource limit exceeded" in case it allocates too much memory. A value of 0 indicates that there is no memory limit.
	MemoryLimit int64 `json:"memoryLimit,omitempty"`
	// The time-to-live for the cursor (in seconds). The cursor will be removed on the server automatically after the specified amount of time. This is useful to ensure garbage collection of cursors that are not fully fetched by clients. If not set, a server-defined value will be used.
	TTL float64 `json:"ttl,omitempty"`
	// contains the query string to be executed
	Query string `json:"query"`
	// key/value pairs representing the bind parameters.
	BindVars map[string]interface{} `json:"bindVars,omitempty"`
	Options  struct {
		// If set to true, then the additional query profiling information will be returned in the sub-attribute profile of the extra return attribute if the query result is not served from the query cache.
		Profile bool `json:"profile,omitempty"`
		// A list of to-be-included or to-be-excluded optimizer rules can be put into this attribute, telling the optimizer to include or exclude specific rules. To disable a rule, prefix its name with a -, to enable a rule, prefix it with a +. There is also a pseudo-rule all, which will match all optimizer rules.
		OptimizerRules string `json:"optimizer.rules,omitempty"`
		// This enterprise parameter allows to configure how long a DBServer will have time to bring the satellite collections involved in the query into sync. The default value is 60.0 (seconds). When the max time has been reached the query will be stopped.
		SatelliteSyncWait float64 `json:"satelliteSyncWait,omitempty"`
		// if set to true and the query contains a LIMIT clause, then the result will have an extra attribute with the sub-attributes stats and fullCount, { ... , "extra": { "stats": { "fullCount": 123 } } }. The fullCount attribute will contain the number of documents in the result before the last LIMIT in the query was applied. It can be used to count the number of documents that match certain filter criteria, but only return a subset of them, in one go. It is thus similar to MySQL's SQL_CALC_FOUND_ROWS hint. Note that setting the option will disable a few LIMIT optimizations and may lead to more documents being processed, and thus make queries run longer. Note that the fullCount attribute will only be present in the result if the query has a LIMIT clause and the LIMIT clause is actually used in the query.
		FullCount bool `json:"fullCount,omitempty"`
		// Limits the maximum number of plans that are created by the AQL query optimizer.
		MaxPlans int `json:"maxPlans,omitempty"`
	} `json:"options,omitempty"`
}
