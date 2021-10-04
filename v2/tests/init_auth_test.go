//
// DISCLAIMER
//
// Copyright 2020-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
// Author Tomasz Mielech
//

package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/arangodb/go-driver/v2/connection"
	"github.com/stretchr/testify/require"
)

func createAuthenticationFromEnv(t testing.TB, conn connection.Connection) connection.Connection {
	authSpec := os.Getenv("TEST_AUTHENTICATION")
	if authSpec == "" {
		return conn
	}
	parts := strings.Split(authSpec, ":")
	switch parts[0] {
	case "basic":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for basic authentication")
		}
		auth := connection.NewBasicAuth(parts[1], parts[2])

		require.NoError(t, conn.SetAuthentication(auth))

		return conn
	case "jwt":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for jwt authentication")
		}
		return connection.NewJWTAuthWrapper(parts[1], parts[2])(conn)
	//case "super":
	//	if len(parts) != 2 {
	//		t.Fatalf("Expected 'super' and jwt secret")
	//	}
	//	header, err := jwt.CreateArangodJwtAuthorizationHeader(parts[1], "arangodb")
	//	if err != nil {
	//		t.Fatalf("Could not create JWT authentication header: %s", describe(err))
	//	}
	//	return driver.RawAuthentication(header)
	default:
		t.Fatalf("Unknown authentication: '%s'", parts[0])
		return nil
	}
}
