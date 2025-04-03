//
// DISCLAIMER
//
// Copyright 2025 ArangoDB GmbH, Cologne, Germany
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

package tests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/utils"
	"github.com/stretchr/testify/require"
)

func parseInt64FromInterface(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8, int16, int32, int64:
		return v.(int64), nil
	case uint, uint8, uint16, uint32, uint64:
		return int64(v.(uint64)), nil
	case float32, float64:
		return int64(v.(float64)), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("value is of type %T, not convertible to int64", v)
	}
}

func Test_CollectionComputedValues(t *testing.T) {
	Wrap(t, func(t *testing.T, client arangodb.Client) {
		WithDatabase(t, client, nil, func(db arangodb.Database) {

			t.Run("Create with ComputedValues", func(t *testing.T) {
				name := "test_users_computed_values"

				// Add an attribute with the creation timestamp to new documents
				computedValue := arangodb.ComputedValue{
					Name:       "createdAt",
					Expression: "RETURN DATE_NOW()",
					Overwrite:  true,
					ComputeOn:  []arangodb.ComputeOn{arangodb.ComputeOnInsert},
				}

				_, err := db.CreateCollectionV2(nil, name, &arangodb.CreateCollectionPropertiesV2{
					ComputedValues: &[]arangodb.ComputedValue{computedValue},
				})
				require.NoError(t, err)

				// Collection must exist now
				col, err := db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err := col.Properties(nil)
				require.NoError(t, err)

				// Check if the computed value is in the list of computed values
				require.Len(t, prop.ComputedValues, 1)
				require.Equal(t, computedValue.Name, prop.ComputedValues[0].Name)
				require.Len(t, prop.ComputedValues[0].ComputeOn, 1)
				require.Equal(t, computedValue.ComputeOn[0], prop.ComputedValues[0].ComputeOn[0])
				require.Equal(t, computedValue.Expression, prop.ComputedValues[0].Expression)

				// Create a document
				doc := UserDoc{Name: fmt.Sprintf("Jakub")}
				meta, err := col.CreateDocument(nil, doc)
				if err != nil {
					t.Fatalf("Failed to create document: %s", err)
				}

				// Read document
				var readDoc map[string]interface{}
				if _, err := col.ReadDocument(nil, meta.Key, &readDoc); err != nil {
					t.Fatalf("Failed to read document '%s': %s", meta.Key, err)
				}

				require.Equal(t, doc.Name, readDoc["name"])

				// Verify that the computed value is set
				createdAtValue, createdAtIsPresent := readDoc["createdAt"]
				require.True(t, createdAtIsPresent)

				t.Logf("createdAtValue raw value: %v", createdAtValue)
				createdAtValueInt64, err := parseInt64FromInterface(createdAtValue)
				require.NoError(t, err)
				t.Logf("createdAtValue parsed value: %v", createdAtValueInt64)

				tm := time.Unix(createdAtValueInt64, 0)
				require.True(t, tm.After(time.Now().Add(-time.Second)))
			})

			t.Run("Update to ComputedValues", func(t *testing.T) {
				name := "test_update_computed_values"

				// Add an attribute with the creation timestamp to new documents
				computedValue := arangodb.ComputedValue{
					Name:       "createdAt",
					Expression: "RETURN DATE_NOW()",
					Overwrite:  true,
					ComputeOn:  []arangodb.ComputeOn{arangodb.ComputeOnInsert},
				}

				_, err := db.CreateCollectionV2(nil, name, nil)
				require.NoError(t, err)

				// Collection must exist now
				col, err := db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err := col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 0)

				err = col.SetPropertiesV2(nil, arangodb.SetCollectionPropertiesOptionsV2{
					ComputedValues: &[]arangodb.ComputedValue{computedValue},
				})
				require.NoError(t, err)

				// Check if the computed value is in the list of computed values
				col, err = db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err = col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 1)
			})

			t.Run("Use default ComputeOn values in ComputedValues", func(t *testing.T) {
				name := "test_default_computeon_computed_values"

				// Add an attribute with the creation timestamp to new documents
				computedValue := arangodb.ComputedValue{
					Name:       "createdAt",
					Expression: "RETURN DATE_NOW()",
					Overwrite:  true,
				}

				_, err := db.CreateCollectionV2(nil, name, nil)
				require.NoError(t, err)

				// Collection must exist now
				col, err := db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err := col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 0)

				err = col.SetPropertiesV2(nil, arangodb.SetCollectionPropertiesOptionsV2{
					ComputedValues: &[]arangodb.ComputedValue{computedValue},
				})
				require.NoError(t, err)

				// Check if the computed value is in the list of computed values
				col, err = db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err = col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 1)
				// we should get the default value for ComputeOn - ["insert", "update", "replace"]
				require.Len(t, prop.ComputedValues[0].ComputeOn, 3)
			})

			t.Run("Update to remove ComputedValues", func(t *testing.T) {
				name := "test_update_remove_computed_values"

				// Add an attribute with the creation timestamp to new documents
				computedValue := arangodb.ComputedValue{
					Name:       "createdAt",
					Expression: "RETURN DATE_NOW()",
					Overwrite:  true,
					ComputeOn:  []arangodb.ComputeOn{arangodb.ComputeOnInsert},
				}

				_, err := db.CreateCollectionV2(nil, name, &arangodb.CreateCollectionPropertiesV2{
					ComputedValues: &[]arangodb.ComputedValue{computedValue},
				})
				require.NoError(t, err)

				// Collection must exist now
				col, err := db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err := col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 1)

				err = col.SetPropertiesV2(nil,
					arangodb.SetCollectionPropertiesOptionsV2{
						ComputedValues: &[]arangodb.ComputedValue{},
					},
				)
				require.NoError(t, err)

				prop, err = col.Properties(nil)
				require.NoError(t, err)
				require.Len(t, prop.ComputedValues, 0)

			})

			t.Run("Update without removing ComputedValues", func(t *testing.T) {
				name := "test_update_wIo_removing_computed_values"

				// Add an attribute with the creation timestamp to new documents
				computedValue := arangodb.ComputedValue{
					Name:       "createdAt",
					Expression: "RETURN DATE_NOW()",
					Overwrite:  true,
					ComputeOn:  []arangodb.ComputeOn{arangodb.ComputeOnInsert},
				}

				_, err := db.CreateCollectionV2(nil, name, &arangodb.CreateCollectionPropertiesV2{
					ComputedValues: &[]arangodb.ComputedValue{computedValue},
				})
				require.NoError(t, err)

				// Collection must exist now
				col, err := db.GetCollection(nil, name, nil)
				require.NoError(t, err)

				prop, err := col.Properties(nil)
				require.NoError(t, err)

				require.Len(t, prop.ComputedValues, 1)

				err = col.SetPropertiesV2(nil,
					arangodb.SetCollectionPropertiesOptionsV2{
						MinReplicationFactor: utils.NewType(3),
					},
				)
				require.NoError(t, err)

				prop, err = col.Properties(nil)
				require.NoError(t, err)
				require.Len(t, prop.ComputedValues, 1)

			})
		})
	})
}
