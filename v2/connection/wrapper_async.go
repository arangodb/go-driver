//
// DISCLAIMER
//
// Copyright 2023-2024 ArangoDB GmbH, Cologne, Germany
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

package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
)

const (
	ArangoHeaderAsyncIDKey = "x-arango-async-id"
	ArangoHeaderAsyncKey   = "x-arango-async"
	ArangoHeaderAsyncValue = "store"
)

type AsyncConnectionWrapper struct {
	Connection
}

func NewConnectionAsyncWrapper(conn Connection) Connection {
	return &AsyncConnectionWrapper{
		Connection: conn,
	}
}

func (a *AsyncConnectionWrapper) Do(ctx context.Context, request Request, output interface{}, allowedStatusCodes ...int) (Response, error) {
	if id, ok := HasAsyncID(ctx); ok {
		// We have ID Set so a job is in progress, request should be done with job api
		req, err := a.Connection.NewRequest(http.MethodPut, path.Join("/_api/job", id))
		if err != nil {
			return nil, err
		}

		resp, err := a.Connection.Do(ctx, req, output, allowedStatusCodes...)
		if err != nil {
			return nil, err
		}

		switch resp.Code() {
		case http.StatusNoContent:

			asyncID := resp.Header(ArangoHeaderAsyncIDKey)
			if asyncID == id {
				// Job is done
				return resp, nil
			}

			// Job is in progress
			return nil, ErrorAsyncJobInProgress{id}
		default:
			return resp, nil
		}
	} else if IsAsyncRequest(ctx) {
		// Send request with async header
		request.AddHeader(ArangoHeaderAsyncKey, ArangoHeaderAsyncValue)

		resp, err := a.Connection.Do(ctx, request, nil, http.StatusAccepted)
		if err != nil {
			return nil, err
		}

		if asyncID := resp.Header(ArangoHeaderAsyncIDKey); len(asyncID) == 0 {
			return nil, errors.New("missing async key response")
		} else {
			return nil, ErrorAsyncJobInProgress{asyncID}
		}
	} else {
		return a.Connection.Do(ctx, request, output, allowedStatusCodes...)
	}
}

func IsAsyncJobInProgress(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	// Unwrap error WithStack case
	var v ErrorAsyncJobInProgress
	if errors.As(errors.Unwrap(err), &v) {
		return v.jobID, true
	}

	var v2 ErrorAsyncJobInProgress
	if errors.As(err, &v2) {
		return v2.jobID, true
	}

	return "", false
}

type ErrorAsyncJobInProgress struct {
	jobID string
}

func (a ErrorAsyncJobInProgress) Error() string {
	return fmt.Sprintf("Job with ID %s in progress", a.jobID)
}
