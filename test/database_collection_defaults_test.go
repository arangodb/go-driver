//
// DISCLAIMER
//
// Copyright 2019 ArangoDB GmbH, Cologne, Germany
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
//

package test

import (
	"github.com/arangodb/kube-arangodb/.gobuild/pkg/mod/github.com/dchest/uniuri@v0.0.0-20160212164326-8902c56451e9"
	"strings"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
)

// TestDatabaseSharding test if proper sharding is passed to database
func TestDatabaseSharding(t *testing.T) {
	c := createClientFromEnv(t, true)

	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)

	skipNoEnterprise(t)

	type scenario struct {
		sharding, expected driver.DatabaseSharding
	}

	var scenarios = map[string]scenario{
		"empty": {
			sharding: "",
			expected: driver.DatabaseShardingNone,
		},
		"unknown": {
			sharding: "unknown",
			expected: driver.DatabaseShardingNone,
		},
		"single": {
			sharding: driver.DatabaseShardingSingle,
			expected: driver.DatabaseShardingSingle,
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			name := databaseName("database_sharding", strings.ReplaceAll(name, " ", "_"))

			opt := driver.CreateDatabaseOptions{
				Options: driver.CreateDatabaseDefaultOptions{
					Sharding: scenario.sharding,
				},
			}

			db, err := c.CreateDatabase(nil, name, &opt)
			require.NoError(t, err)

			info, err := db.Info(nil)
			require.NoError(t, err)

			require.Equal(t, scenario.expected, info.Sharding)
		})
	}
}

func TestDatabaseDefaults(t *testing.T) {
	c := createClientFromEnv(t, true)

	skipBelowVersion(c, "3.6", t)
	skipNoCluster(c, t)

	skipNoEnterprise(t)

	type scenario struct {
		name     string
		db       driver.CreateDatabaseDefaultOptions
		col      driver.CreateCollectionOptions
		expected driver.CollectionProperties
	}

	scenarios := []scenario{
		{
			name: "replication factor from db",
			db: driver.CreateDatabaseDefaultOptions{
				ReplicationFactor: 3,
			},
			expected: driver.CollectionProperties{
				ReplicationFactor: 3,
			},
		},
		{
			name: "replication factor from col",
			db: driver.CreateDatabaseDefaultOptions{
				ReplicationFactor: 3,
			},
			col: driver.CreateCollectionOptions{
				ReplicationFactor: 2,
			},
			expected: driver.CollectionProperties{
				ReplicationFactor: 2,
			},
		},
		{
			name: "min replication factor from db",
			db: driver.CreateDatabaseDefaultOptions{
				WriteConcern:      3,
				ReplicationFactor: 3,
			},
			expected: driver.CollectionProperties{
				MinReplicationFactor: 3,
				ReplicationFactor:    3,
			},
		},
		{
			name: "min replication factor from col",
			db: driver.CreateDatabaseDefaultOptions{
				WriteConcern:      3,
				ReplicationFactor: 3,
			},
			col: driver.CreateCollectionOptions{
				MinReplicationFactor: 2,
			},
			expected: driver.CollectionProperties{
				MinReplicationFactor: 2,
				ReplicationFactor:    3,
			},
		},
		{
			name: "min replication and replication factor from col",
			db: driver.CreateDatabaseDefaultOptions{
				WriteConcern:      3,
				ReplicationFactor: 3,
			},
			col: driver.CreateCollectionOptions{
				MinReplicationFactor: 2,
				ReplicationFactor:    2,
			},
			expected: driver.CollectionProperties{
				MinReplicationFactor: 2,
				ReplicationFactor:    2,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {

			name := databaseName("database_defaults", strings.ToLower(uniuri.NewLen(5)))

			opt := driver.CreateDatabaseOptions{
				Options: scenario.db,
			}

			db, err := c.CreateDatabase(nil, name, &opt)
			require.NoError(t, err)

			col, err := db.CreateCollection(nil, "test", &scenario.col)
			require.NoError(t, err)

			prop, err := col.Properties(nil)
			require.NoError(t, err)

			if scenario.expected.ReplicationFactor != 0 {
				require.Equal(t, scenario.expected.ReplicationFactor, prop.ReplicationFactor)
			}

			if scenario.expected.MinReplicationFactor != 0 {
				require.Equal(t, scenario.expected.MinReplicationFactor, prop.MinReplicationFactor)
			}
		})
	}
}
