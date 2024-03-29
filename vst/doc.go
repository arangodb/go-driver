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

/*
Package vst implements driver.Connection using a VelocyStream connection.

This connection uses VelocyStream (with optional TLS) to connect to the ArangoDB database.
It encodes its contents as Velocypack.

# Creating an Insecure Connection

To create a VST connection, use code like this.

	// Create a VST connection to the database
	conn, err := vst.NewConnection(vst.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		// Handle error
	}

The resulting connection is used to create a client which you will use
for normal database requests.

	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		// Handle error
	}

# Creating a Secure Connection

To create a secure VST connection, use code like this.

	// Create a VST over TLS connection to the database
	conn, err := vst.NewConnection(vst.ConnectionConfig{
		Endpoints: []string{"https://localhost:8529"},
		TLSConfig: &tls.Config{
			InsecureSkipVerify: trueWhenUsingNonPublicCertificates,
		},
	})
	if err != nil {
		// Handle error
	}
*/
package vst
