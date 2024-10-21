//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

	"github.com/arangodb/go-driver/v2/connection"
)

type DatabaseQuery interface {
	// Query performs an AQL query, returning a cursor used to iterate over the returned documents.
	// Note that the returned Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
	Query(ctx context.Context, query string, opts *QueryOptions) (Cursor, error)

	// QueryBatch performs an AQL query, returning a cursor used to iterate over the returned documents in batches.
	// In contrast to Query, QueryBatch does not load all documents into memory, but returns them in batches and allows for retries in case of errors.
	// Note that the returned Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed
	QueryBatch(ctx context.Context, query string, opts *QueryOptions, result interface{}) (CursorBatch, error)

	// ValidateQuery validates an AQL query.
	// When the query is valid, nil returned, otherwise an error is returned.
	// The query is not executed.
	ValidateQuery(ctx context.Context, query string) error

	// ExplainQuery explains an AQL query and return information about it.
	ExplainQuery(ctx context.Context, query string, bindVars map[string]interface{}, opts *ExplainQueryOptions) (ExplainQueryResult, error)
}

type QuerySubOptions struct {
	// If you set this option to true and execute the query against a cluster deployment, then the Coordinator is
	// allowed to read from any shard replica and not only from the leader.
	// You may observe data inconsistencies (dirty reads) when reading from followers, namely obsolete revisions of
	// documents because changes have not yet been replicated to the follower, as well as changes to documents before
	// they are officially committed on the leader.
	//
	//This feature is only available in the Enterprise Edition.
	AllowDirtyReads bool `json:"allowDirtyReads,omitempty"`

	// AllowRetry If set to `true`, ArangoDB will store cursor results in such a way
	// that batch reads can be retried in the case of a communication error.
	AllowRetry bool `json:"allowRetry,omitempty"`

	// When set to true, the query will throw an exception and abort instead of producing a warning.
	// This option should be used during development to catch potential issues early.
	// When the attribute is set to false, warnings will not be propagated to exceptions and will be returned
	// with the query result. There is also a server configuration option --query.fail-on-warning for setting
	// the default value for failOnWarning so it does not need to be set on a per-query level.
	FailOnWarning *bool `json:"failOnWarning,omitempty"`

	// If set to true or not specified, this will make the query store the data it reads via the RocksDB storage engine
	// in the RocksDB block cache. This is usually the desired behavior. The option can be set to false for queries that
	// are known to either read a lot of data which would thrash the block cache, or for queries that read data which
	// are known to be outside of the hot set. By setting the option to false, data read by the query will not make it
	// into the RocksDB block cache if not already in there, thus leaving more room for the actual hot set.
	FillBlockCache bool `json:"fillBlockCache,omitempty"`

	// if set to true and the query contains a LIMIT clause, then the result will have an extra attribute with the sub-attributes
	// stats and fullCount, { ... , "extra": { "stats": { "fullCount": 123 } } }. The fullCount attribute will contain the number
	// of documents in the result before the last LIMIT in the query was applied. It can be used to count the number of documents
	// that match certain filter criteria, but only return a subset of them, in one go. It is thus similar to MySQL's SQL_CALC_FOUND_ROWS hint.
	// Note that setting the option will disable a few LIMIT optimizations and may lead to more documents being processed, and
	// thus make queries run longer. Note that the fullCount attribute will only be present in the result if the query has a LIMIT clause
	// and the LIMIT clause is actually used in the query.
	FullCount bool `json:"fullCount,omitempty"`

	// The maximum number of operations after which an intermediate commit is performed automatically.
	IntermediateCommitCount *int `json:"intermediateCommitCount,omitempty"`

	// The maximum total size of operations after which an intermediate commit is performed automatically.
	IntermediateCommitSize *int `json:"intermediateCommitSize,omitempty"`

	// A threshold for the maximum number of OR sub-nodes in the internal representation of an AQL FILTER condition.
	// Yon can use this option to limit the computation time and memory usage when converting complex AQL FILTER
	// conditions into the internal DNF (disjunctive normal form) format. FILTER conditions with a lot of logical
	// branches (AND, OR, NOT) can take a large amount of processing time and memory. This query option limits
	// the computation time and memory usage for such conditions.
	//
	// Once the threshold value is reached during the DNF conversion of a FILTER condition, the conversion is aborted,
	// and the query continues with a simplified internal representation of the condition,
	// which cannot be used for index lookups.
	//
	// You can set the threshold globally instead of per query with the --query.max-dnf-condition-members startup option.
	MaxDNFConditionMembers *int `json:"maxDNFConditionMembers,omitempty"`

	// The number of execution nodes in the query plan after that stack splitting is performed to avoid a potential
	// stack overflow. Defaults to the configured value of the startup option `--query.max-nodes-per-callstack`.
	// This option is only useful for testing and debugging and normally does not need any adjustment.
	MaxNodesPerCallstack *int `json:"maxNodesPerCallstack,omitempty"`

	// Limits the maximum number of plans that are created by the AQL query optimizer.
	MaxNumberOfPlans *int `json:"maxNumberOfPlans,omitempty"`

	// MaxRuntime specify the timeout which can be used to kill a query on the server after the specified
	// amount in time. The timeout value is specified in seconds. A value of 0 means no timeout will be enforced.
	MaxRuntime float64 `json:"maxRuntime,omitempty"`

	// The transaction size limit in bytes.
	MaxTransactionSize *int `json:"maxTransactionSize,omitempty"`

	// Limits the maximum number of warnings a query will return. The number of warnings a query will return is limited
	// to 10 by default, but that number can be increased or decreased by setting this attribute.
	MaxWarningCount *int `json:"maxWarningCount,omitempty"`

	// Optimizer contains options related to the query optimizer.
	Optimizer QuerySubOptionsOptimizer `json:"optimizer,omitempty"`

	// Profile If set to 1, then the additional query profiling information is returned in the profile sub-attribute
	// of the extra return attribute, unless the query result is served from the query cache.
	// If set to 2, the query includes execution stats per query plan node in stats.nodes
	// sub-attribute of the extra return attribute.
	// Additionally, the query plan is returned in the extra.plan sub-attribute.
	Profile uint `json:"profile,omitempty"`

	// This Enterprise Edition parameter allows to configure how long a DBServer will have time to bring the satellite collections
	// involved in the query into sync. The default value is 60.0 (seconds). When the max time has been reached the query will be stopped.
	SatelliteSyncWait float64 `json:"satelliteSyncWait,omitempty"`

	// Let AQL queries (especially graph traversals) treat collection to which a user has no access rights for as if
	// these collections are empty. Instead of returning a forbidden access error, your queries execute normally.
	// This is intended to help with certain use-cases: A graph contains several collections and different users
	// execute AQL queries on that graph. You can naturally limit the accessible results by changing the access rights
	// of users on collections.
	//
	// This feature is only available in the Enterprise Edition.
	SkipInaccessibleCollections *bool `json:"skipInaccessibleCollections,omitempty"`

	// This option allows queries to store intermediate and final results temporarily on disk if the amount of memory
	// used (in bytes) exceeds the specified value. This is used for decreasing the memory usage during the query execution.
	//
	// This option only has an effect on queries that use the SORT operation but without a LIMIT, and if you enable
	//the spillover feature by setting a path for the directory to store the temporary data in with
	// the --temp.intermediate-results-path startup option.
	//
	// Default value: 128MB.
	SpillOverThresholdMemoryUsage *int `json:"spillOverThresholdMemoryUsage,omitempty"`

	// This option allows queries to store intermediate and final results temporarily on disk if the number of rows
	// produced by the query exceeds the specified value. This is used for decreasing the memory usage during the query
	// execution. In a query that iterates over a collection that contains documents, each row is a document, and in
	// a query that iterates over temporary values (i.e. FOR i IN 1..100), each row is one of such temporary values.
	//
	// This option only has an effect on queries that use the SORT operation but without a LIMIT, and if you enable
	// the spillover feature by setting a path for the directory to store the temporary data in with
	// the --temp.intermediate-results-path startup option.
	//
	// Default value: 5000000 rows.
	SpillOverThresholdNumRows *int `json:"spillOverThresholdNumRows,omitempty"`

	// Specify true and the query will be executed in a streaming fashion. The query result is not stored on
	// the server, but calculated on the fly. Beware: long-running queries will need to hold the collection
	// locks for as long as the query cursor exists. When set to false a query will be executed right away in
	// its entirety.
	Stream bool `json:"stream,omitempty"`

	/* Not officially documented options, please use them with care. */

	// [unofficial] Limits the maximum number of plans that are created by the AQL query optimizer.
	MaxPlans int `json:"maxPlans,omitempty"`

	// [unofficial] ShardId query option
	ShardIds []string `json:"shardIds,omitempty"`

	// [unofficial] This query option can be used in complex queries in case the query optimizer cannot
	// automatically detect that the query can be limited to only a single server (e.g. in a disjoint smart graph case).
	ForceOneShardAttributeValue *string `json:"forceOneShardAttributeValue,omitempty"`
}

type QueryOptions struct {
	// Set this to true to allow the Coordinator to ask any shard replica for the data, not only the shard leader.
	// This may result in “dirty reads”.
	// This option is ignored if this operation is part of a DatabaseTransaction (TransactionID option).
	// The header set when creating the transaction decides about dirty reads for the entire transaction,
	// not the individual read operations.
	AllowDirtyReads *bool `json:"-"`

	// To make this operation a part of a Stream Transaction, set this header to the transaction ID returned by the
	// DatabaseTransaction.BeginTransaction() method.
	TransactionID string `json:"-"`

	// Indicates whether the number of documents in the result set should be returned in the "count" attribute of the result.
	// Calculating the "count" attribute might have a performance impact for some queries in the future so this option is
	// turned off by default, and "count" is only returned when requested.
	Count bool `json:"count,omitempty"`

	// maximum number of result documents to be transferred from the server to the client in one roundtrip.
	// If this attribute is not set, a server-controlled default value will be used. A batchSize value of 0 is disallowed.
	BatchSize int `json:"batchSize,omitempty"`

	// flag to determine whether the AQL query cache shall be used. If set to false, then any query cache lookup
	// will be skipped for the query. If set to true, it will lead to the query cache being checked for the query
	// if the query cache mode is either on or demand.
	Cache bool `json:"cache,omitempty"`

	// the maximum number of memory (measured in bytes) that the query is allowed to use. If set, then the query will fail
	// with error "resource limit exceeded" in case it allocates too much memory. A value of 0 indicates that there is no memory limit.
	MemoryLimit int64 `json:"memoryLimit,omitempty"`

	// The time-to-live for the cursor (in seconds). The cursor will be removed on the server automatically after the specified
	// amount of time. This is useful to ensure garbage collection of cursors that are not fully fetched by clients.
	// If not set, a server-defined value will be used.
	TTL float64 `json:"ttl,omitempty"`

	// key/value pairs representing the bind parameters.
	BindVars map[string]interface{} `json:"bindVars,omitempty"`
	Options  QuerySubOptions        `json:"options,omitempty"`
}

func (q *QueryOptions) modifyRequest(r connection.Request) error {
	if q == nil {
		return nil
	}

	if q.AllowDirtyReads != nil {
		r.AddHeader(HeaderDirtyReads, boolToString(*q.AllowDirtyReads))
	}

	if q.TransactionID != "" {
		r.AddHeader(HeaderTransaction, q.TransactionID)
	}

	return nil
}

// QuerySubOptionsOptimizer describes optimization's settings for AQL queries.
type QuerySubOptionsOptimizer struct {
	// A list of to-be-included or to-be-excluded optimizer rules can be put into this attribute,
	// telling the optimizer to include or exclude specific rules.
	// To disable a rule, prefix its name with a -, to enable a rule, prefix it with a +.
	// There is also a pseudo-rule all, which will match all optimizer rules.
	Rules []string `json:"rules,omitempty"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type ExplainQueryOptimizerOptions struct {
	// A list of to-be-included or to-be-excluded optimizer rules can be put into this attribute,
	// telling the optimizer to include or exclude specific rules.
	//  To disable a rule, prefix its name with a "-", to enable a rule, prefix it with a "+".
	// There is also a pseudo-rule "all", which matches all optimizer rules. "-all" disables all rules.
	Rules []string `json:"rules,omitempty"`
}

type ExplainQueryOptions struct {
	// If set to true, all possible execution plans will be returned.
	// The default is false, meaning only the optimal plan will be returned.
	AllPlans bool `json:"allPlans,omitempty"`

	// An optional maximum number of plans that the optimizer is allowed to generate.
	// Setting this attribute to a low value allows to put a cap on the amount of work the optimizer does.
	MaxNumberOfPlans *int `json:"maxNumberOfPlans,omitempty"`

	// Options related to the query optimizer.
	Optimizer ExplainQueryOptimizerOptions `json:"optimizer,omitempty"`
}

type ExplainQueryResultExecutionNodeRaw map[string]interface{}

type ExplainQueryResultExecutionCollection struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ExplainQueryResultExecutionVariable struct {
	ID                           int    `json:"id"`
	Name                         string `json:"name"`
	IsDataFromCollection         bool   `json:"isDataFromCollection"`
	IsFullDocumentFromCollection bool   `json:"isFullDocumentFromCollection"`
}

type ExplainQueryResultPlan struct {
	// Execution nodes of the plan.
	NodesRaw []ExplainQueryResultExecutionNodeRaw `json:"nodes,omitempty"`

	// List of rules the optimizer applied
	Rules []string `json:"rules,omitempty"`

	// List of collections used in the query
	Collections []ExplainQueryResultExecutionCollection `json:"collections,omitempty"`

	// List of variables used in the query (note: this may contain internal variables created by the optimizer)
	Variables []ExplainQueryResultExecutionVariable `json:"variables,omitempty"`

	// The total estimated cost for the plan. If there are multiple plans, the optimizer will choose the plan with the lowest total cost
	EstimatedCost float64 `json:"estimatedCost,omitempty"`

	// The estimated number of results.
	EstimatedNrItems int `json:"estimatedNrItems,omitempty"`
}

type ExplainQueryResultExecutionStats struct {
	RulesExecuted   int     `json:"rulesExecuted,omitempty"`
	RulesSkipped    int     `json:"rulesSkipped,omitempty"`
	PlansCreated    int     `json:"plansCreated,omitempty"`
	PeakMemoryUsage uint64  `json:"peakMemoryUsage,omitempty"`
	ExecutionTime   float64 `json:"executionTime,omitempty"`
}

type ExplainQueryResult struct {
	Plan  ExplainQueryResultPlan   `json:"plan,omitempty"`
	Plans []ExplainQueryResultPlan `json:"plans,omitempty"`

	// List of warnings that occurred during optimization or execution plan creation
	Warnings []string `json:"warnings,omitempty"`

	// Info about optimizer statistics
	Stats ExplainQueryResultExecutionStats `json:"stats,omitempty"`

	// Cacheable states whether the query results can be cached on the server if the query result cache were used.
	// This attribute is not present when allPlans is set to true.
	Cacheable *bool `json:"cacheable,omitempty"`
}
