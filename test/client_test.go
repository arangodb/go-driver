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

package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	httplib "net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "net/http/pprof"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/jwt"
	"github.com/arangodb/go-driver/vst"
	"github.com/arangodb/go-driver/vst/protocol"
)

var (
	logEndpointsOnce   sync.Once
	runPProfServerOnce sync.Once
)

// skipBetweenVersion skips the test if the current server version is less than
// the min version or higher/equal max version
func skipBetweenVersion(c driver.Client, minVersion, maxVersion driver.Version, t *testing.T) driver.VersionInfo {
	x, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", describe(err))
	}
	if x.Version.CompareTo(minVersion) < 0 {
		t.Skipf("Skipping below version '%s', got version '%s'", minVersion, x.Version)
	}
	if x.Version.CompareTo(maxVersion) >= 0 {
		t.Skipf("Skipping above version '%s', got version '%s'", maxVersion, x.Version)
	}
	return x
}

// skipBelowVersion skips the test if the current server version is less than
// the given version.
func skipBelowVersion(c driver.Client, version driver.Version, t *testing.T) driver.VersionInfo {
	x, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Failed to get version info: %s", describe(err))
	}
	if x.Version.CompareTo(version) < 0 {
		t.Skipf("Skipping below version '%s', got version '%s'", version, x.Version)
	}
	return x
}

// getEndpointsFromEnv returns the endpoints specified in the TEST_ENDPOINTS
// environment variable.
func getEndpointsFromEnv(t testEnv) []string {
	eps := strings.Split(os.Getenv("TEST_ENDPOINTS"), ",")
	if len(eps) == 0 {
		t.Fatal("No endpoints found in environment variable TEST_ENDPOINTS")
	}
	return eps
}

// getContentTypeFromEnv returns the content-type specified in the TEST_CONTENT_TYPE
// environment variable (json|vpack).
func getContentTypeFromEnv(t testEnv) driver.ContentType {
	switch ct := os.Getenv("TEST_CONTENT_TYPE"); ct {
	case "vpack":
		return driver.ContentTypeVelocypack
	case "json", "":
		return driver.ContentTypeJSON
	default:
		t.Fatalf("Unknown content type '%s'", ct)
		return 0
	}
}

// createAuthenticationFromEnv initializes an authentication specified in the TEST_AUTHENTICATION
// environment variable.
func createAuthenticationFromEnv(t testEnv) driver.Authentication {
	authSpec := os.Getenv("TEST_AUTHENTICATION")
	if authSpec == "" {
		return nil
	}
	parts := strings.Split(authSpec, ":")
	switch parts[0] {
	case "basic":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for basic authentication")
		}
		return driver.BasicAuthentication(parts[1], parts[2])
	case "jwt":
		if len(parts) != 3 {
			t.Fatalf("Expected username & password for jwt authentication")
		}
		return driver.JWTAuthentication(parts[1], parts[2])
	case "super":
		if len(parts) != 2 {
			t.Fatalf("Expected 'super' and jwt secret")
		}
		header, err := jwt.CreateArangodJwtAuthorizationHeader(parts[1], "arangodb")
		if err != nil {
			t.Fatalf("Could not create JWT authentication header: %s", describe(err))
		}
		return driver.RawAuthentication(header)
	default:
		t.Fatalf("Unknown authentication: '%s'", parts[0])
		return nil
	}
}

// createConnectionFromEnv initializes a Connection from information specified in environment variables.
func createConnectionFromEnv(t testEnv) driver.Connection {
	connSpec := os.Getenv("TEST_CONNECTION")
	connVer := os.Getenv("TEST_CVERSION")
	switch connSpec {
	case "vst":
		var version protocol.Version
		switch connVer {
		case "1.0", "":
			version = protocol.Version1_0
		case "1.1":
			version = protocol.Version1_1
		default:
			t.Fatalf("Unknown connection version '%s'", connVer)
		}
		config := vst.ConnectionConfig{
			Endpoints: getEndpointsFromEnv(t),
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
			Transport: protocol.TransportConfig{
				Version: version,
			},
		}
		conn, err := vst.NewConnection(config)
		if err != nil {
			t.Fatalf("Failed to create new vst connection: %s", describe(err))
		}
		return conn

	case "http", "":
		config := http.ConnectionConfig{
			Endpoints:   getEndpointsFromEnv(t),
			TLSConfig:   &tls.Config{InsecureSkipVerify: true},
			ContentType: getContentTypeFromEnv(t),
		}
		conn, err := http.NewConnection(config)
		if err != nil {
			t.Fatalf("Failed to create new http connection: %s", describe(err))
		}
		return conn

	default:
		t.Fatalf("Unknown connection type: '%s'", connSpec)
		return nil
	}
}

// createClientFromEnv initializes a Client from information specified in environment variables.
func createClientFromEnv(t testEnv, waitUntilReady bool, connection ...*driver.Connection) driver.Client {
	runPProfServerOnce.Do(func() {
		if os.Getenv("TEST_PPROF") != "" {
			go func() {
				// Start pprof server on port 6060
				// To use it in the test, run a command like:
				// docker exec -it go-driver-test sh -c "apk add -U curl && curl http://localhost:6060/debug/pprof/goroutine?debug=1"
				log.Println(httplib.ListenAndServe("localhost:6060", nil))
			}()
		}
	})
	conn := createConnectionFromEnv(t)
	if len(connection) == 1 {
		*connection[0] = conn
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: createAuthenticationFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new client: %s", describe(err))
	}
	if waitUntilReady {
		timeout := time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if up := waitUntilServerAvailable(ctx, c, t); up != nil {
			t.Fatalf("Connection is not available in %s: %s", timeout, describe(up))
		}

		if testModeSingle != getTestMode() {
			// Synchronize endpoints
			if err := waitUntilEndpointSynchronized(ctx, c, "", t); err != nil {
				t.Errorf("Failed to synchronize endpoints: %s", describe(err))
			} else {
				logEndpointsOnce.Do(func() {
					t.Logf("Found endpoints: %v", conn.Endpoints())
				})
			}
		}
	}
	return c
}

// waitUntilServerAvailable keeps waiting until the server/cluster that the client is addressing is available.
func waitUntilServerAvailable(ctx context.Context, c driver.Client, t testEnv) error {
	instanceUp := make(chan error)
	go func() {
		for {
			verCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if _, err := c.Version(verCtx); err == nil {

				// check if leader challenge is ongoing
				if _, err := c.ServerRole(verCtx); err != nil {
					if !driver.IsNoLeaderOrOngoing(err) {
						cancel()
						instanceUp <- err
						return
					}
					//t.Logf("Retry. Waiting for leader: %s", describe(err))
					continue
				}

				//t.Logf("Found version %s", v.Version)
				cancel()
				instanceUp <- nil
				return
			}
			cancel()
			//t.Logf("Version failed: %s %#v", describe(err), err)
			time.Sleep(time.Second * 2)
		}
	}()
	select {
	case up := <-instanceUp:
		return up
	case <-ctx.Done():
		return nil
	}
}

// waitUntilClusterHealthy keeps waiting until the servers are healthy
func waitUntilClusterHealthy(c driver.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Cluster(ctx); err != nil {
		if driver.IsPreconditionFailed(err) {
			// only in cluster mode
			return nil
		}

		return err
	}

	return retry(time.Second, time.Minute, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cluster, err := c.Cluster(ctx)
		if err != nil {
			return err
		}

		health, err := cluster.Health(ctx)
		if err != nil {
			return err
		}

		for _, h := range health.Health {
			if h.Status != driver.ServerStatusGood {
				return nil
			}
		}

		return interrupt{}
	})
}

// waitUntilEndpointSynchronized keeps waiting until the endpoints are synchronized. leadership might be ongoing.
func waitUntilEndpointSynchronized(ctx context.Context, c driver.Client, dbname string, t testEnv) error {
	endpointsSynced := make(chan error)
	go func() {
		for {
			callCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if err := c.SynchronizeEndpoints2(callCtx, dbname); err != nil {
				t.Logf("SynchonizedEnpoints failed: %s", describe(err))
			} else {
				cancel()
				endpointsSynced <- nil
				return
			}
			cancel()
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
	select {
	case up := <-endpointsSynced:
		return up
	case <-ctx.Done():
		return fmt.Errorf("Timeout while synchronizing endpoints")
	}
}

// TestCreateClientHttpConnection creates an HTTP connection to the environment specified
// endpoints and creates a client for that.
func TestCreateClientHttpConnection(t *testing.T) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: getEndpointsFromEnv(t),
		Transport: NewConnectionTransport(),
	})
	if err != nil {
		t.Fatalf("Failed to create new http connection: %s", describe(err))
	}
	_, err = driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: createAuthenticationFromEnv(t),
	})
	if err != nil {
		t.Fatalf("Failed to create new client: %s", describe(err))
	}
}

// TestResponseHeader checks the Response.Header function.
func TestResponseHeader(t *testing.T) {
	c := createClientFromEnv(t, true)
	ctx := context.Background()

	version, err := c.Version(nil)
	if err != nil {
		t.Fatalf("Version failed: %s", describe(err))
	}
	isv33p := version.Version.CompareTo("3.3") >= 0
	if !isv33p {
		t.Skip("This test requires version 3.3")
	} else {
		var resp driver.Response
		db := ensureDatabase(ctx, c, "_system", nil, t)
		col := ensureCollection(ctx, db, "response_header_test", nil, t)
		defer clean(t, ctx, col)

		// `ETag` header must contain the `_rev` of the new document in quotes.
		doc := map[string]string{
			"Test":   "TestResponseHeader",
			"Intent": "Check Response.Header",
		}
		meta, err := col.CreateDocument(driver.WithResponse(ctx, &resp), doc)
		if err != nil {
			t.Fatalf("CreateDocument failed: %s", describe(err))
		}
		expectedETag := strconv.Quote(meta.Rev)
		if x := resp.Header("ETag"); x != expectedETag {
			t.Errorf("Unexpected result from Header('ETag'), got '%s', expected '%s'", x, expectedETag)
		}
		if x := resp.Header("Etag"); x != expectedETag {
			t.Errorf("Unexpected result from Header('Etag'), got '%s', expected '%s'", x, expectedETag)
		}
		if x := resp.Header("etag"); x != expectedETag {
			t.Errorf("Unexpected result from Header('etag'), got '%s', expected '%s'", x, expectedETag)
		}
		if x := resp.Header("ETAG"); x != expectedETag {
			t.Errorf("Unexpected result from Header('ETAG'), got '%s', expected '%s'", x, expectedETag)
		}
	}
}

type dummyRequestRepeat struct {
	counter int
}

func (r *dummyRequestRepeat) Repeat(conn driver.Connection, resp driver.Response, err error) bool {
	r.counter++
	if r.counter == 2 {
		return false
	}
	return true
}

func TestCreateClientHttpRepeatConnection(t *testing.T) {
	if getTestMode() != testModeSingle {
		t.Skipf("Not a single")
	}
	createClientFromEnv(t, true)

	requestRepeat := dummyRequestRepeat{}
	conn := createConnectionFromEnv(t)
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     http.NewRepeatConnection(conn, &requestRepeat),
		Authentication: createAuthenticationFromEnv(t),
	})

	_, err = c.Connection().SetAuthentication(createAuthenticationFromEnv(t))
	assert.Equal(t, http.ErrAuthenticationNotChanged, err)

	_, err = c.Databases(nil)
	require.NoError(t, err)
	assert.Equal(t, 2, requestRepeat.counter)
}
