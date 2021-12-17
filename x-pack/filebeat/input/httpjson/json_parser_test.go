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
		key string
		inf interface{}
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
				inf: map[string]interface{}{
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
				inf: map[string]interface{}{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonNorm(tt.args.key, tt.args.inf)
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
		key string
		inf interface{}
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
				inf: []interface{}{
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
				inf: []interface{}{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonArr(tt.args.key, tt.args.inf)
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
		jsonbyte string
		keyField string
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
				jsonbyte: `{"a": "a_value"}`,
				keyField: "a",
			},
			want:    []string{"a_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded json",
			args: args{
				jsonbyte: `{"a": {"b": "b_value"}}`,
				keyField: "a.b",
			},
			want:    []string{"b_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				jsonbyte: `{"a": [{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]}`,
				keyField: "a.#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string Array json",
			args: args{
				jsonbyte: `[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`,
				keyField: "#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				jsonbyte: `{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`,
				keyField: "a.#.b.c",
			},
			want:    []string{"c_value_1", "c_value_2", "c_value_3"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.args.jsonbyte, tt.args.keyField)
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

func TestJsonInterface(t *testing.T) {
	type args struct {
		key       string
		comkey    string
		jsonbytes []byte
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "will return interface value from bytes",
			args: args{
				key:       "{",
				comkey:    "a",
				jsonbytes: []byte(`{"a":"a_value"}`),
			},
			want: map[string]interface{}{
				"a": "a_value",
			},
			wantErr: false,
		},
		{
			name: "will return embedded interface value from bytes",
			args: args{
				key:       "{",
				comkey:    "a",
				jsonbytes: []byte(`{"a": {"b": "b_value"}}`),
			},
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "b_value",
				},
			},
			wantErr: false,
		},
		{
			name: "can not use # if json value is normal json",
			args: args{
				key:       "{",
				comkey:    "#",
				jsonbytes: []byte(`{"a": {"b": "b_value"}}`),
			},
			wantErr: true,
		},
		{
			name: "will return []interface{} if value is array of json",
			args: args{
				key:       "[",
				comkey:    "#",
				jsonbytes: []byte(`[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`),
			},
			want: []interface{}{
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonInterface(tt.args.key, tt.args.comkey, tt.args.jsonbytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}
