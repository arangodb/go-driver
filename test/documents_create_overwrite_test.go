//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech <tomasz@arangodb.com>
//

package test

import (
	"context"
	"strings"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/dchest/uniuri"
	"github.com/stretchr/testify/require"
)

func generateIDs(count int) []string {
	s := make([]string, count)
	for i := 0; i < count; i++ {
		s[i] = strings.ToLower(uniuri.NewLen(16))
	}
	return s
}

// TestCreateOverwriteDocument creates a document and then checks that it exists. Check with overwrite flag.
func TestCreateOverwriteDocument(t *testing.T) {
	c := createClientFromEnv(t, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_overwrite_test", nil, t)

	t.Run("Single Doc - replace", func(t *testing.T) {
		id := generateIDs(1)[0]

		first := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName",
			Age:  10,
		}

		_, err := col.CreateDocument(ctx, first)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
		}

		second := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName2",
			Age:  100,
		}

		_, err = col.CreateDocument(driver.WithOverwrite(ctx), second)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.NotEqual(t, first, result)
			require.Equal(t, second, result)
		}
	})

}

// TestCreateOverwriteModeDocument creates a document and then checks that it exists. Check with overwriteMode flag.
func TestCreateOverwriteModeDocument(t *testing.T) {
	c := createClientFromEnv(t, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.7.0"))

	db := ensureDatabase(nil, c, "document_test", nil, t)
	col := ensureCollection(nil, db, "document_test", nil, t)

	t.Run("Single Doc - ignore", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeIgnore)
		id := generateIDs(1)[0]

		first := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName",
			Age:  10,
		}

		_, err := col.CreateDocument(newC, first)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
		}

		second := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName2",
			Age:  100,
		}

		_, err = col.CreateDocument(newC, second)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
			require.NotEqual(t, second, result)
		}
	})

	t.Run("Single Doc - replace", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeReplace)
		id := generateIDs(1)[0]

		first := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName",
			Age:  10,
		}

		_, err := col.CreateDocument(newC, first)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
		}

		second := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName2",
			Age:  100,
		}

		_, err = col.CreateDocument(newC, second)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.NotEqual(t, first, result)
			require.Equal(t, second, result)
		}
	})

	t.Run("Single Doc - update", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeUpdate)
		id := generateIDs(1)[0]

		first := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName",
			Age:  10,
		}

		_, err := col.CreateDocument(newC, first)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
		}

		second := UserDocWithKeyWithOmit{
			Key: id,
			Age: 100,
		}

		_, err = col.CreateDocument(newC, second)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.NotEqual(t, first, result)
			require.NotEqual(t, second, result)

			require.Equal(t, first.Name, result.Name)
			require.Equal(t, second.Age, result.Age)

			require.NotEqual(t, second.Name, result.Name)
			require.NotEqual(t, first.Age, result.Age)
		}
	})

	t.Run("Single Doc - conflict", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeConflict)
		id := generateIDs(1)[0]

		first := UserDocWithKeyWithOmit{
			Key:  id,
			Name: "MyName",
			Age:  10,
		}

		_, err := col.CreateDocument(newC, first)
		require.NoError(t, err)

		{
			var result UserDocWithKeyWithOmit
			_, err := col.ReadDocument(ctx, id, &result)
			require.NoError(t, err)

			require.Equal(t, first, result)
		}

		second := UserDocWithKeyWithOmit{
			Key: id,
			Age: 100,
		}

		_, err = col.CreateDocument(newC, second)
		require.Error(t, err)
		require.True(t, driver.IsConflict(err))
	})

	t.Run("Multi Doc - ignore", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeIgnore)
		id := generateIDs(2)

		firstDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1",
				Age:  11,
			},
			{
				Key:  id[1],
				Name: "Name2",
				Age:  22,
			},
		}

		_, _, err := col.CreateDocuments(newC, firstDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.Equal(t, firstDocs, result)
		}

		secondDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1-2",
				Age:  111,
			},
			{
				Key:  id[1],
				Name: "Name2-2",
				Age:  222,
			},
		}

		_, _, err = col.CreateDocuments(newC, secondDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.Equal(t, firstDocs, result)
			require.NotEqual(t, secondDocs, result)
		}
	})

	t.Run("Multi Doc - replace", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeReplace)
		id := generateIDs(2)

		firstDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1",
				Age:  11,
			},
			{
				Key:  id[1],
				Name: "Name2",
				Age:  22,
			},
		}

		_, _, err := col.CreateDocuments(newC, firstDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.Equal(t, firstDocs, result)
		}

		secondDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1-2",
				Age:  111,
			},
			{
				Key:  id[1],
				Name: "Name2-2",
				Age:  222,
			},
		}

		_, _, err = col.CreateDocuments(newC, secondDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.NotEqual(t, firstDocs, result)
			require.Equal(t, secondDocs, result)
		}
	})

	t.Run("Multi Doc - update", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeUpdate)
		id := generateIDs(2)

		firstDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1",
				Age:  11,
			},
			{
				Key:  id[1],
				Name: "Name2",
				Age:  22,
			},
		}

		_, _, err := col.CreateDocuments(newC, firstDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.Equal(t, firstDocs, result)
		}

		secondDocs := []UserDocWithKeyWithOmit{
			{
				Key: id[0],
				Age: 111,
			},
			{
				Key:  id[1],
				Name: "Name2-new",
			},
		}

		_, _, err = col.CreateDocuments(newC, secondDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.NotEqual(t, firstDocs, result)
			require.NotEqual(t, secondDocs, result)

			require.Equal(t, firstDocs[0].Name, result[0].Name)
			require.Equal(t, secondDocs[0].Age, result[0].Age)
			require.Equal(t, secondDocs[1].Name, result[1].Name)
			require.Equal(t, firstDocs[1].Age, result[1].Age)

			require.NotEqual(t, secondDocs[0].Name, result[0].Name)
			require.NotEqual(t, firstDocs[0].Age, result[0].Age)
			require.NotEqual(t, firstDocs[1].Name, result[1].Name)
			require.NotEqual(t, secondDocs[1].Age, result[1].Age)
		}
	})

	t.Run("Multi Doc - conflict", func(t *testing.T) {
		newC := driver.WithOverwriteMode(ctx, driver.OverwriteModeConflict)
		id := generateIDs(2)

		firstDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1",
				Age:  11,
			},
			{
				Key:  id[1],
				Name: "Name2",
				Age:  22,
			},
		}

		_, _, err := col.CreateDocuments(newC, firstDocs)
		require.NoError(t, err)

		{
			result := make([]UserDocWithKeyWithOmit, 2)
			_, _, err := col.ReadDocuments(ctx, id, result)
			require.NoError(t, err)

			require.Equal(t, firstDocs, result)
		}

		secondDocs := []UserDocWithKeyWithOmit{
			{
				Key:  id[0],
				Name: "Name1",
				Age:  11,
			},
			{
				Key:  id[1],
				Name: "Name2-new",
			},
		}

		_, errSlice, err := col.CreateDocuments(newC, secondDocs)
		require.NoError(t, err)

		require.EqualError(t, errSlice[0], "unique constraint violated - in index primary of type primary over '_key'; conflicting key: "+id[0])
		require.EqualError(t, errSlice[1], "unique constraint violated - in index primary of type primary over '_key'; conflicting key: "+id[1])
	})
}
