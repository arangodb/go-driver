package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

type TaskParams struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Test_TaskCreation(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		testCases := map[string]*arangodb.TaskOptions{
			"TestDataForTask": {
				Name:    "TestDataForTask",
				Command: "(function(params) { require('@arangodb').print(params); })(params)",
				Period:  2,
				Params: map[string]interface{}{
					"test": "hello",
				},
			},
			"TestDataForCreateTask": {
				Name:    "TestDataForCreateTask",
				Command: "(function() { require('@arangodb').print(Hello); })()",
				Period:  2,
			},
		}

		for name, options := range testCases {
			withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
				createtdTask, err := client.CreateTask(ctx, options)
				require.NoError(t, err)
				require.NotNil(t, createtdTask)
				require.Equal(t, name, createtdTask.Name())

				taskInfo, err := client.Task(ctx, createtdTask.ID())
				require.NoError(t, err)
				require.NotNil(t, taskInfo)
				require.Equal(t, name, taskInfo.Name())

				tasks, err := client.Tasks(ctx)
				require.NoError(t, err)
				require.NotNil(t, tasks)
				require.Greater(t, len(tasks), 0, "Expected at least one task to be present")
				t.Logf("Found tasks: %v", tasks)
				fmt.Printf("Number of tasks: %s\n", tasks[0].ID())

				require.NoError(t, client.RemoveTask(ctx, createtdTask.ID()))
				t.Logf("Task %s removed successfully", createtdTask.ID())
			})
		}
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_TaskCreationWithId(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			taskID := "test-task-id"
			options := &arangodb.TaskOptions{
				ID:      taskID, // Optional if CreateTaskWithID sets it, but safe to keep
				Name:    "TestTaskWithID",
				Command: "console.log('This is a test task with ID');",
				Period:  5,
			}

			// Create the task with explicit ID
			task, err := client.CreateTaskWithID(ctx, taskID, options)
			require.NoError(t, err, "Expected task creation to succeed")
			require.NotNil(t, task, "Expected task to be non-nil")
			require.Equal(t, taskID, task.ID(), "Task ID mismatch")
			require.Equal(t, options.Name, task.Name(), "Task Name mismatch")

			// Retrieve and validate
			retrievedTask, err := client.Task(ctx, taskID)
			require.NoError(t, err, "Expected task retrieval to succeed")
			require.NotNil(t, retrievedTask, "Expected retrieved task to be non-nil")
			require.Equal(t, taskID, retrievedTask.ID(), "Retrieved task ID mismatch")
			require.Equal(t, options.Name, retrievedTask.Name(), "Retrieved task Name mismatch")
			// Try to create task again with same ID — expect 429
			_, err = client.CreateTaskWithID(ctx, taskID, options)
			require.Error(t, err, "Creating a duplicate task should fail")

			// Clean up
			err = client.RemoveTask(ctx, taskID)
			require.NoError(t, err, "Expected task removal to succeed")
		})
	})
}
