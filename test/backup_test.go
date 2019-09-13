//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
)

var backupAPIAvailable *bool

func setBackupAvailable(av bool) {
	backupAPIAvailable = &av
}

func skipIfNoBackup(c driver.Client, t *testing.T) {
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}
	con := c.Connection()

	if backupAPIAvailable == nil {

		t.Log("Checking for backup api")

		req, err := con.NewRequest("POST", "_admin/backup/list")
		if err != nil {
			t.Fatalf("Failed to send test request: %s", describe(err))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		resp, err := con.Do(ctx, req)
		if err != nil {
			if !driver.IsTimeout(err) {
				t.Fatalf("Test request failed: %s", describe(err))
			}
		} else {
			switch resp.StatusCode() {
			case 404:
				setBackupAvailable(false)
			case 200:
				setBackupAvailable(true)
				return
			default:
				t.Fatalf("Test request failed with unexpected error code: %d", resp.StatusCode())
			}
		}

	} else {
		if *backupAPIAvailable {
			return
		}
	}

	t.Skip("Backup API not available")
}

func getTransfereConfigFromEnv(t *testing.T) (repo string, config map[string]json.RawMessage) {

	repoenv := os.Getenv("TEST_BACKUP_REMOTE_REPO")
	confenv := os.Getenv("TEST_BACKUP_REMOTE_CONFIG")

	if repoenv == "" || confenv == "" {
		t.Skipf("TEST_BACKUP_REMOTE_REPO and TEST_BACKUP_REMOTE_CONFIG must be set for remote transfere tests")
	}

	var confMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(confenv), &confMap); err != nil {
		t.Fatalf("Failed to unmarshal remote config: %s %s", describe(err), confenv)
	}

	return repoenv, confMap
}

func ensureBackup(ctx context.Context, b driver.ClientBackup, t *testing.T) driver.BackupID {
	if id, _, err := b.Create(ctx, nil); err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
		return ""
	} else {
		return id
	}
}

func hasBackup(ctx context.Context, id driver.BackupID, b driver.ClientBackup, t *testing.T) bool {
	if list, err := b.List(ctx, &driver.BackupListOptions{ID: id}); err != nil {
		if driver.IsNotFound(err) {
			return false
		}

		t.Fatalf("Unexpected error: %s", describe(err))
	} else {
		if meta, ok := list[id]; ok {
			if meta.ID == id {
				return true
			}
			t.Fatalf("meta.ID is different: %s, expected %s", meta.ID, id)
		} else {
			t.Fatalf("List does not contain the backup")
		}
	}
	// Not reached
	return false
}

func TestBackupCreate(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()

	id, _, err := b.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	t.Logf("Created backup %s", id)
}

func TestBackupCreateWithLabel(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()

	label := "test_label"

	id, _, err := b.Create(ctx, &driver.BackupCreateOptions{Label: label})
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	// Check if id is suffixed with _test_label
	if !strings.HasSuffix(string(id), label) {
		t.Fatalf("BackupID is not suffixed with label")
	}
}

func TestBackupCreateWithForce(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
	}()
	b := c.Backup()
	dbname := "backup_db_test"
	colname := "backup_query_target"

	// First create a long running transaction
	db := ensureDatabase(nil, c, dbname, nil, t)
	ensureCollection(nil, db, colname, nil, t)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := db.Query(nil, fmt.Sprintf("FOR i IN 1..10 INSERT {s:SLEEP(1)} INTO %s", colname), nil); err != nil {
			t.Fatalf("Failed to run query: %s", describe(err))
		}
	}()

	time.Sleep(time.Second)

	_, _, err := b.Create(nil, &driver.BackupCreateOptions{AllowInconsistent: false, Timeout: time.Second})
	if err == nil {
		t.Fatalf("Creating backup should fail but did not!")
	}

	_, resp, err := b.Create(nil, &driver.BackupCreateOptions{AllowInconsistent: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	if !resp.PotentiallyInconsistent {
		t.Error("Expected PotentiallyInconsistent to be set to true, but it is not")
	}
}

func TestBackupListWithID(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()
	id := ensureBackup(ctx, b, t)

	var raw []byte
	ctx = driver.WithRawResponse(ctx, &raw)

	// check if the id is present
	if list, err := b.List(ctx, &driver.BackupListOptions{ID: id}); err != nil {
		t.Fatalf("Failed to list backups: %s", describe(err))
	} else {
		t.Logf("Response: %s", string(raw))

		found := false
		for backup := range list {
			if backup == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Backup %s was created but is not listed", id)
		}
	}
}

func TestBackupListWithNonExistingID(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()

	var raw []byte
	ctx = driver.WithRawResponse(ctx, &raw)

	// check if the id is present
	if _, err := b.List(ctx, &driver.BackupListOptions{ID: "this_id_does_not_exist"}); err != nil {
		if !driver.IsNotFound(err) {
			t.Errorf("Unexpected error: %s", describe(err))
		}
	} else {
		t.Errorf("List did not fail")
	}
}

func TestBackupList(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()
	id := ensureBackup(ctx, b, t)

	var raw []byte
	ctx = driver.WithRawResponse(ctx, &raw)

	// check if the id is present
	if list, err := b.List(ctx, &driver.BackupListOptions{ID: id}); err != nil {
		t.Fatalf("Failed to list backups: %s", describe(err))
	} else {
		t.Logf("Response: %s", string(raw))

		found := false
		version, err := c.Version(ctx)
		if err != nil {
			t.Fatalf("Failed to get server version: %s", describe(err))
		}
		for backup, meta := range list {
			t.Logf("Found backup %s", backup)
			if backup == id {
				found = true
			}
			if meta.Version != string(version.Version) {
				t.Errorf("Different version string in backup: %s, actual version: %s", meta.Version, version.String())
			}
		}

		if !found {
			t.Errorf("Backup %s was created but not listed", id)
		}
	}
}

func TestBackupDelete(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()
	id := ensureBackup(ctx, b, t)

	if !hasBackup(ctx, id, b, t) {
		t.Fatalf("Backup was not created: %s", id)
	}

	if err := b.Delete(ctx, id); err != nil {
		t.Errorf("Failed to delete backup: %s", describe(err))
	}

	if hasBackup(ctx, id, b, t) {
		t.Errorf("Backup was not delete: %s", id)
	}
}

func TestBackupDeleteNonExisting(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()

	if err := b.Delete(ctx, "does_not_exist"); err != nil {
		if !driver.IsNotFound(err) {
			t.Errorf("Unexpected error: %s", describe(err))
		}
	} else {
		t.Errorf("Expected NotFound error")
	}
}

func waitForServerRestart(ctx context.Context, c driver.Client, t *testing.T) {

	serverWasDown := false

	for {
		vctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		if _, err := c.Version(vctx); err != nil {
			if driver.IsTimeout(err) {
				serverWasDown = true
			}
		} else {
			if serverWasDown {
				cancel()
				return
			}
		}

		cancel()
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			break
		}
	}

}

func TestBackupRestore(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	ctx := context.Background()
	b := c.Backup()

	isSingle := false
	if role, err := c.ServerRole(ctx); err != nil {
		t.Fatalf("Failed to obtain server role: %s", describe(err))
	} else {
		isSingle = role == driver.ServerRoleSingle
	}

	dbname := "backup"
	colname := "col"

	db := ensureDatabase(ctx, c, dbname, nil, t)
	col := ensureCollection(ctx, db, colname, nil, t)

	// Write a document
	book1 := Book{
		Title: "Hello World",
	}

	meta1, err := col.CreateDocument(ctx, book1)
	if err != nil {
		t.Fatalf("Failed to create document %s", describe(err))
	}

	id, _, err := b.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	// Insert another document
	book2 := Book{
		Title: "How to Backups",
	}

	meta2, err := col.CreateDocument(ctx, book2)
	if err != nil {
		t.Fatalf("Failed to create document %s", describe(err))
	}

	// Now restore
	if err := b.Restore(ctx, id, nil); err != nil {
		t.Fatalf("Failed to restore backup: %s", describe(err))
	}

	if isSingle {
		waitctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		waitForServerRestart(waitctx, c, t)
	}

	if ok, err := col.DocumentExists(ctx, meta1.Key); err != nil {
		t.Errorf("Failed to lookup document: %s", describe(err))
	} else if !ok {
		t.Errorf("Document missing: %s", meta1.Key)
	}

	if ok, err := col.DocumentExists(ctx, meta2.Key); err != nil {
		t.Errorf("Failed to lookup document: %s", describe(err))
	} else if ok {
		t.Errorf("Document should not be there: %s", meta2.Key)
	}
}

func TestBackupUploadNonExisting(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	skipNoEnterprise(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	b := c.Backup()
	repo, conf := getTransfereConfigFromEnv(t)

	jobID, err := b.Upload(ctx, "not_there", repo, conf)
	if err != nil {
		t.Errorf("Starting upload failed: %s", describe(err))
	}

	for {
		progress, err := b.Progress(ctx, jobID)
		if err != nil {
			t.Fatalf("Progress failed: %s", describe(err))
		}

		// Wait for completion
		completedCount := 0
		for dbserver, status := range progress.DBServers {
			switch status.Status {
			case driver.TransferCompleted:
				t.Fatalf("Upload should not complete: %s", dbserver)
			case driver.TransferFailed:
				completedCount++
			}
			t.Logf("Status on %s: %s", dbserver, status.Status)
		}

		if completedCount == len(progress.DBServers) {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Upload failed: %s", describe(ctx.Err()))
		case <-time.After(5 * time.Second):
			break
		}
	}
}

func waitForTransfereJobCompletion(ctx context.Context, jobID driver.BackupTransferJobID, b driver.ClientBackup, t *testing.T) {
	t.Logf("Waiting for completion of %s", jobID)

	for {
		progress, err := b.Progress(ctx, jobID)
		if err != nil {
			t.Errorf("Progress failed: %s", describe(err))
		}

		// Wait for completion
		completedCount := 0
		for dbserver, status := range progress.DBServers {
			switch status.Status {
			case driver.TransferCompleted:
				completedCount++
				break
			case driver.TransferFailed:
				t.Fatalf("Job %s on %s failed: %s (%d)", jobID, dbserver, status.ErrorMessage, status.Error)
			}

			t.Logf("Status on %s: %s (%d / %d)", dbserver, status.Status, status.Progress.Done, status.Progress.Total)
		}

		if completedCount == len(progress.DBServers) {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Job %s failed: %s", jobID, describe(ctx.Err()))
		case <-time.After(8 * time.Second):
			break
		}
	}
}

func uploadBackupWaitForCompletion(ctx context.Context, id driver.BackupID, b driver.ClientBackup, t *testing.T) {
	repo, conf := getTransfereConfigFromEnv(t)

	jobID, err := b.Upload(ctx, id, repo, conf)
	if err != nil {
		t.Fatalf("Failed to trigger upload: %s", describe(err))
	}

	defer func() {
		b.Abort(ctx, jobID)
	}()

	waitForTransfereJobCompletion(ctx, jobID, b, t)
}

func TestBackupUpload(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	skipNoEnterprise(t)
	getTransfereConfigFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	b := c.Backup()
	id := ensureBackup(ctx, b, t)
	uploadBackupWaitForCompletion(ctx, id, b, t)
}

func TestBackupUploadAbort(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	skipNoEnterprise(t)
	repo, conf := getTransfereConfigFromEnv(t)
	ctx := context.Background()
	b := c.Backup()
	id := ensureBackup(ctx, b, t)

	jobID, err := b.Upload(ctx, id, repo, conf)
	if err != nil {
		t.Fatalf("Failed to start upload: %s", describe(err))
	}

	if err := b.Abort(ctx, jobID); err != nil {
		t.Fatalf("Failed to abort upload: %s", describe(err))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {

		if progress, err := b.Progress(ctx, jobID); err != nil {
			t.Errorf("Unexpected error: %s", describe(err))
		} else if progress.Cancelled {

			cancelledCount := 0

			for _, detail := range progress.DBServers {
				if detail.Status == driver.TransferCancelled {
					cancelledCount++
				}
			}

			if cancelledCount == len(progress.DBServers) {
				break
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Progress was not cancelled: %s", ctx.Err())
		case <-time.After(time.Second):
			break
		}
	}
}

func TestBackupCompleteCycle(t *testing.T) {
	skipNoEnterprise(t)
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)
	repo, conf := getTransfereConfigFromEnv(t)

	ctx := context.Background()
	b := c.Backup()

	dbname := "backup"
	colname := "col"

	db := ensureDatabase(ctx, c, dbname, nil, t)
	col := ensureCollection(ctx, db, colname, nil, t)

	isSingle := false
	if role, err := c.ServerRole(ctx); err != nil {
		t.Fatalf("Failed to obtain server role: %s", describe(err))
	} else {
		isSingle = role == driver.ServerRoleSingle
	}

	// Write a document
	book1 := Book{
		Title: "Hello World",
	}

	meta1, err := col.CreateDocument(ctx, book1)
	if err != nil {
		t.Fatalf("Failed to create document %s", describe(err))
	}

	id, _, err := b.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	// start upload
	uploadID, err := b.Upload(ctx, id, repo, conf)
	if err != nil {
		t.Fatalf("Failed to start upload: %s", describe(err))
	}

	// Insert another document
	book2 := Book{
		Title: "How to Backups",
	}

	meta2, err := col.CreateDocument(ctx, book2)
	if err != nil {
		t.Fatalf("Failed to create document %s", describe(err))
	}

	// Wait for upload to be completed
	waitForTransfereJobCompletion(ctx, uploadID, b, t)

	// delete the backup
	if err := b.Delete(ctx, id); err != nil {
		t.Fatalf("Failed to delete backup: %s", describe(err))
	}

	// Trigger a download
	downloadID, err := b.Download(ctx, id, repo, conf)
	if err != nil {
		t.Fatalf("Failed to trigger download: %s", describe(err))
	}

	// Wait for download to be completed
	waitForTransfereJobCompletion(ctx, downloadID, b, t)

	// Now restore
	if err := b.Restore(ctx, id, nil); err != nil {
		t.Fatalf("Failed to restore backup: %s", describe(err))
	}

	if isSingle {
		waitctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		waitForServerRestart(waitctx, c, t)
	}

	if ok, err := col.DocumentExists(ctx, meta1.Key); err != nil {
		t.Errorf("Failed to lookup document: %s", describe(err))
	} else if !ok {
		t.Errorf("Document missing: %s", meta1.Key)
	}

	if ok, err := col.DocumentExists(ctx, meta2.Key); err != nil {
		t.Errorf("Failed to lookup document: %s", describe(err))
	} else if ok {
		t.Errorf("Document should not be there: %s", meta2.Key)
	}
}

type backupResult struct {
	ID    driver.BackupID
	Error error
}

func TestBackupCreateManyBackupsFast(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)

	numTries := 5

	ctx := context.Background()
	b := c.Backup()

	idchan := make(chan backupResult)
	defer close(idchan)
	var wg sync.WaitGroup

	oneWasSuccessful := false

	for i := 0; i < numTries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if id, _, err := b.Create(ctx, nil); err == nil {
				idchan <- backupResult{ID: id}

			} else {
				idchan <- backupResult{Error: err}
			}
		}()
	}

	foundSet := make(map[driver.BackupID]struct{})
	for i := 0; i < numTries; i++ {
		res := <-idchan
		if res.Error != nil {
			t.Logf("Creating Backup failed: %s", describe(res.Error))
			continue
		}
		oneWasSuccessful = true
		if _, ok := foundSet[res.ID]; ok {
			t.Errorf("Duplicate id: %s", res.ID)
		} else {
			t.Logf("Created backup %s", res.ID)
			foundSet[res.ID] = struct{}{}
		}
	}

	if !oneWasSuccessful {
		t.Fatalf("All concurrent create requests failed!")
	}

	wg.Wait()

	list, err := b.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to obtain list of backups: %s", describe(err))
	}

	for id := range foundSet {
		if _, ok := list[id]; !ok {
			t.Errorf("Backup %s not contained in list", id)
		}
	}
}

func TestBackupCreateRestoreParallel(t *testing.T) {
	c := createClientFromEnv(t, true)
	skipIfNoBackup(c, t)

	ctx := context.Background()
	b := c.Backup()
	id := ensureBackup(ctx, b, t)

	isSingle := false
	if role, err := c.ServerRole(ctx); err != nil {
		t.Fatalf("Failed to obtain server role: %s", describe(err))
	} else {
		isSingle = role == driver.ServerRoleSingle
	}

	errchan := make(chan error)
	defer close(errchan)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, _, err := b.Create(ctx, nil); err == nil {
			errchan <- nil
		} else {
			errchan <- err
			t.Logf("Create failed: %s", describe(err))
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := b.Restore(ctx, id, nil); err == nil {
			errchan <- nil
			if isSingle {
				waitctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				waitForServerRestart(waitctx, c, t)
			}
		} else {
			errchan <- err
			t.Logf("Restore failed: %s", describe(err))
		}
	}()

	errCount := 0
	for i := 0; i < 2; i++ {
		err := <-errchan
		if err != nil {
			errCount++
		}
	}

	wg.Wait()

	if errCount >= 2 {
		t.Fatalf("Both operation failed!")
	}
}

func ensureRemoteBackup(ctx context.Context, b driver.ClientBackup, t *testing.T) driver.BackupID {
	id := ensureBackup(ctx, b, t)
	uploadBackupWaitForCompletion(ctx, id, b, t)
	if err := b.Delete(ctx, id); err != nil {
		t.Fatalf("Failed to remove backup: %s", err)
	}
	return id
}
