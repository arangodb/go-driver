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

package arangodb

import (
	"context"
)

// ClientTasks defines the interface for managing tasks in ArangoDB.
type ClientTasks interface {
	// Task retrieves an existing task by its ID.
	// If no task with the given ID exists, a NotFoundError is returned.
	Task(ctx context.Context, id string) (Task, error)

	// Tasks returns a list of all tasks on the server.
	Tasks(ctx context.Context) ([]Task, error)

	// CreateTask creates a new task with the specified options.
	CreateTask(ctx context.Context, options *TaskOptions) (Task, error)

	// If a task with the given ID already exists, a Conflict error is returned.
	CreateTaskWithID(ctx context.Context, id string, options *TaskOptions) (Task, error)

	// RemoveTask deletes an existing task by its ID.
	RemoveTask(ctx context.Context, id string) error
}

// TaskOptions contains options for creating a new task.
type TaskOptions struct {
	// ID is an optional identifier for the task.
	ID string `json:"id,omitempty"`
	// Name is an optional name for the task.
	Name string `json:"name,omitempty"`

	// Command is the JavaScript code to be executed.
	Command string `json:"command"`

	// Params are optional parameters passed to the command.
	Params interface{} `json:"params,omitempty"`

	// Period is the interval (in seconds) at which the task runs periodically.
	// If zero, the task runs once after the offset.
	Period int64 `json:"period,omitempty"`

	// Offset is the delay (in milliseconds) before the task is first executed.
	Offset float64 `json:"offset,omitempty"`
}

// Task provides access to a single task on the server.
type Task interface {
	// ID returns the ID of the task.
	ID() string

	// Name returns the name of the task.
	Name() string

	// Command returns the JavaScript code of the task.
	Command() string

	// Params returns the parameters of the task.
	Params(result interface{}) error

	// Period returns the period (in seconds) of the task.
	Period() int64

	// Offset returns the offset (in milliseconds) of the task.
	Offset() float64
}
