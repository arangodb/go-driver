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

package async

import (
	"context"
	"errors"
	"net/http"
	"path"

	"github.com/arangodb/go-driver"
)

const (
	ArangoHeaderAsyncIDKey = "x-arango-async-id"
	ArangoHeaderAsyncKey   = "x-arango-async"
	ArangoHeaderAsyncValue = "store"
)

type asyncConnectionWrapper struct {
	driver.Connection
}

func NewConnectionAsyncWrapper(conn driver.Connection) driver.Connection {
	return &asyncConnectionWrapper{conn}
}

func (a asyncConnectionWrapper) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if id, ok := driver.IsAsyncIDSet(ctx); ok {
		// We have ID Set so a job is in progress, request should be done with job api
		req, err := a.Connection.NewRequest(http.MethodPut, path.Join("/_api/job", id))
		if err != nil {
			return nil, err
		}

		resp, err := a.Connection.Do(ctx, req)
		if err != nil {
			return nil, err
		}

		switch resp.StatusCode() {
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
	} else if driver.IsAsyncRequest(ctx) {
		// Send request with async header
		req.SetHeader(ArangoHeaderAsyncKey, ArangoHeaderAsyncValue)

		resp, err := a.Connection.Do(ctx, req)
		if err != nil {
			return nil, err
		}

		switch resp.StatusCode() {
		case http.StatusAccepted:
			if asyncID := resp.Header(ArangoHeaderAsyncIDKey); len(asyncID) == 0 {
				return nil, errors.New("missing async key response")
			} else {
				return nil, ErrorAsyncJobInProgress{asyncID}
			}
		default:
			// we expect a 202 status code only
			return nil, resp.CheckStatus(http.StatusAccepted)
		}
	} else {
		return a.Connection.Do(ctx, req)
	}
}
