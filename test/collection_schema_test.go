//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arangodb/go-driver"
)

func jsonEqual(t *testing.T, a, b interface{}) {
	ad, err := json.Marshal(a)
	require.NoError(t, err)
	bd, err := json.Marshal(b)
	require.NoError(t, err)

	require.Equal(t, string(ad), string(bd))
}

// TestCreateOverwriteDocument creates a document and then checks that it exists. Check with overwrite flag
func TestCollectionSchema(t *testing.T) {
	c := createClient(t, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	EnsureVersion(t, ctx, c).CheckVersion(MinimumVersion("3.10.0"))

	name := "document_schema_validation_test"
	db := ensureDatabase(nil, c, name, nil, t)
	defer func() {
		err := db.Remove(nil)
		if err != nil {
			t.Logf("Failed to drop database %s: %s ...", db.Name(), err)
		}
	}()
	t.Run("Create collection with schema validation", func(t *testing.T) {
		opts := driver.CreateCollectionOptions{
			Schema: &driver.CollectionSchemaOptions{
				Level:   driver.CollectionSchemaLevelStrict,
				Message: "Validation Err",
				Type:    "json",
			},
		}

		require.NoError(t, opts.Schema.LoadRule([]byte(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"description": "Age in years",
				"type": "integer",
				"minimum": 0
			}
		},
		"required": ["firstName", "lastName"]
}`)))

		col := ensureCollection(nil, db, "document_schema_validation_test", &opts, t)

		loadOpts, err := col.Properties(ctx)
		require.NoError(t, err)

		jsonEqual(t, opts.Schema, loadOpts.Schema)
	})

	col, err := db.Collection(ctx, name)
	require.NoError(t, err)

	t.Run("Update collection with schema validation", func(t *testing.T) {
		schema := &driver.CollectionSchemaOptions{
			Level:   driver.CollectionSchemaLevelStrict,
			Message: "Validation Err",
			Type:    "json",
		}

		require.NoError(t, schema.LoadRule([]byte(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"description": "Age in years",
				"type": "integer",
				"minimum": 0
			},
			"age2": {
				"description": "Age in years",
				"type": "integer",
				"minimum": 0
			}
		},
		"required": ["firstName", "lastName"]
}`)))

		require.NoError(t, col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{
			Schema: schema,
		}))

		loadOpts, err := col.Properties(ctx)
		require.NoError(t, err)

		jsonEqual(t, schema, loadOpts.Schema)
	})

	t.Run("Update collection with invalid schema", func(t *testing.T) {
		schema := &driver.CollectionSchemaOptions{
			Level:   driver.CollectionSchemaLevelStrict,
			Message: "Validation Err",
			Type:    "json",
		}

		require.NoError(t, schema.LoadRule([]byte(`{
		"type": 4,
		"properties": [],
		"required": {}
}`)))

		err := col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{
			Schema: schema,
		})
		require.Error(t, err)

		arangoErr, ok := err.(driver.ArangoError)
		require.True(t, ok)

		require.Equal(t, http.StatusBadRequest, arangoErr.Code)
	})

	t.Run("Update collection with valid schema and create docs", func(t *testing.T) {
		schema := &driver.CollectionSchemaOptions{
			Level:   driver.CollectionSchemaLevelStrict,
			Message: "Validation Err",
			Type:    "json",
		}

		require.NoError(t, schema.LoadRule([]byte(`{
			"properties": {
				"name": {
					"type": "string"
				}
			},
			"required": ["name"]
}`)))

		col := ensureCollection(nil, db, "document_schema_validation_test_wo_opts", &driver.CreateCollectionOptions{
			Schema: schema,
		}, t)

		t.Run("Success", func(t *testing.T) {
			u := UserDocWithKeyWithOmit{
				Key:  NewUUID(),
				Name: "name",
			}

			_, err := col.CreateDocument(ctx, u)
			require.NoError(t, err)
		})
		t.Run("Failure", func(t *testing.T) {

			u := UserDocWithKeyWithOmit{
				Key: NewUUID(),
			}

			_, err := col.CreateDocument(ctx, u)
			require.Error(t, err)

			arangoErr, ok := err.(driver.ArangoError)
			require.True(t, ok)

			require.Equal(t, http.StatusBadRequest, arangoErr.Code)
		})
	})
}
