// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"reflect"
	"testing"
)

func TestJsonNorm(t *testing.T) {
	type args struct {
		key    string
		mapStr interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "will return string value from interface",
			args: args{
				key: "a",
				mapStr: map[string]interface{}{
					"a": "a_value",
				},
			},
			want:    "a_value",
			wantErr: false,
		},
		{
			name: "will return interface value from interface",
			args: args{
				key: "a",
				mapStr: map[string]interface{}{
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
			name: "can not pass wrong key in key parameter",
			args: args{
				key: "b",
				mapStr: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "b_value",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "can not pass mapStr other then type map[string]interface{} in key parameter",
			args: args{
				key: "a",
				mapStr: []interface{}{
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
			got, err := jsonNorm(tt.args.key, tt.args.mapStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonNorm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonNorm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonArr(t *testing.T) {
	type args struct {
		key            string
		arrayInterface []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			name: "will return array of interface values from interface",
			args: args{
				key: "a",
				arrayInterface: []interface{}{
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
			name: "will return array of embedded interface values from interface",
			args: args{
				key: "a",
				arrayInterface: []interface{}{
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
			name: "can not collect values from non-existing key",
			args: args{
				key: "b",
				arrayInterface: []interface{}{
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
			got, err := jsonArr(tt.args.key, tt.args.arrayInterface)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonArr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonArr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		rawJSON string
		key     string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "will return string value from string json",
			args: args{
				rawJSON: `{"a": "a_value"}`,
				key:     "a",
			},
			want:    []string{"a_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded json",
			args: args{
				rawJSON: `{"a": {"b": "b_value"}}`,
				key:     "a.b",
			},
			want:    []string{"b_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				rawJSON: `{"a": [{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]}`,
				key:     "a.#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string Array json",
			args: args{
				rawJSON: `[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`,
				key:     "#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				rawJSON: `{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`,
				key:     "a.#.b.c",
			},
			want:    []string{"c_value_1", "c_value_2", "c_value_3"},
			wantErr: false,
		},
		{
			name: "can not extract value using unstructured key",
			args: args{
				rawJSON: `{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`,
				key:     "a.b.c",
			},
			wantErr: true,
		},
		{
			name: "can not extract value using unstructured key",
			args: args{
				rawJSON: `{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`,
				key:     "a.b.#.c",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.args.rawJSON, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseInterface(t *testing.T) {
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
			name: "will return string value from string json",
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
			name: "will return string value from string embedded json",
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
			name: "will return string value from string embedded Array json",
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
			name: "will return string value from string Array json",
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
			name: "will return string value from string embedded Array json",
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
			name: "can not extract value using unstructured key",
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
			name: "can not extract value using unstructured key",
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
			gotValues, err := parseInterface(tt.args.data, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotValues, tt.wantValues) {
				t.Errorf("parseInterface() = %v, want %v", gotValues, tt.wantValues)
			}
		})
	}
}
