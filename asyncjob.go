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

package driver

import (
	"context"
	"time"
)

type ClientAsyncJob interface {
	AsyncJob() AsyncJobService
}

type AsyncJobService interface {
	List(ctx context.Context, jobType AsyncJobStatusType, opts *AsyncJobListOptions) ([]string, error)
	Status(ctx context.Context, jobID string) (AsyncJobStatusType, error)
	Cancel(ctx context.Context, jobID string) (bool, error)
	Delete(ctx context.Context, deleteType AsyncJobDeleteType, opts *AsyncJobDeleteOptions) (bool, error)
}

type AsyncJobStatusType string

const (
	JobDone    AsyncJobStatusType = "done"
	JobPending AsyncJobStatusType = "pending"
)

type AsyncJobListOptions struct {
	// Count The maximum number of ids to return per call.
	// If not specified, a server-defined maximum value will be used.
	Count int `json:"count,omitempty"`
}

type AsyncJobDeleteType string

const (
	DeleteAllJobs     AsyncJobDeleteType = "all"
	DeleteExpiredJobs AsyncJobDeleteType = "expired"
	DeleteSingleJob   AsyncJobDeleteType = "single"
)

type AsyncJobDeleteOptions struct {
	JobID string    `json:"id,omitempty"`
	Stamp time.Time `json:"stamp,omitempty"`
}
