// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
)

func TestEventEditor(t *testing.T) {
	metadataNestedNestedMap := mapstr.M{
		"metaLevel2Value": "metavalue3",
	}
	metadataNestedMap := mapstr.M{
		"metaLevel1Map": metadataNestedNestedMap,
	}

	fieldsNestedNestedMap := mapstr.M{
		"fieldsLevel2Value": "fieldsvalue3",
	}
	fieldsNestedMap := mapstr.M{
		"fieldsLevel1Map": fieldsNestedNestedMap,
	}

	metaUntouchedMap := mapstr.M{}
	fieldsUntouchedMap := mapstr.M{}

	event := &Event{
		Timestamp: time.Now(),
		Meta: mapstr.M{
			"a.b":             "c",
			"metaLevel0Map":   metadataNestedMap,
			"metaLevel0Value": "metavalue1",
			// this key should never be edited by the tests
			// to verify that existing keys remain
			"metaLevel0Value2": "untouched",
			// this map should never be checked out
			"metaUntouchedMap": metaUntouchedMap,
		},
		Fields: mapstr.M{
			"a.b":               "c",
			"fieldsLevel0Map":   fieldsNestedMap,
			"fieldsLevel0Value": "fieldsvalue1",
			// this key should never be edited by the tests
			// to verify that existing keys remain
			"fieldsLevel0Value2": "untouched",
			// this map should never be checked out
			"fieldsUntouchedMap": fieldsUntouchedMap,
		},
	}

	t.Run("rootKey", func(t *testing.T) {
		event := event.Clone()
		event.Meta["some.dot.metakey"] = mapstr.M{
			"that.should": mapstr.M{
				"be": "supported",
			},
		}
		event.Fields["some.dot.key"] = mapstr.M{
			"that.should": mapstr.M{
				"be": "supported",
			},
		}
		cases := []struct {
			val string
			exp string
		}{
			{
				val: "@metadata.a.b",
				exp: "@metadata.a.b",
			},
			{
				val: "@metadata.metaLevel0Value",
				exp: "@metadata.metaLevel0Value",
			},
			{
				val: "@metadata.metaLevel0Map.metaLevel1Map",
				exp: "@metadata.metaLevel0Map",
			},
			{
				val: "@metadata.some.dot.metakey.that.should.be",
				exp: "@metadata.some.dot.metakey",
			},
			{
				val: "a.b",
				exp: "a.b",
			},
			{
				val: "fieldsLevel0Map.fieldsLevel1Value",
				exp: "fieldsLevel0Map",
			},
			{
				val: "fieldsLevel0Map.fieldsLevel1Map.fieldsLevel2Value",
				exp: "fieldsLevel0Map",
			},
			{
				val: "fieldsLevel0Value",
				exp: "fieldsLevel0Value",
			},
			{
				val: "some.dot.key.that.should.be",
				exp: "some.dot.key",
			},
		}

		for _, tc := range cases {
			e := NewEventEditor(event)
			t.Run(tc.val, func(t *testing.T) {
				require.Equal(t, tc.exp, e.rootKey(tc.val))
			})
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Run("Apply", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				editor.Apply()
			})
		})

		t.Run("Reset", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				editor.Reset()
			})
		})

		t.Run("Delete", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				err := editor.Delete("@metadata.some")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				err = editor.Delete("some")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				editor.Apply()
			})
		})

		t.Run("GetValue", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				_, err := editor.GetValue("@metadata.some")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				_, err = editor.GetValue("some")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})
		})

		t.Run("PutValue", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				prev, err := editor.PutValue("@metadata.some", "value")
				require.NoError(t, err)
				require.Nil(t, prev)
				prev, err = editor.PutValue("some", "value")
				require.NoError(t, err)
				require.Nil(t, prev)
				editor.Apply()
			})
		})

		t.Run("DeepUpdate", func(t *testing.T) {
			editor := NewEventEditor(nil)
			require.NotPanics(t, func() {
				editor.DeepUpdate(mapstr.M{
					"@metadata": mapstr.M{"key": "value"},
					"key":       "value",
				})
				editor.Apply()
			})
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("@timestamp", func(t *testing.T) {
			editor := NewEventEditor(event)
			val, err := editor.GetValue("@timestamp")
			require.NoError(t, err)
			require.Equal(t, event.Timestamp, val)
		})

		t.Run("@metadata", func(t *testing.T) {
			t.Run("no acess to @metadata key", func(t *testing.T) {
				editor := NewEventEditor(event)
				metadata, err := editor.GetValue("@metadata")
				require.Nil(t, metadata)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrMetadataAccess)
			})

			t.Run("non-existing key", func(t *testing.T) {
				editor := NewEventEditor(event)
				val, err := editor.GetValue("@metadata.none")
				require.Nil(t, val)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("gets a value type", func(t *testing.T) {
				editor := NewEventEditor(event)
				requireMapValues(t, editor, map[string]interface{}{
					"@metadata.metaLevel0Value": "metavalue1",
				})
			})

			t.Run("gets a root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				requireMapValues(t, editor, map[string]interface{}{
					"@metadata.a.b": "c",
				})
			})

			t.Run("gets a nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				nested, err := editor.GetValue("@metadata.metaLevel0Map")
				require.NoError(t, err)
				requireClonedMaps(t, metadataNestedMap, nested)
			})

			t.Run("gets a deeper nested map", func(t *testing.T) {
				editor := NewEventEditor(event)

				nested, err := editor.GetValue("@metadata.metaLevel0Map.metaLevel1Map")
				require.NoError(t, err)
				requireClonedMaps(t, metadataNestedNestedMap, nested)

				// the higher level map should be also cloned by accessing the inner map
				higher, err := editor.GetValue("@metadata.metaLevel0Map")
				require.NoError(t, err)
				requireClonedMaps(t, metadataNestedMap, higher)

				// the nested map we got previously should be a part of this cloned higher level map
				require.IsType(t, mapstr.M{}, higher)
				higherMap := higher.(mapstr.M)
				nested2, err := higherMap.GetValue("metaLevel1Map")
				require.NoError(t, err)
				requireSameMap(t, nested, nested2)
			})

			t.Run("returns the same nested map twice", func(t *testing.T) {
				editor := NewEventEditor(event)
				nested1, err := editor.GetValue("@metadata.metaLevel0Map.metaLevel1Map")
				require.NoError(t, err)
				nested2, err := editor.GetValue("@metadata.metaLevel0Map.metaLevel1Map")
				require.NoError(t, err)
				requireSameMap(t, nested2, nested1)
			})
		})

		t.Run("fields", func(t *testing.T) {
			t.Run("non-existing key", func(t *testing.T) {
				editor := NewEventEditor(event)
				val, err := editor.GetValue("none")
				require.Nil(t, val)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("gets a value type", func(t *testing.T) {
				editor := NewEventEditor(event)
				requireMapValues(t, editor, map[string]interface{}{
					"fieldsLevel0Value": "fieldsvalue1",
				})
			})

			t.Run("gets a root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				requireMapValues(t, editor, map[string]interface{}{
					"a.b": "c",
				})
			})

			t.Run("gets a nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				nested, err := editor.GetValue("fieldsLevel0Map")
				require.NoError(t, err)
				requireClonedMaps(t, fieldsNestedMap, nested)
			})

			t.Run("gets a deeper nested map", func(t *testing.T) {
				editor := NewEventEditor(event)

				nested, err := editor.GetValue("fieldsLevel0Map.fieldsLevel1Map")
				require.NoError(t, err)
				requireClonedMaps(t, fieldsNestedNestedMap, nested)

				// the higher level map should be also cloned by accessing the inner map
				higher, err := editor.GetValue("fieldsLevel0Map")
				require.NoError(t, err)
				requireClonedMaps(t, fieldsNestedMap, higher)

				// the nested map we got previously should be a part of this higher level map
				require.IsType(t, mapstr.M{}, higher)
				higherMap := higher.(mapstr.M)
				nested2, err := higherMap.GetValue("fieldsLevel1Map")
				require.NoError(t, err)
				requireSameMap(t, nested, nested2)
			})

			t.Run("returns the same nested map twice", func(t *testing.T) {
				editor := NewEventEditor(event)
				nested1, err := editor.GetValue("fieldsLevel0Map.fieldsLevel1Map")
				require.NoError(t, err)
				nested2, err := editor.GetValue("fieldsLevel0Map.fieldsLevel1Map")
				require.NoError(t, err)
				requireSameMap(t, nested2, nested1)
			})
		})

	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("@timestamp", func(t *testing.T) {
			editor := NewEventEditor(event)
			err := editor.Delete("@timestamp")
			require.Error(t, err)
			require.ErrorIs(t, err, ErrDeleteTimestamp)
		})

		t.Run("@metadata", func(t *testing.T) {
			t.Run("metadata itself", func(t *testing.T) {
				editor := NewEventEditor(event)
				err := editor.Delete("@metadata")
				require.Error(t, err)
				require.ErrorIs(t, err, ErrAlterMetadataKey)
			})

			t.Run("non-existent key", func(t *testing.T) {
				editor := NewEventEditor(event)

				err := editor.Delete("@metadata.wrong")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				err = editor.Delete("@metadata.wrong.key")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("root-level value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Value"
				value := "metavalue1"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.a.b"
				value := "c"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("root-level map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, metadataNestedMap, val3)
			})

			t.Run("nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map.metaLevel2Value"
				value := "metavalue3"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				val1, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val1)

				// checking if the original event still has it
				val2, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, metadataNestedNestedMap, val2)
			})

			t.Run("nested map twice", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				err = editor.Delete(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// checking if the original event still has it
				val1, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, metadataNestedNestedMap, val1)
			})
		})

		t.Run("fields", func(t *testing.T) {
			t.Run("non-existent key", func(t *testing.T) {
				editor := NewEventEditor(event)

				err := editor.Delete("wrong")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				err = editor.Delete("wrong.key")
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("root-level value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Value"
				value := "fieldsvalue1"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "a.b"
				value := "c"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("root-level map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, fieldsNestedMap, val3)
			})

			t.Run("nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map.fieldsLevel2Value"
				value := "fieldsvalue3"

				val1, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val1)

				err = editor.Delete(key)
				require.NoError(t, err)

				val2, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val2)

				// checking if the original event still has it
				val3, err := event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val3)
			})

			t.Run("nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				val1, err := editor.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				require.Nil(t, val1)

				// checking if the original event still has it
				val2, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, fieldsNestedNestedMap, val2)
			})

			t.Run("nested map twice", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map"

				err := editor.Delete(key)
				require.NoError(t, err)

				err = editor.Delete(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// checking if the original event still has it
				val1, err := event.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, fieldsNestedNestedMap, val1)
			})
		})
	})

	t.Run("PutValue", func(t *testing.T) {
		t.Run("@timestamp", func(t *testing.T) {
			editor := NewEventEditor(event)
			newTs := time.Now().Add(time.Hour)
			preVal, err := editor.PutValue("@timestamp", newTs)
			require.NoError(t, err)
			require.Equal(t, event.Timestamp, preVal)

			val, err := editor.GetValue("@timestamp")
			require.NoError(t, err)
			require.Equal(t, newTs, val)

			// the original event should not have this change
			val, err = event.GetValue("@timestamp")
			require.NoError(t, err)
			require.Equal(t, event.Timestamp, val)
		})

		t.Run("@metadata", func(t *testing.T) {
			t.Run("metadata itself", func(t *testing.T) {
				editor := NewEventEditor(event)
				prevVal, err := editor.PutValue("@metadata", "some")
				require.Error(t, err)
				require.ErrorIs(t, err, ErrAlterMetadataKey)
				require.Nil(t, prevVal)
			})

			t.Run("new root-level value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.new"
				value := "value"
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("new root-level map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.new"
				value := mapstr.M{
					"some": "value",
				}
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("existing root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.a.b"
				value := mapstr.M{
					"some": "value",
				}
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Equal(t, "c", prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should have the previous value
				val, err = event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, "c", val)
			})

			t.Run("new nested value in existing map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map.new"
				value := "newvalue"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// the original `metaLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("@metadata.metaLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, metadataNestedMap, val)
			})

			t.Run("absolutely new nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.new1.new2.new3"
				value := "newvalue"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("new nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map.new"
				value := mapstr.M{
					"some": "value",
				}

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// the original `metaLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("@metadata.metaLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, metadataNestedMap, val)
			})

			t.Run("replacing nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "@metadata.metaLevel0Map.metaLevel1Map.metaLevel2Value"
				replacingValue := "metavalue3"
				value := "new"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Equal(t, replacingValue, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should have the previous value
				val, err = event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, replacingValue, val)

				// the original `metaLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("@metadata.metaLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, metadataNestedMap, val)
			})
		})

		t.Run("fields", func(t *testing.T) {
			t.Run("new root-level value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "new"
				value := "value"
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("new root-level map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "new"
				value := mapstr.M{
					"some": "value",
				}
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("existing root-level dot-key", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "a.b"
				value := mapstr.M{
					"some": "value",
				}
				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Equal(t, "c", prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should have the previous value
				val, err = event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, "c", val)
			})

			t.Run("new nested value in existing map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map.new"
				value := "newvalue"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// the original `fieldsLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("fieldsLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, fieldsNestedMap, val)
			})

			t.Run("absolutely new nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "new1.new2.new3"
				value := "newvalue"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			})

			t.Run("new nested map", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map.new"
				value := mapstr.M{
					"some": "value",
				}

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Nil(t, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				requireSameMap(t, value, val)

				// the original event should not have it
				_, err = event.GetValue(key)
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)

				// the original `metaLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("fieldsLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, fieldsNestedMap, val)
			})

			t.Run("replacing nested value", func(t *testing.T) {
				editor := NewEventEditor(event)
				key := "fieldsLevel0Map.fieldsLevel1Map.fieldsLevel2Value"
				replacingValue := "fieldsvalue3"
				value := "new"

				prevVal, err := editor.PutValue(key, value)
				require.NoError(t, err)
				require.Equal(t, replacingValue, prevVal)

				val, err := editor.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, value, val)

				// the original event should have the previous value
				val, err = event.GetValue(key)
				require.NoError(t, err)
				require.Equal(t, replacingValue, val)

				// the original `metaLevel0Map` should be cloned/checkedout now
				val, err = editor.GetValue("fieldsLevel0Map")
				require.NoError(t, err)
				requireNotSameMap(t, fieldsNestedMap, val)
			})
		})

		t.Run("type conflict", func(t *testing.T) {
			event := &Event{
				Meta: mapstr.M{
					"a": 9,
					"c": 10,
				},
				Fields: mapstr.M{
					"a": 9,
					"c": 10,
				},
			}

			editor := NewEventEditor(event)
			_, err := editor.PutValue("a.c", 10)
			require.Error(t, err)
			require.Equal(t, "expected map but type is int", err.Error())
			_, err = editor.PutValue("a.value", 9)
			require.Error(t, err)
			require.Equal(t, "expected map but type is int", err.Error())
		})
	})

	t.Run("Apply", func(t *testing.T) {
		// we're going to make changes, so working with the clone in this test
		cloned := event.Clone()
		// need later for address comparison
		metaNested := cloned.Meta["metaLevel0Map"]
		metaUntouched := cloned.Meta["metaUntouchedMap"]
		fieldsNested := cloned.Fields["fieldsLevel0Map"]
		fieldsUntouched := cloned.Fields["fieldsUntouchedMap"]

		editor := NewEventEditor(cloned)

		// verify that it does nothing without pending changes
		editor.Apply()
		requireClonedMaps(t, event.Meta, cloned.Meta)
		requireClonedMaps(t, event.Fields, cloned.Fields)

		keysToDelete := []string{
			"@metadata.metaLevel0Map.metaLevel1Map.metaLevel2Value",
			"@metadata.metaLevel0Value",
			"fieldsLevel0Map.fieldsLevel1Map.fieldsLevel2Value",
			"fieldsLevel0Value",
		}
		for _, key := range keysToDelete {
			err := editor.Delete(key)
			require.NoError(t, err)
		}
		newTs := time.Now().Add(time.Hour)
		keysToPut := map[string]interface{}{
			"@timestamp": newTs,
			"@metadata.metaLevel0Map.metaLevel1Map.new1": "newmetavalue1",
			"@metadata.metaLevel0Value":                  "metareplaced1",
			"@metadata.new2":                             "newmetavalue2",
			"fieldsLevel0Map.fieldsLevel1Map.new3":       "newfieldsvalue1",
			"new4":                                       "newfieldsvalue2",
			"fieldsLevel0Value":                          "fieldsreplaced1",
		}
		for key, val := range keysToPut {
			_, err := editor.PutValue(key, val)
			require.NoError(t, err)
		}

		// making sure that there are no changes yet
		require.Equal(t, event.Timestamp, cloned.Timestamp)
		requireClonedMaps(t, event.Meta, cloned.Meta)
		requireClonedMaps(t, event.Fields, cloned.Fields)

		expEvent := &Event{
			Timestamp: newTs,
			Meta: mapstr.M{
				"a.b": "c",
				"metaLevel0Map": mapstr.M{
					"metaLevel1Map": mapstr.M{
						"new1": "newmetavalue1",
					},
				},
				"metaLevel0Value":  "metareplaced1",
				"new2":             "newmetavalue2",
				"metaLevel0Value2": "untouched",
				"metaUntouchedMap": metaUntouchedMap,
			},
			Fields: mapstr.M{
				"a.b": "c",
				"fieldsLevel0Map": mapstr.M{
					"fieldsLevel1Map": mapstr.M{
						"new3": "newfieldsvalue1",
					},
				},
				"new4":               "newfieldsvalue2",
				"fieldsLevel0Value":  "fieldsreplaced1",
				"fieldsLevel0Value2": "untouched",
				"fieldsUntouchedMap": fieldsUntouchedMap,
			},
		}

		editor.Apply()

		require.Equal(t, expEvent.Timestamp, cloned.Timestamp)
		require.Equal(t, expEvent.Meta.StringToPrint(), cloned.Meta.StringToPrint())
		require.Equal(t, expEvent.Fields.StringToPrint(), cloned.Fields.StringToPrint())

		// verifying that only the changed nested maps were cloned
		requireNotSameMap(t, metaNested, cloned.Meta["metaLevel0Map"])
		requireNotSameMap(t, fieldsNested, cloned.Fields["fieldsLevel0Map"])
		requireSameMap(t, metaUntouched, cloned.Meta["metaUntouchedMap"])
		requireSameMap(t, fieldsUntouched, cloned.Fields["fieldsUntouchedMap"])
	})

	t.Run("Reset", func(t *testing.T) {
		// we might make changes, so working with the cloned event here
		cloned := event.Clone()
		metaNested := cloned.Meta["metaLevel0Map"]
		fieldsNested := cloned.Fields["fieldsLevel0Map"]
		editor := NewEventEditor(cloned)

		// verify that `Reset` does nothing without pending changes
		editor.Reset()
		requireClonedMaps(t, event.Meta, cloned.Meta)
		requireClonedMaps(t, event.Fields, cloned.Fields)

		keysToDelete := []string{
			"@metadata.metaLevel0Map.metaLevel1Map.metaLevel2Value",
			"@metadata.metaLevel0Value",
			"fieldsLevel0Map.fieldsLevel1Map.fieldsLevel2Value",
			"fieldsLevel0Value",
		}
		for _, key := range keysToDelete {
			err := editor.Delete(key)
			require.NoError(t, err)
		}
		newTs := time.Now().Add(time.Hour)
		keysToPut := map[string]interface{}{
			"@timestamp": newTs,
			"@metadata.metaLevel0Map.metaLevel1Map.new1": "newmetavalue1",
			"@metadata.metaLevel0Value":                  "metareplaced1",
			"@metadata.new2":                             "newmetavalue2",
			"fieldsLevel0Map.fieldsLevel1Map.new3":       "newfieldsvalue1",
			"new4":                                       "newfieldsvalue2",
			"fieldsLevel0Value":                          "fieldsreplaced1",
		}
		for key, val := range keysToPut {
			_, err := editor.PutValue(key, val)
			require.NoError(t, err)
		}

		// making sure that there are no changes yet
		require.Equal(t, event.Timestamp, cloned.Timestamp)
		requireClonedMaps(t, event.Meta, cloned.Meta)
		requireClonedMaps(t, event.Fields, cloned.Fields)

		editor.Reset()

		require.Equal(t, event.Timestamp, cloned.Timestamp)
		requireClonedMaps(t, event.Meta, cloned.Meta)
		requireClonedMaps(t, event.Fields, cloned.Fields)

		// verifying that the nested maps were not cloned
		requireSameMap(t, metaNested, cloned.Meta["metaLevel0Map"])
		requireSameMap(t, fieldsNested, cloned.Fields["fieldsLevel0Map"])

		// verify that deletions are reset and every value is back
		for _, key := range keysToDelete {
			_, err := editor.GetValue(key)
			require.NoError(t, err)
		}
		// verify that all the edits are reset
		for key, val := range keysToPut {
			got, err := editor.GetValue(key)
			if strings.Contains(key, "new") {
				require.Error(t, err)
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
				continue
			}
			require.NoError(t, err)
			require.NotEqual(t, val, got)
		}
	})

	t.Run("DeepUpdate", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			cloned := event.Clone()
			editor := NewEventEditor(cloned)
			editor.DeepUpdate(nil)
			editor.Apply()
			requireClonedMaps(t, event.Meta, cloned.Meta)
			requireClonedMaps(t, event.Fields, cloned.Fields)
		})

		newTs := time.Now().Add(time.Hour)
		update := map[string]interface{}{
			"@timestamp": newTs,
			"@metadata": map[string]interface{}{
				"metaLevel0Map": mapstr.M{ // mix types on purpose, should support both
					"metaLevel1Map": map[string]interface{}{
						"new1": "newmetavalue1",
					},
				},
				"metaLevel0Value": "metareplaced1",
				"new2":            "newmetavalue2",
			},
			"fieldsLevel0Map": map[string]interface{}{
				"fieldsLevel1Map": mapstr.M{
					"new3": "newfieldsvalue1",
				},
				"newmap": map[string]interface{}{
					"new4": "newfieldsvalue2",
				},
			},
			"fieldsLevel0Value": "fieldsreplaced1",
		}

		t.Run("overwrite", func(t *testing.T) {
			cloned := event.Clone()
			editor := NewEventEditor(cloned)
			editor.DeepUpdate(update)

			expEvent := &Event{
				Timestamp: newTs,
				Meta: mapstr.M{
					"a.b": "c",
					"metaLevel0Map": mapstr.M{
						"metaLevel1Map": mapstr.M{
							"metaLevel2Value": "metavalue3",
							"new1":            "newmetavalue1",
						},
					},
					"metaLevel0Value":  "metareplaced1",
					"metaLevel0Value2": "untouched",
					"new2":             "newmetavalue2",
					"metaUntouchedMap": metaUntouchedMap,
				},
				Fields: mapstr.M{
					"a.b": "c",
					"fieldsLevel0Map": mapstr.M{
						"fieldsLevel1Map": mapstr.M{
							"fieldsLevel2Value": "fieldsvalue3",
							"new3":              "newfieldsvalue1",
						},
						"newmap": map[string]interface{}{
							"new4": "newfieldsvalue2",
						},
					},
					"fieldsLevel0Value":  "fieldsreplaced1",
					"fieldsLevel0Value2": "untouched",
					"fieldsUntouchedMap": fieldsUntouchedMap,
				},
			}

			// edited nested maps in metadata and fields should be checked out
			requireNotSameMap(t, cloned.Meta["metaLevel0Map"], editor.pending.Meta["metaLevel0Map"])
			requireNotSameMap(t, cloned.Fields["fieldsLevel0Map"], editor.pending.Fields["fieldsLevel0Map"])
			require.Nil(t, editor.pending.Meta["metaUntouchedMap"])
			require.Nil(t, editor.pending.Fields["fieldsUntouchedMap"])

			editor.Apply()

			require.Equal(t, expEvent.Timestamp, cloned.Timestamp)
			requireClonedMaps(t, expEvent.Meta, cloned.Meta)
			requireClonedMaps(t, expEvent.Fields, cloned.Fields)
		})

		t.Run("no overwrite", func(t *testing.T) {
			cloned := event.Clone()
			editor := NewEventEditor(cloned)
			editor.DeepUpdateNoOverwrite(update)

			expEvent := &Event{
				// should have the original/non-overwritten timestamp value
				Timestamp: event.Timestamp,
				Meta: mapstr.M{
					"a.b": "c",
					"metaLevel0Map": mapstr.M{
						"metaLevel1Map": mapstr.M{
							"metaLevel2Value": "metavalue3",
							"new1":            "newmetavalue1",
						},
					},
					"metaLevel0Value":  "metavalue1",
					"metaLevel0Value2": "untouched",
					"new2":             "newmetavalue2",
					"metaUntouchedMap": metaUntouchedMap,
				},
				Fields: mapstr.M{
					"a.b": "c",
					"fieldsLevel0Map": mapstr.M{
						"fieldsLevel1Map": mapstr.M{
							"fieldsLevel2Value": "fieldsvalue3",
							"new3":              "newfieldsvalue1",
						},
						"newmap": map[string]interface{}{
							"new4": "newfieldsvalue2",
						},
					},
					"fieldsLevel0Value":  "fieldsvalue1",
					"fieldsLevel0Value2": "untouched",
					"fieldsUntouchedMap": fieldsUntouchedMap,
				},
			}

			// only edited nested maps in metadata and fields should be checked out
			requireNotSameMap(t, cloned.Meta["metaLevel0Map"], editor.pending.Meta["metaLevel0Map"])
			requireNotSameMap(t, cloned.Fields["fieldsLevel0Map"], editor.pending.Fields["fieldsLevel0Map"])
			require.Nil(t, editor.pending.Meta["metaUntouchedMap"])
			require.Nil(t, editor.pending.Fields["fieldsUntouchedMap"])

			editor.Apply()

			require.Equal(t, expEvent.Timestamp, cloned.Timestamp)
			requireClonedMaps(t, expEvent.Meta, cloned.Meta)
			requireClonedMaps(t, expEvent.Fields, cloned.Fields)
		})
	})

	t.Run("hierarchy", func(t *testing.T) {
		event := &Event{
			Fields: mapstr.M{
				"a.b": 1,
			},
		}
		editor := NewEventEditor(event)
		err := editor.Delete("a.b")
		require.NoError(t, err)

		prev, err := editor.PutValue("a.b.c", 1)
		require.NoError(t, err)
		require.Nil(t, prev)

		expFields := mapstr.M{
			"a": mapstr.M{
				"b": mapstr.M{
					"c": 1,
				},
			},
		}

		editor.Apply()
		requireClonedMaps(t, expFields, event.Fields)
	})
}

func requireClonedMaps(t *testing.T, expected, actual interface{}) {
	t.Helper()
	requireNotSameMap(t, expected, actual)
	require.IsType(t, mapstr.M{}, expected)
	require.IsType(t, mapstr.M{}, actual)

	expectedMap := expected.(mapstr.M)
	actualMap := actual.(mapstr.M)

	require.Equal(t, expectedMap.StringToPrint(), actualMap.StringToPrint())
}

func requireSameMap(t *testing.T, expected, actual interface{}) {
	t.Helper()
	expectedAddr := fmt.Sprintf("%p", expected)
	actualAddr := fmt.Sprintf("%p", actual)
	require.Equalf(t, expectedAddr, actualAddr, "should reference the same map: %s != %s", expectedAddr, actualAddr)
}

func requireNotSameMap(t *testing.T, expected, actual interface{}) {
	t.Helper()
	expectedAddr := fmt.Sprintf("%p", expected)
	actualAddr := fmt.Sprintf("%p", actual)
	require.NotEqualf(t, expectedAddr, actualAddr, "should reference different maps: %s == %s", expectedAddr, actualAddr)
}

func requireMapValues(t *testing.T, e *EventEditor, expected map[string]interface{}) {
	t.Helper()
	for key := range expected {
		val, err := e.GetValue(key)
		require.NoError(t, err)
		require.Equal(t, expected[key], val)
	}
}
