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

// +build !auth

// This example demonstrates how to create a graph, how to add vertices and edges and how to delete it again.
package driver_test

import (
	"log"
	"fmt"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

type MyObject struct {
	Name string `json:"_key"`
	Age  int    `json:"age"`
}

type MyEdgeObject struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}

func Example_createGraph() {
	fmt.Println("Hello World")

	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}
	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})

	// Create database
	db, err := c.CreateDatabase(nil, "my_graph_db", nil)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// define the edgeCollection to store the edges
	var edgeDefinition driver.EdgeDefinition
	edgeDefinition.Collection = "myEdgeCollection"
	// define a set of collections where an edge is going out...
	edgeDefinition.From = []string{"myCollection1", "myCollection2"}

	// repeat this for the collections where an edge is going into
	edgeDefinition.To = []string{"myCollection1", "myCollection3"}

	// A graph can contain additional vertex collections, defined in the set of orphan collections
	var options driver.CreateGraphOptions
	options.OrphanVertexCollections = []string{"myCollection4", "myCollection5"}
	options.EdgeDefinitions = []driver.EdgeDefinition{edgeDefinition}

	// now it's possible to create a graph
	graph, err := db.CreateGraph(nil, "myGraph", &options)
	if err != nil {
		log.Fatalf("Failed to create graph: %v", err)
	}

	// add vertex
	vertexCollection1, err := graph.VertexCollection(nil, "myCollection1")
	if err != nil {
		log.Fatalf("Failed to get vertex collection: %v", err)
	}

	myObjects := []MyObject{
		MyObject{
			"Homer",
			38,
		},
		MyObject{
			"Marge",
			36,
		},
	}
	_, _, err = vertexCollection1.CreateDocuments(nil, myObjects)
	if err != nil {
		log.Fatalf("Failed to create vertex documents: %v", err)
	}

	// add edge
	edgeCollection, _, err := graph.EdgeCollection(nil, "myEdgeCollection")
	if err != nil {
		log.Fatalf("Failed to select edge collection: %v", err)
	}

	edge := MyEdgeObject{From: "myCollection1/Homer", To: "myCollection1/Marge"}
	_, err = edgeCollection.CreateDocument(nil, edge)
	if err != nil {
		log.Fatalf("Failed to create edge document: %v", err)
	}

	// delete graph
	graph.Remove(nil)
}
