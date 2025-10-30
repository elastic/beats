// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

package filters

import "testing"

func TestToBool(t *testing.T) {
	type args struct {
		input any
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "bool_true", args: args{input: true}, want: true},
		{name: "bool_false", args: args{input: false}, want: false},
		{name: "bool_string_true", args: args{input: "true"}, want: true},
		{name: "bool_string_false", args: args{input: "false"}, want: false},
		{name: "bool_string_true_uppercase", args: args{input: "TRUE"}, want: true},
		{name: "bool_string_false_uppercase", args: args{input: "FALSE"}, want: false},
		{name: "bool_string_non_boolean", args: args{input: "not a boolean"}, want: false},
		{name: "bool_int_1", args: args{input: 1}, want: true},
		{name: "bool_int_0", args: args{input: 0}, want: false},
		{name: "bool_int_non_boolean", args: args{input: 100}, want: true},
		{name: "bool_float_1.0", args: args{input: 1.0}, want: true},
		{name: "bool_float_0.0", args: args{input: 0.0}, want: false},
		{name: "bool_float_non_boolean", args: args{input: 100.0}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToBool(tt.args.input); if !ok {
				if tt.want == false {
					return
				}
				t.Errorf("%s: ToBool() failed to convert input to bool", tt.name)
			} else if got != tt.want {
				t.Errorf("%s: ToBool() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	type args struct {
		input any
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "int64_1", args: args{input: 1}, want: 1},
		{name: "int64_0", args: args{input: 0}, want: 0},
		{name: "int64_non_integer", args: args{input: 100.0}, want: 100},
		{name: "int64_string_1", args: args{input: "1"}, want: 1},
		{name: "int64_string_0", args: args{input: "0"}, want: 0},
		{name: "int64_string_non_integer", args: args{input: "not an integer"}, want: 0},
		{name: "int64_float_1.0", args: args{input: 1.0}, want: 1},
		{name: "int64_float_0.0", args: args{input: 0.0}, want: 0},
		{name: "int64_float_non_integer", args: args{input: 100.0}, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt64(tt.args.input); if !ok {
				if tt.want == 0 {
					return
				}
				t.Errorf("%s: ToInt64() failed to convert input to int64", tt.name)
			} else if got != tt.want {
				t.Errorf("%s: ToInt64() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	type args struct {
		input any
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{name: "float64_1.0", args: args{input: 1.0}, want: 1.0},
		{name: "float64_0.0", args: args{input: 0.0}, want: 0.0},
		{name: "float64_non_float", args: args{input: 100}, want: 100.0},
		{name: "float64_string_1.0", args: args{input: "1.0"}, want: 1.0},
		{name: "float64_string_0.0", args: args{input: "0.0"}, want: 0.0},
		{name: "float64_string_non_float", args: args{input: "not a float"}, want: 0.0},
		{name: "float64_int_1", args: args{input: 1}, want: 1.0},
		{name: "float64_int_0", args: args{input: 0}, want: 0.0},
		{name: "float64_int_non_float", args: args{input: 100}, want: 100.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToFloat64(tt.args.input); if !ok {
				if tt.want == 0.0 {
					return
				}
				t.Errorf("%s: ToFloat64() failed to convert input to float64", tt.name)
			} else if got != tt.want {
				t.Errorf("%s: ToFloat64() = %v, want %v", tt.name, got, tt.want)
			}
		})	
	}
}
