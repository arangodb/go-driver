// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/arangodb/shared"
	"github.com/arangodb/go-driver/v2/connection"
)

// newClientTask initializes a new task client with the given database name.
func newClientTask(client *client) *clientTask {
	return &clientTask{
		client: client,
	}
}

// will check all methods in ClientTasks are implemented with the clientTask struct.
var _ ClientTasks = &clientTask{}

type clientTask struct {
	client *client
}

type taskResponse struct {
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Command string          `json:"command,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Period  int64           `json:"period,omitempty"`
	Offset  float64         `json:"offset,omitempty"`
}

func newTask(client *client, resp *taskResponse) Task {
	return &task{
		client:  client,
		id:      resp.ID,
		name:    resp.Name,
		command: resp.Command,
		params:  resp.Params,
		period:  resp.Period,
		offset:  resp.Offset,
	}
}

type task struct {
	client  *client
	id      string
	name    string
	command string
	params  json.RawMessage
	period  int64
	offset  float64
}

func (t *task) ID() string {
	return t.id
}

func (t *task) Name() string {
	return t.name
}

func (t *task) Command() string {
	return t.command
}

func (t *task) Params(result interface{}) error {
	if t.params == nil {
		return nil
	}
	return json.Unmarshal(t.params, result)
}

func (t *task) Period() int64 {
	return t.period
}

func (t *task) Offset() float64 {
	return t.offset
}

func (c clientTask) Tasks(ctx context.Context) ([]Task, error) {
	urlEndpoint := connection.NewUrl("_api", "tasks") // Note: This should include database context, see below
	response := make([]taskResponse, 0)               // Direct array response
	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		result := make([]Task, len(response))
		for i, task := range response {
			fmt.Printf("Task %d: %+v\n", i, task)
			result[i] = newTask(c.client, &task)
		}
		return result, nil
	default:
		// Attempt to get error details from response headers or body
		return nil, shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c clientTask) Task(ctx context.Context, id string) (Task, error) {
	urlEndpoint := connection.NewUrl("_api", "tasks", url.PathEscape(id))

	response := struct {
		taskResponse          `json:",inline"`
		shared.ResponseStruct `json:",inline"`
	}{}

	resp, err := connection.CallGet(ctx, c.client.connection, urlEndpoint, &response)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	switch code := resp.Code(); code {
	case http.StatusOK:
		return newTask(c.client, &response.taskResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientTask) CreateTask(ctx context.Context, options *TaskOptions) (Task, error) {
	var urlEndpoint string
	if options.ID != "" {
		urlEndpoint = connection.NewUrl("_api", "tasks", url.PathEscape(options.ID))
	} else {
		urlEndpoint = connection.NewUrl("_api", "tasks")
	}
	// Prepare the request body
	createRequest := struct {
		ID      string          `json:"id,omitempty"`
		Name    string          `json:"name,omitempty"`
		Command string          `json:"command,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
		Period  int64           `json:"period,omitempty"`
		Offset  float64         `json:"offset,omitempty"`
	}{
		ID:      options.ID,
		Name:    options.Name,
		Command: options.Command,
		Period:  options.Period,
		Offset:  options.Offset,
	}

	if options.Params != nil {
		raw, err := json.Marshal(options.Params)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		createRequest.Params = raw
	}

	response := struct {
		shared.ResponseStruct `json:",inline"`
		taskResponse          `json:",inline"`
	}{}

	resp, err := connection.CallPost(ctx, c.client.connection, urlEndpoint, &response, &createRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch code := resp.Code(); code {
	case http.StatusCreated, http.StatusOK:
		return newTask(c.client, &response.taskResponse), nil
	default:
		return nil, response.AsArangoError()
	}
}

func (c clientTask) RemoveTask(ctx context.Context, id string) error {
	urlEndpoint := connection.NewUrl("_api", "tasks", url.PathEscape(id))

	resp, err := connection.CallDelete(ctx, c.client.connection, urlEndpoint, nil)
	if err != nil {
		return err
	}

	switch code := resp.Code(); code {
	case http.StatusAccepted, http.StatusOK:
		return nil
	default:
		return shared.NewResponseStruct().AsArangoErrorWithCode(code)
	}
}

func (c clientTask) CreateTaskWithID(ctx context.Context, id string, options *TaskOptions) (Task, error) {
	// Check if task already exists
	existingTask, err := c.Task(ctx, id)
	fmt.Printf("Checking existing task with ID: %s, existingTask: %v, Error:%v", id, existingTask, err)
	if err == nil && existingTask != nil {
		return nil, &shared.ArangoError{
			Code:         http.StatusConflict,
			ErrorMessage: fmt.Sprintf("Task with ID %s already exists", id),
		}
	}

	// Set the ID and call CreateTask
	options.ID = id
	return c.CreateTask(ctx, options)
}
