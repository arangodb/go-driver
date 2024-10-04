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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/utils"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/arangodb/shared"
)

func Test_CreateBackupSimple(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipNoEnterprise(client, ctx, t)

			backup, err := client.BackupCreate(ctx, nil)
			require.NoError(t, err, "CreateBackup failed")
			require.NotNil(t, backup, "CreateBackup did not return a backup")

			t.Run("Create Backup with options", func(t *testing.T) {
				opts := &arangodb.BackupCreateOptions{
					Label: "test",
				}

				backupWithOpts, err := client.BackupCreate(ctx, opts)
				require.NoError(t, err, "CreateBackup failed")
				require.NotNil(t, backupWithOpts, "CreateBackup did not return a backup")
				require.True(t, strings.HasSuffix(backupWithOpts.ID, "test"))

				defer func() {
					err = client.BackupDelete(ctx, backupWithOpts.ID)
					require.NoError(t, err, "DeleteBackup failed")
				}()
			})

			backups, err := client.BackupList(ctx, nil)
			require.NoError(t, err, "BackupList failed")
			require.NotNil(t, backups, "BackupList did not return a list of backups")
			require.Contains(t, backups.Backups, backup.ID, "BackupList did not return the created backup")

			t.Run("List with single", func(t *testing.T) {
				opt := &arangodb.BackupListOptions{
					ID: backup.ID,
				}
				backupsWithOpts, err := client.BackupList(ctx, opt)
				require.NoError(t, err, "BackupList failed")
				require.NotNil(t, backupsWithOpts, "BackupList did not return a list of backups")
				require.Contains(t, backupsWithOpts.Backups, backup.ID, "BackupList did not return the created backup")
				require.Len(t, backupsWithOpts.Backups, 1, "BackupList did not return the correct number of backups")

			})

			backupMeta := backups.Backups[backup.ID]
			require.Greater(t, backupMeta.NumberOfFiles, uint(0))
			require.Greater(t, backupMeta.NumberOfDBServers, uint(0))
			require.Greater(t, backupMeta.SizeInBytes, uint64(0))

			err = client.BackupDelete(ctx, backup.ID)
			require.NoError(t, err, "DeleteBackup failed")
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_RestoreBackupSimple(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {
			WithCollection(t, db, nil, func(col arangodb.Collection) {
				withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
					skipNoEnterprise(client, ctx, t)

					book1 := DocWithRev{
						Name: "Hello World",
					}

					meta1, err := col.CreateDocument(ctx, book1)
					require.NoError(t, err)

					backup, err := client.BackupCreate(ctx, nil)
					require.NoError(t, err, "CreateBackup failed")
					require.NotNil(t, backup, "CreateBackup did not return a backup")

					book2 := DocWithRev{
						Name: "How to Backups",
					}
					meta2, err := col.CreateDocument(ctx, book2)
					require.NoError(t, err)

					resp, err := client.BackupRestore(ctx, backup.ID)
					require.NoError(t, err, "RestoreBackup failed")
					require.NotNil(t, resp, "RestoreBackup did not return a task")
					require.NotEmpty(t, resp.Previous)

					time.Sleep(15 * time.Second)
					WaitForHealthyCluster(t, client, time.Minute, true)

					exist, err := col.DocumentExists(ctx, meta1.Key)
					require.NoError(t, err)
					require.True(t, exist)

					exist, err = col.DocumentExists(ctx, meta2.Key)
					require.NoError(t, err)
					require.False(t, exist)

					err = client.BackupDelete(ctx, backup.ID)
					require.NoError(t, err, "DeleteBackup failed")

					waitForSync(t, ctx, client)
				})
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

/*
Sometimes after restore, we observe the following error during db creation:

```
Could not create database: executing createSystemCollectionsAndIndices
(creates all system collections including their indices) failed.
```

Looks like not all DB servers have reported ready in Current/DBServers in the agency when the database creation attempt runs.
This could be the case if DB servers are all restarted (as after a hotbackup) and start reacting to liveliness probes each,
but the coordinator has not yet fetched the latest agency state.
There may be a small window of time in which a DB server already responds to `/_admin/server/availablity`,
but has not reported ready to the agency, or the coordinator has not yet fetched the latest state from the agency and
does not yet see the server available in Current/DBServers.
*/
func waitForSync(t *testing.T, ctx context.Context, client arangodb.Client) {
	NewTimeout(func() error {
		name := GenerateUUID("test-backup-DB")

		db, err := client.CreateDatabase(ctx, name, nil)
		if err != nil {
			if ok, arangoErr := shared.IsArangoError(err); ok {
				t.Logf("waitForSync ERROR: errorNum: %d, errCode: %d, msg: %s", arangoErr.ErrorNum, arangoErr.Code, arangoErr.ErrorMessage)
				if strings.Contains(arangoErr.ErrorMessage, "executing createSystemCollectionsAndIndices (creates all system collections including their indices) failed") {
					return err
				}
			}
		}
		require.NoError(t, err, fmt.Sprintf("waitForSync Failed to create DB %s", name))
		require.NoError(t, db.Remove(ctx))
		return Interrupt{}
	}).TimeoutT(t, 2*time.Minute, 125*time.Millisecond)
}

func Test_BackupFullFlow(t *testing.T) {
	requireClusterMode(t)

	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipNoEnterprise(client, ctx, t)

			repo, config := getTransferConfigFromEnv(t)

			backup, err := client.BackupCreate(ctx, nil)
			require.NoError(t, err, "CreateBackup failed")
			require.NotNil(t, backup, "CreateBackup did not return a backup")

			t.Run("Upload Backup", func(t *testing.T) {
				tf, err := client.BackupUpload(ctx, backup.ID, repo, config)
				require.NoError(t, err, "UploadBackup failed")
				require.NotNil(t, tf)

				waitForTransferJobCompletion(t, ctx, tf, false, false)
			})

			t.Run("Delete Backup", func(t *testing.T) {
				err = client.BackupDelete(ctx, backup.ID)
				require.NoError(t, err, "DeleteBackup failed")
			})

			t.Run("Download Backup", func(t *testing.T) {
				tf, err := client.BackupDownload(ctx, backup.ID, repo, config)
				require.NoError(t, err, "DownloadBackup failed")
				require.NotNil(t, tf)

				waitForTransferJobCompletion(t, ctx, tf, false, false)
			})

			t.Run("Restore Backup", func(t *testing.T) {
				resp, err := client.BackupRestore(ctx, backup.ID)
				require.NoError(t, err, "RestoreBackup failed")
				require.NotNil(t, resp, "RestoreBackup did not return a task")
				require.NotEmpty(t, resp.Previous)

				time.Sleep(15 * time.Second)
				WaitForHealthyCluster(t, client, time.Minute, true)
			})

			waitForSync(t, ctx, client)
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func Test_UploadBackupFailAndAbort(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		withContextT(t, defaultTestTimeout, func(ctx context.Context, tb testing.TB) {
			skipNoEnterprise(client, ctx, t)

			t.Run("Upload Backup not existing backup", func(t *testing.T) {
				repo, config := getTransferConfigFromEnv(t)

				tf, err := client.BackupUpload(ctx, "not_there", repo, config)
				require.NoError(t, err, "UploadBackup failed")
				require.NotNil(t, tf)

				waitForTransferJobCompletion(t, ctx, tf, true, false)
			})

			t.Run("Upload Backup aborted", func(t *testing.T) {
				repo, config := getTransferConfigFromEnv(t)

				backup, err := client.BackupCreate(ctx, nil)
				require.NoError(t, err, "CreateBackup failed")
				require.NotNil(t, backup, "CreateBackup did not return a backup")

				tf, err := client.BackupUpload(ctx, backup.ID, repo, config)
				require.NoError(t, err, "UploadBackup failed")
				require.NotNil(t, tf)

				require.NoError(t, tf.Abort(ctx))

				waitForTransferJobCompletion(t, ctx, tf, false, true)
			})
		})
	}, WrapOptions{
		Parallel: utils.NewType(false),
	})
}

func waitForTransferJobCompletion(t *testing.T, ctx context.Context, tf arangodb.TransferMonitor, shouldFail, shouldAbort bool) {
	for {
		progress, err := tf.Progress(ctx)
		require.NoError(t, err)

		// Wait for completion
		completedCount := 0
		for dbServer, status := range progress.DBServers {
			switch status.Status {
			case arangodb.TransferCompleted:
				if !shouldFail && !shouldAbort {
					completedCount++
					break
				} else {
					t.Fatalf("Upload should not complete: %s", dbServer)
				}
			case arangodb.TransferFailed:
				if shouldFail {
					completedCount++
					break
				} else {
					t.Fatalf("Job on %s failed: %s (%d)", dbServer, status.ErrorMessage, status.Error)
				}
			case arangodb.TransferCancelled:
				if shouldAbort {
					completedCount++
					break
				}
				t.Fatalf("Job on %s was cancelled", dbServer)
			}

			t.Logf("Status on %s: %s (%d / %d)", dbServer, status.Status, status.Progress.Done, status.Progress.Total)
		}

		if completedCount == len(progress.DBServers) {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Job failed: %s", ctx.Err())
		case <-time.After(8 * time.Second):
			break
		}
	}
}

func getTransferConfigFromEnv(t *testing.T) (repo string, config map[string]json.RawMessage) {
	repoEnv := os.Getenv("TEST_BACKUP_REMOTE_REPO")
	confEnv := os.Getenv("TEST_BACKUP_REMOTE_CONFIG")

	if repoEnv == "" || confEnv == "" {
		t.Skipf("TEST_BACKUP_REMOTE_REPO and TEST_BACKUP_REMOTE_CONFIG must be set for remote transfer tests")
	}

	var confMap map[string]json.RawMessage
	err := json.Unmarshal([]byte(confEnv), &confMap)
	require.NoError(t, err, "Failed to unmarshal remote config")

	require.NotEmpty(t, repoEnv)
	require.NotEmpty(t, confMap)

	return repoEnv, confMap
}
