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

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

var _ ClientAsyncJob = &clientAsyncJob{}

type clientAsyncJob struct {
	client *client
}

func newClientAsyncJob(client *client) *clientAsyncJob {
	return &clientAsyncJob{
		client: client,
	}
}

func (c *clientAsyncJob) AsyncJobList(ctx context.Context, jobType AsyncJobStatusType, opts *AsyncJobListOptions) ([]string, error) {
	var result []string

	var mods []connection.RequestModifier
	if opts != nil {
		if opts.Count != 0 {
			mods = append(mods, connection.WithQuery("count", fmt.Sprintf("%d", opts.Count)))
		}
	}

	resp, err := connection.CallGet(ctx, c.client.connection, c.url(string(jobType)), &result, mods...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return result, nil
	default:
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c *clientAsyncJob) AsyncJobStatus(ctx context.Context, jobID string) (AsyncJobStatusType, error) {
	resp, err := connection.CallGet(ctx, c.client.connection, c.url(jobID), nil)
	if err != nil {
		return "", errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return JobDone, nil
	case http.StatusNotFound:
		return JobPending, nil
	default:
		return "", shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

type cancelResponse struct {
	Result bool `json:"result"`
}

func (c *clientAsyncJob) AsyncJobCancel(ctx context.Context, jobID string) (bool, error) {
	var data cancelResponse
	resp, err := connection.CallPut(ctx, c.client.connection, c.url(jobID, "cancel"), &data, nil)
	if err != nil {
		return false, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return data.Result, nil
	default:
		return false, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

type deleteResponse struct {
	Result bool `json:"result"`
}

func (c *clientAsyncJob) AsyncJobDelete(ctx context.Context, deleteType AsyncJobDeleteType, opts *AsyncJobDeleteOptions) (bool, error) {
	p := ""
	switch deleteType {
	case DeleteAllJobs:
		p = c.url(string(deleteType))
	case DeleteExpiredJobs:
		if opts == nil || opts.Stamp.IsZero() {
			return false, errors.WithStack(shared.InvalidArgumentError{Message: "stamp must be set when deleting expired jobs"})
		}
		p = c.url(string(deleteType))
	case DeleteSingleJob:
		if opts == nil || opts.JobID == "" {
			return false, errors.WithStack(shared.InvalidArgumentError{Message: "jobID must be set when deleting a single job"})
		}
		p = c.url(opts.JobID)
	}

	var mods []connection.RequestModifier
	if deleteType == DeleteExpiredJobs {
		mods = append(mods, connection.WithQuery("stamp", fmt.Sprintf("%d", opts.Stamp.Unix())))
	}

	var data deleteResponse
	resp, err := connection.CallDelete(ctx, c.client.connection, p, &data, mods...)
	if err != nil {
		return false, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusOK:
		return data.Result, nil
	default:
		return false, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c *clientAsyncJob) url(parts ...string) string {
	return connection.NewUrl(append([]string{"_api", "job"}, parts...)...)
}
