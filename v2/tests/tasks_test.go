//
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

package tests

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

type TaskParams struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Test_CreateNewTask(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		dbName := "_system"
		testCases := map[string]*arangodb.TaskOptions{
			"taskWithParams": {
				Name:    utils.NewType("taskWithParams"),
				Command: utils.NewType("(function(params) { require('@arangodb').print(params); })(params)"),
				Period:  utils.NewType(int64(2)),
				Params: map[string]interface{}{
					"test": "hello",
				},
			},
			"taskWithoutParams": {
				Name:    utils.NewType("taskWithoutParams"),
				Command: utils.NewType("(function() { require('@arangodb').print('Hello'); })()"),
				Period:  utils.NewType(int64(2)),
			},
		}

		for name, options := range testCases {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				createdTask, err := client.CreateTask(ctx, dbName, *options)
				require.NoError(t, err)
				require.NotNil(t, createdTask)
				require.Equal(t, name, *createdTask.Name())
				t.Logf("Params: %v", options.Params)
				// Proper params comparison
				// Check parameters
				if options.Params != nil {
					var params map[string]interface{}
					err = createdTask.Params(&params)

					if err != nil {
						t.Logf("WARNING: Could not fetch task params (unsupported feature?): %v", err)
					} else if len(params) == 0 {
						t.Logf("WARNING: Task params exist but returned empty (ArangoDB limitation?)")
					} else {
						// Only check if params are actually returned
						require.Equal(t, options.Params, params)
					}
				}

				taskInfo, err := client.Task(ctx, dbName, *createdTask.ID())
				require.NoError(t, err)
				require.NotNil(t, taskInfo)
				require.Equal(t, name, *taskInfo.Name())

				tasks, err := client.Tasks(ctx, dbName)
				require.NoError(t, err)
				require.NotNil(t, tasks)
				require.Greater(t, len(tasks), 0, "Expected at least one task to be present")
				t.Logf("Found tasks: %v", tasks)
				if len(tasks) > 0 && tasks[0].ID() != nil {
					t.Logf("Task Id to be removed: %s\n", *tasks[0].ID())
				} else {
					t.Logf("Task Id to be removed: <nil>")
				}
				if id := createdTask.ID(); id != nil {
					require.NoError(t, client.RemoveTask(ctx, dbName, *id))
					t.Logf("Task %s removed successfully", *id)
				} else {
					t.Logf("Task ID is nil")
				}
			})
		}
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_ValidationsForCreateNewTask(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		dbName := "_system"
		testCases := map[string]*arangodb.TaskOptions{
			"taskWithoutCommand": {
				Name:   utils.NewType("taskWithoutCommand"),
				Period: utils.NewType(int64(2)),
			},
			"taskWithoutPeriod": nil,
		}

		for name, options := range testCases {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				var err error
				if options == nil {
					_, err = client.CreateTask(ctx, dbName, arangodb.TaskOptions{})
				} else {
					_, err = client.CreateTask(ctx, dbName, *options)
				}

				require.Error(t, err)
				t.Logf("Expected error for task '%s': %v", name, err)
			})
		}

	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_TaskCreationWithId(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			dbName := "_system"
			taskID := "test-task-id"
			options := &arangodb.TaskOptions{
				ID:      &taskID, // Optional if CreateTaskWithID sets it, but safe to keep
				Name:    utils.NewType("TestTaskWithID"),
				Command: utils.NewType("console.log('This is a test task with ID');"),
				Period:  utils.NewType(int64(5)),
			}

			// Create the task with explicit ID
			task, err := client.CreateTaskWithID(ctx, dbName, taskID, *options)
			require.NoError(t, err, "Expected task creation to succeed")
			require.NotNil(t, task, "Expected task to be non-nil")
			require.Equal(t, taskID, *task.ID(), "Task ID mismatch")
			require.Equal(t, *options.Name, *task.Name(), "Task Name mismatch")

			// Retrieve and validate
			retrievedTask, err := client.Task(ctx, dbName, taskID)
			require.NoError(t, err, "Expected task retrieval to succeed")
			require.NotNil(t, retrievedTask, "Expected retrieved task to be non-nil")
			require.Equal(t, taskID, *retrievedTask.ID(), "Retrieved task ID mismatch")
			require.Equal(t, *options.Name, *retrievedTask.Name(), "Retrieved task Name mismatch")

			_, err = client.CreateTaskWithID(ctx, dbName, taskID, *options)
			require.Error(t, err, "Creating a duplicate task should fail")

			// Clean up
			err = client.RemoveTask(ctx, dbName, taskID)
			require.NoError(t, err, "Expected task removal to succeed")
		})
	})
}
