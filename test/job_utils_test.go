//
// DISCLAIMER
//
// Copyright 2019 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package test

import (
	"context"
	"testing"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/agency"
	"github.com/stretchr/testify/require"
)

var (
	agencyJobStateKeyPrefixes = [][]string{
		{"arango", "Target", "ToDo"},
		{"arango", "Target", "Pending"},
		{"arango", "Target", "Finished"},
		{"arango", "Target", "Failed"},
	}
)

type agencyJob struct {
	Reason string `json:"reason,omitempty"`
	Server string `json:"server,omitempty"`
	JobID  string `json:"jobId,omitempty"`
	Type   string `json:"type,omitempty"`
}

type jobStatus int

const (
	JobNotFound jobStatus = 0
	JobToDo     jobStatus = 1
	JobPending  jobStatus = 2
	JobFinished jobStatus = 3
	JobFailed   jobStatus = 4
)

func fetchJobStatus(t *testing.T, jobID string, client driver.Client) jobStatus {
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()

	a, err := getHttpAuthAgencyConnection(ctx, t, client)
	require.NoError(t, err)

	for _, keyPrefix := range agencyJobStateKeyPrefixes {
		key := append(keyPrefix, jobID)
		var job agencyJob
		if err := a.ReadKey(ctx, key, &job); err == nil {
			switch keyPrefix[len(keyPrefix)-1] {
			case "ToDo":
				return JobToDo
			case "Pending":
				return JobPending
			case "Finished":
				return JobFinished
			case "Failed":
				return JobFailed
			}
		} else if agency.IsKeyNotFound(err) {
			continue
		} else {
			require.NoError(t, err)
		}
	}

	return JobNotFound
}

// waitForJob will check if job exists and return function which will wait for job to finish (desired to use with defer)
func waitForJob(t *testing.T, jobID string, client driver.Client) func() {
	require.NotEqual(t, JobNotFound, fetchJobStatus(t, jobID, client))

	t.Logf("waiting for job %s before test end", jobID)

	return func() {
		t.Logf("waiting for job %s to finish", jobID)
		err := retry(125*time.Millisecond, time.Minute, func() error {
			result := fetchJobStatus(t, jobID, client)

			require.NotEqual(t, JobFailed, result, "job failed")

			if result == JobFinished {
				return interrupt{}
			}

			t.Logf("(%s) job %s not yet finished - %d", time.Now().String(), jobID, result)

			return nil
		})
		require.NoError(t, err)
	}
}
