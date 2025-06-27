//
// DISCLAIMER
//
// Copyright 2018-2023 ArangoDB GmbH, Cologne, Germany
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

package test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"sync"
	"testing"

	driver "github.com/arangodb/go-driver"
)

// TestConcurrentCreateSmallDocuments make a lot of concurrent CreateDocument calls.
// It then verifies that all documents "have arrived".
func TestConcurrentCreateSmallDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip on short tests")
	}

	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}

	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})

	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv33p := version.Version.CompareTo("3.3") >= 0
	if !isv33p && os.Getenv("TEST_CONNECTION") == "vst" {
		t.Skip("Skipping VST load test on 3.2")
	} else {
		db := ensureDatabase(nil, c, "document_test", nil, t)
		defer func() {
			err := db.Remove(nil)
			if err != nil {
				t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
			}
		}()
		col := ensureCollection(nil, db, "TestConcurrentCreateSmallDocuments", nil, t)

		docChan := make(chan driver.DocumentMeta, 128*1024)

		creator := func(limit, interval int) {
			for i := 0; i < limit; i++ {
				ctx := context.Background()
				doc := UserDoc{
					"Jan",
					i * interval,
				}
				meta, err := col.CreateDocument(ctx, doc)
				if err != nil {
					t.Fatalf("Failed to create new document: %s", describe(err))
				}
				docChan <- meta
			}
		}

		reader := func() {
			for {
				meta, ok := <-docChan
				if !ok {
					return
				}
				// Document must exists now
				if found, err := col.DocumentExists(nil, meta.Key); err != nil {
					t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
				} else if !found {
					t.Errorf("DocumentExists returned false for '%s', expected true", meta.Key)
				}
				// Read document
				var readDoc UserDoc
				if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
					t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
				}
			}
		}

		noCreators := getIntFromEnv("NOCREATORS", 25)
		noReaders := getIntFromEnv("NOREADERS", 50)
		noDocuments := getIntFromEnv("NODOCUMENTS", 1000) // per creator

		wgCreators := sync.WaitGroup{}
		// Run N concurrent creators
		for i := 0; i < noCreators; i++ {
			wgCreators.Add(1)
			go func() {
				defer wgCreators.Done()
				creator(noDocuments, noCreators)
			}()
		}
		wgReaders := sync.WaitGroup{}
		// Run M readers
		for i := 0; i < noReaders; i++ {
			wgReaders.Add(1)
			go func() {
				defer wgReaders.Done()
				reader()
			}()
		}
		wgCreators.Wait()
		close(docChan)
		wgReaders.Wait()
	}
}

// TestConcurrentCreateBigDocuments make a lot of concurrent CreateDocument calls.
// It then verifies that all documents "have arrived".
func TestConcurrentCreateBigDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip on short tests")
	}

	// Disable those tests for active failover
	if getTestMode() == testModeResilientSingle {
		t.Skip("Disabled in active failover mode")
	}

	// don't use disallowUnknownFields in this test - we have here custom structs defined
	c := createClient(t, &testsClientConfig{skipDisallowUnknownFields: true})

	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv33p := version.Version.CompareTo("3.3") >= 0
	if !isv33p && os.Getenv("TEST_CONNECTION") == "vst" {
		t.Skip("Skipping VST load test on 3.2")
	} else {
		db := ensureDatabase(nil, c, "document_test", nil, t)
		defer func() {
			err := db.Remove(nil)
			if err != nil {
				t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
			}
		}()
		col := ensureCollection(nil, db, "TestConcurrentCreateBigDocuments", nil, t)

		docChan := make(chan driver.DocumentMeta, 16*1024)

		creator := func(limit, interval int) {
			data := make([]byte, 1024)
			for i := 0; i < limit; i++ {
				rand.Read(data)
				ctx := context.Background()
				doc := UserDoc{
					"Jan" + strconv.Itoa(i) + hex.EncodeToString(data),
					i * interval,
				}
				meta, err := col.CreateDocument(ctx, doc)
				if err != nil {
					t.Fatalf("Failed to create new document: %s", describe(err))
				}
				docChan <- meta
			}
		}

		reader := func() {
			for {
				meta, ok := <-docChan
				if !ok {
					return
				}
				// Document must exists now
				if found, err := col.DocumentExists(nil, meta.Key); err != nil {
					t.Fatalf("DocumentExists failed for '%s': %s", meta.Key, describe(err))
				} else if !found {
					t.Errorf("DocumentExists returned false for '%s', expected true", meta.Key)
				}
				// Read document
				var readDoc UserDoc
				if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
					t.Fatalf("Failed to read document '%s': %s", meta.Key, describe(err))
				}
			}
		}

		noCreators := getIntFromEnv("NOCREATORS", 25)
		noReaders := getIntFromEnv("NOREADERS", 50)
		noDocuments := getIntFromEnv("NODOCUMENTS", 100) // per creator

		wgCreators := sync.WaitGroup{}
		// Run N concurrent creators
		for i := 0; i < noCreators; i++ {
			wgCreators.Add(1)
			go func() {
				defer wgCreators.Done()
				creator(noDocuments, noCreators)
			}()
		}
		wgReaders := sync.WaitGroup{}
		// Run M readers
		for i := 0; i < noReaders; i++ {
			wgReaders.Add(1)
			go func() {
				defer wgReaders.Done()
				reader()
			}()
		}
		wgCreators.Wait()
		close(docChan)
		wgReaders.Wait()
	}
}
