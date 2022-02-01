// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"reflect"
	"testing"
)

func TestGetMapValue(t *testing.T) {
	type args struct {
		key  string
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "will return map from input data",
			args: args{
				key: "a",
				data: map[string]interface{}{
					"a": "a_value",
				},
			},
			want:    "a_value",
			wantErr: false,
		},
		{
			name: "will return map from input data",
			args: args{
				key: "a",
				data: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "b_value",
					},
				},
			},
			want: map[string]interface{}{
				"b": "b_value",
			},
			wantErr: false,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				key: "b",
				data: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "b_value",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if data is not type map[string]interface{}",
			args: args{
				key: "a",
				data: []interface{}{
					map[string]interface{}{
						"a": "a_value_1",
					},
					map[string]interface{}{
						"a": "a_value_2",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMapValue(tt.args.key, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMapValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMapValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArrayValue(t *testing.T) {
	type args struct {
		key   string
		array []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			name: "will return array of maps from input JSON",
			args: args{
				key: "a",
				array: []interface{}{
					map[string]interface{}{
						"a": "a_value_1",
					},
					map[string]interface{}{
						"a": "a_value_2",
					},
				},
			},
			want: []interface{}{
				"a_value_1",
				"a_value_2",
			},
			wantErr: false,
		},
		{
			name: "will return array of maps from input JSON",
			args: args{
				key: "a",
				array: []interface{}{
					map[string]interface{}{
						"a": map[string]interface{}{
							"b": "b_value",
						},
					},
					map[string]interface{}{
						"a": map[string]interface{}{
							"b": "b_value",
						},
					},
				},
			},
			want: []interface{}{
				map[string]interface{}{
					"b": "b_value",
				},
				map[string]interface{}{
					"b": "b_value",
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				key: "b",
				array: []interface{}{
					map[string]interface{}{
						"a": "a_value_1",
					},
					map[string]interface{}{
						"a": "a_value_2",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getArrayValue(tt.args.key, tt.args.array)
			if (err != nil) != tt.wantErr {
				t.Errorf("getArrayValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArrayValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKeyedArrayValue(t *testing.T) {
	type args struct {
		rawData []byte
		key     string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`{"a":[{"b":1},{"b":2},{"b":3},{"b":4},{"b":5},{"b":6}]}`),
				key:     "a.#.b",
			},
			want:    []string{"1", "2", "3", "4", "5", "6"},
			wantErr: false,
		},
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`{"a": "a_value"}`),
				key:     "a",
			},
			want:    []string{"a_value"},
			wantErr: false,
		},
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`{"a": {"b": "b_value"}}`),
				key:     "a.b",
			},
			want:    []string{"b_value"},
			wantErr: false,
		},
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`{"a": [{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]}`),
				key:     "a.#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`),
				key:     "#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return a array of string from JSON",
			args: args{
				rawData: []byte(`{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`),
				key:     "a.#.b.c",
			},
			want:    []string{"c_value_1", "c_value_2", "c_value_3"},
			wantErr: false,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				rawData: []byte(`{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`),
				key:     "a.b.c",
			},
			wantErr: true,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				rawData: []byte(`{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`),
				key:     "a.b.#.c",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getKeyedArrayValue(tt.args.rawData, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKeyedArrayValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getKeyedArrayValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKeyedInterfaceValues(t *testing.T) {
	type args struct {
		data interface{}
		key  string
	}
	tests := []struct {
		name       string
		args       args
		wantValues []string
		wantErr    bool
	}{
		{
			name: "will return array of string from input data",
			args: args{
				data: map[string]interface{}{
					"a": "a_value",
				},
				key: "a",
			},
			wantValues: []string{"a_value"},
			wantErr:    false,
		},
		{
			name: "will return array of string from input data",
			args: args{
				data: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "b_value",
					},
				},
				key: "a.b",
			},
			wantValues: []string{"b_value"},
			wantErr:    false,
		},
		{
			name: "will return array of string from input data",
			args: args{
				data: map[string]interface{}{
					"a": []interface{}{
						map[string]interface{}{
							"b": "b_value_1",
						},
						map[string]interface{}{
							"b": "b_value_2",
						},
						map[string]interface{}{
							"b": "b_value_3",
						},
					},
				},
				key: "a.#.b",
			},
			wantValues: []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr:    false,
		},
		{
			name: "will return array of string from input data",
			args: args{
				data: []interface{}{
					map[string]interface{}{
						"b": "b_value_1",
					},
					map[string]interface{}{
						"b": "b_value_2",
					},
					map[string]interface{}{
						"b": "b_value_3",
					},
				},
				key: "#.b",
			},
			wantValues: []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr:    false,
		},
		{
			name: "will return array of string from input data",
			args: args{
				data: map[string]interface{}{
					"a": []interface{}{
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_1",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_2",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_3",
							},
						},
					},
				},
				key: "a.#.b.c",
			},
			wantValues: []string{"c_value_1", "c_value_2", "c_value_3"},
			wantErr:    false,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				data: map[string]interface{}{
					"a": []interface{}{
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_1",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_2",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_3",
							},
						},
					},
				},
				key: "a.b.c",
			},
			wantErr: true,
		},
		{
			name: "returns an error if the key is not found",
			args: args{
				data: map[string]interface{}{
					"a": []interface{}{
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_1",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_2",
							},
						},
						map[string]interface{}{
							"b": map[string]interface{}{
								"c": "c_value_3",
							},
						},
					},
				},
				key: "a.b.#.c",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValues, err := getKeyedInterfaceValues(tt.args.data, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKeyedInterfaceValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotValues, tt.wantValues) {
				t.Errorf("getKeyedInterfaceValues() = %v, want %v", gotValues, tt.wantValues)
			}
		})
	}
}
