//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

// +build !auth

// This example demonstrates how to create a single document.
package driver_test

import (
	"fmt"
	"log"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

type Book struct {
	Title   string `json:"title"`
	NoPages int    `json:"no_pages"`
}

func Example_createDocument() {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})

	// Create database
	db, err := c.CreateDatabase(nil, "examples_books", nil)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Create collection
	col, err := db.CreateCollection(nil, "books", nil)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	// Create document
	book := Book{
		Title:   "ArangoDB Cookbook",
		NoPages: 257,
	}
	meta, err := col.CreateDocument(nil, book)
	if err != nil {
		log.Fatalf("Failed to create document: %v", err)
	}
	fmt.Printf("Created document in collection '%s' in database '%s'\n", col.Name(), db.Name())

	// Read the document back
	var result Book
	if _, err := col.ReadDocument(nil, meta.Key, &result); err != nil {
		log.Fatalf("Failed to read document: %v", err)
	}
	fmt.Printf("Read book '%+v'\n", result)

	// Output:
	// Created document in collection 'books' in database 'examples_books'
	// Read book '{Title:ArangoDB Cookbook NoPages:257}'
}
