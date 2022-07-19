//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

func Test_Pool(t *testing.T) {
	WrapConnectionFactory(t, func(t *testing.T, connFactory ConnectionFactory) {
		conn, err := connection.NewPool(5, func() (connection.Connection, error) {
			return connFactory(t), nil
		})
		require.NoError(t, err)

		client := arangodb.NewClient(conn)

		var wg sync.WaitGroup
		ctx := context.Background()

		for i := 0; i < 8; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				for j := 0; j < 16; j++ {
					_, err := client.Version(ctx)
					require.NoError(t, err)
				}
			}()
		}

		wg.Wait()
	})
}
