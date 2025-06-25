// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"reflect"
	"testing"
)

func TestConvertObjectArrayToMapArray(t *testing.T) {
	type testStruct struct {
		Name   string `json:"name"`
		Value  int    `json:"value"`
		Active bool   `json:"active"`
	}

	tests := []struct {
		name  string
		input []testStruct
		want  []map[string]any
	}{
		{
			name:  "nil slice",
			input: nil,
			want:  []map[string]any{},
		},
		{
			name:  "empty slice",
			input: []testStruct{},
			want:  []map[string]any{},
		},
		{
			name: "single item",
			input: []testStruct{
				{Name: "A", Value: 1, Active: true},
			},
			want: []map[string]any{
				{"name": "A", "value": float64(1), "active": true},
			},
		},
		{
			name: "multiple items",
			input: []testStruct{
				{Name: "A", Value: 1, Active: true},
				{Name: "B", Value: 2, Active: false},
			},
			want: []map[string]any{
				{"name": "A", "value": float64(1), "active": true},
				{"name": "B", "value": float64(2), "active": false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertObjectArrayToMapArray(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertObjectArrayToMapArray() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestConvertObjectArrayToMapArray_MarshalError(t *testing.T) {
	type errorStruct struct {
		C chan int `json:"c,omitempty"`
	}

	input := []any{
		errorStruct{C: make(chan int)}, // This will cause json.Marshal to fail
		map[string]any{},               // This will be an empty object
	}

	got := convertObjectArrayToMapArray(input)
	want := []map[string]any{
		{},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("convertObjectArrayToMapArray() with marshal error got = %#v, want %#v", got, want)
	}
}
