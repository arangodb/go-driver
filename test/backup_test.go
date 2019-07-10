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
	"os"
	"strings"
	"testing"

	driver "github.com/arangodb/go-driver"
)

func skipIfNoBackup(t *testing.T) {
	if _, set := os.LookupEnv("TEST_ENABLE_BACKUP"); !set {
		t.Skip("Backup Tests not enabled")
	}
}

func ensureBackup(ctx context.Context, b driver.ClientBackup, t *testing.T) driver.BackupID {
	if id, err := b.Create(ctx, nil); err != nil {
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
			} else {
				t.Fatalf("meta.ID is different: %s, expected %s", meta.ID, id)
			}
		} else {
			t.Fatalf("List does not contain the backup")
		}
	}
	// Not reached
	return false
}

func TestBackupCreate(t *testing.T) {
	skipIfNoBackup(t)
	ctx := context.Background()
	b := createClientFromEnv(t, true).Backup()

	id, err := b.Create(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	t.Logf("Created backup %s", id)
}

func TestBackupCreateWithLabel(t *testing.T) {
	skipIfNoBackup(t)
	ctx := context.Background()
	b := createClientFromEnv(t, true).Backup()

	label := "test_label"

	id, err := b.Create(ctx, &driver.BackupCreateOptions{Label: label})
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	// Check if id is suffixed with _test_label
	if !strings.HasSuffix(string(id), label) {
		t.Fatalf("BackupID is not suffixed with label")
	}

	t.Logf("Created backup %s", id)
}

func TestBackupCreateWithForce(t *testing.T) {
	skipIfNoBackup(t)
	ctx := context.Background()
	b := createClientFromEnv(t, true).Backup()

	id, err := b.Create(ctx, &driver.BackupCreateOptions{Force: true})
	if err != nil {
		t.Fatalf("Failed to create backup: %s", describe(err))
	}

	t.Logf("Force created backup %s", id)
}

func TestBackupListWithID(t *testing.T) {
	skipIfNoBackup(t)
	ctx := context.Background()
	b := createClientFromEnv(t, true).Backup()
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
	skipIfNoBackup(t)
	ctx := context.Background()
	b := createClientFromEnv(t, true).Backup()

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
	skipIfNoBackup(t)
	ctx := context.Background()
	c := createClientFromEnv(t, true)
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
	skipIfNoBackup(t)
	ctx := context.Background()
	c := createClientFromEnv(t, true)
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
	skipIfNoBackup(t)
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	b := c.Backup()

	if err := b.Delete(ctx, "does_not_exist"); err != nil {
		if !driver.IsNotFound(err) {
			t.Errorf("Unexpected error: %s", describe(err))
		}
	} else {
		t.Errorf("Expected NotFound error")
	}
}

func TestBackupRestore(t *testing.T) {
	skipIfNoBackup(t)
	ctx := context.Background()
	c := createClientFromEnv(t, true)
	b := c.Backup()

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

	id, err := b.Create(ctx, nil)
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
