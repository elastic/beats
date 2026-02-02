// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"reflect"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
)

type constraintArgs struct {
	columnName string
	operator   table.Operator
	value      string
}

func getQueryContext(constraints []constraintArgs) table.QueryContext {
	queryContext := table.QueryContext{
		Constraints: make(map[string]table.ConstraintList),
	}

	for _, constraint := range constraints {
		columnName := constraint.columnName
		constraintList, ok := queryContext.Constraints[columnName]
		if !ok {
			constraintList = table.ConstraintList{
				Affinity:    table.ColumnTypeText,
				Constraints: []table.Constraint{},
			}
		}
		constraintList.Constraints = append(constraintList.Constraints, table.Constraint{
			Operator:   constraint.operator,
			Expression: constraint.value,
		})
		queryContext.Constraints[columnName] = constraintList
	}
	return queryContext
}

func TestGetConstraintFilters(t *testing.T) {
	tests := []struct {
		name         string
		queryContext table.QueryContext
		want         []Filter
	}{
		{
			name: "test_equals",
			queryContext: getQueryContext([]constraintArgs{
				{
					columnName: "program_id",
					operator:   table.OperatorEquals,
					value:      "1234567890",
				},
			}),
			want: []Filter{
				{ColumnName: "program_id", Operator: table.OperatorEquals, Expression: "1234567890"},
			},
		},
		{
			name: "test_like",
			queryContext: getQueryContext([]constraintArgs{
				{
					columnName: "program_id",
					operator:   table.OperatorLike,
					value:      "1234567890",
				},
			}),
			want: []Filter{
				{ColumnName: "program_id", Operator: table.OperatorLike, Expression: "1234567890"},
			},
		},
		{
			name: "test_match",
			queryContext: getQueryContext([]constraintArgs{
				{
					columnName: "program_id",
					operator:   table.OperatorMatch,
					value:      "test match",
				},
			}),
			want: []Filter{
				{ColumnName: "program_id", Operator: table.OperatorMatch, Expression: "test match"},
			},
		},
		{
			name: "test_multiple_filters",
			queryContext: getQueryContext([]constraintArgs{
				{
					columnName: "program_id",
					operator:   table.OperatorEquals,
					value:      "1234567890",
				},
				{
					columnName: "program_id",
					operator:   table.OperatorLike,
					value:      "1234567890",
				},
			}),
			want: []Filter{
				{ColumnName: "program_id", Operator: table.OperatorEquals, Expression: "1234567890"},
				{ColumnName: "program_id", Operator: table.OperatorLike, Expression: "1234567890"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConstraintFilters(tt.queryContext); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: GetConstraintFilters() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

type (
	CustomInt    int
	CustomFloat  float64
	CustomBool   bool
	CustomString string
)

type EmbeddedStruct struct {
	EmbeddedString string  `osquery:"embedded_string"`
	EmbeddedInt    int     `osquery:"embedded_int"`
	EmbeddedFloat  float64 `osquery:"embedded_float"`
	EmbeddedBool   bool    `osquery:"embedded_bool"`
}

type MockEntry struct {
	StringValue       string       `osquery:"string_value"`
	IntValue          int          `osquery:"int_value"`
	FloatValue        float64      `osquery:"float_value"`
	BoolValue         bool         `osquery:"bool_value"`
	CustomIntValue    CustomInt    `osquery:"custom_int_value"`
	CustomFloatValue  CustomFloat  `osquery:"custom_float_value"`
	CustomBoolValue   CustomBool   `osquery:"custom_bool_value"`
	CustomStringValue CustomString `osquery:"custom_string_value"`

	EmbeddedStruct
}

type FilterTestCase struct {
	name   string
	entry  MockEntry
	filter Filter
	want   bool
}

func RunFilterTests(t *testing.T, tests []FilterTestCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.entry)
			if got != tt.want {
				t.Errorf("%s: Matches() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestFilter_Equals(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "string_equals_true",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorEquals, Expression: "Mock Entry"},
			want:   true,
		},
		{
			name: "string_equals_false",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorEquals, Expression: "NOtEquals"},
			want:   false,
		},
		{
			name: "int_equals_true",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "int_equals_false",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorEquals, Expression: "101"},
			want:   false,
		},
		{
			name: "float_equals_true",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "float_equals_false",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorEquals, Expression: "100.1"},
			want:   false,
		},
		{
			name: "bool_equals_true",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorEquals, Expression: "true"},
			want:   true,
		},
		{
			name: "bool_equals_false",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorEquals, Expression: "false"},
			want:   false,
		},
		{
			name: "bool_equals_embedded_string",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedString: "embeddedString"},
			},
			filter: Filter{ColumnName: "embedded_string", Operator: table.OperatorEquals, Expression: "embeddedString"},
			want:   true,
		},
		{
			name: "bool_equals_embedded_int",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 42},
			},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorEquals, Expression: "42"},
			want:   true,
		},
		{
			name: "bool_equals_embedded_float",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 3.14},
			},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorEquals, Expression: "3.14"},
			want:   true,
		},
		{
			name: "bool_equals_embedded_bool",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedBool: true},
			},
			filter: Filter{ColumnName: "embedded_bool", Operator: table.OperatorEquals, Expression: "true"},
			want:   true,
		},
	}
	RunFilterTests(t, tests)
}

func TestFilter_GreaterThan(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "int_greater_than_true",
			entry: MockEntry{
				IntValue: 100,
			},
		},
		{
			name: "int_greater_than_false",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThan, Expression: "101"},
			want:   false,
		},
		{
			name: "float_greater_than_true",
			entry: MockEntry{
				FloatValue: 100.12,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.01"},
			want:   true,
		},
		{
			name: "float_greater_than_false",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.0"},
			want:   false,
		},
		{
			name: "float_greater_than_false",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.00"},
			want:   false,
		},
		{
			name: "bool_greater_than_true",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorGreaterThan, Expression: "false"},
			want:   true,
		},
		{
			name: "bool_greater_than_false",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorGreaterThan, Expression: "true"},
			want:   false,
		},
		{
			name: "string_greater_than_false_when_not_a_number",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGreaterThan, Expression: "Mock Entry"},
			want:   false,
		},
		{
			name: "string_greater_than_always_false",
			entry: MockEntry{
				StringValue: "101",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGreaterThan, Expression: "100"},
			want:   false,
		},
		{
			name:   "embedded_int_greater_than_true",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 200}},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorGreaterThan, Expression: "100"},
			want:   true,
		},
		{
			name:   "embedded_int_greater_than_false",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 50}},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorGreaterThan, Expression: "100"},
			want:   false,
		},
		{
			name:   "embedded_float_greater_than_true",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 150.5}},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorGreaterThan, Expression: "100.0"},
			want:   true,
		},
		{
			name:   "embedded_float_greater_than_false",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 75.25}},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorGreaterThan, Expression: "100.0"},
			want:   false,
		},
	}
	RunFilterTests(t, tests)
}

func TestFilter_LessThan(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "int_less_than_true",
			entry: MockEntry{
				IntValue: 100,
			},
		},
		{
			name: "int_less_than_false",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThan, Expression: "99"},
			want:   false,
		},
		{
			name: "float_less_than_true",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "100.01"},
			want:   true,
		},
		{
			name: "float_less_than_false",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "100.0"},
			want:   false,
		},
		{
			name: "float_less_than_false",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "99.99"},
			want:   false,
		},
		{
			name: "bool_less_than_true",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorLessThan, Expression: "false"},
			want:   false,
		},
		{
			name: "bool_less_than_false",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorLessThan, Expression: "true"},
			want:   false,
		},
		{
			name: "string_less_than_false_when_not_a_number",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLessThan, Expression: "Mock Entry"},
			want:   false,
		},
		{
			name: "string_less_than_always_false",
			entry: MockEntry{
				StringValue: "99",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLessThan, Expression: "100"},
			want:   false,
		},
		{
			name:   "embedded_int_less_than_true",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 50}},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorLessThan, Expression: "100"},
			want:   true,
		},
		{
			name:   "embedded_int_less_than_false",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 200}},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorLessThan, Expression: "100"},
			want:   false,
		},
		{
			name:   "embedded_float_less_than_true",
			entry:  MockEntry{EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 75.25}},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorLessThan, Expression: "100.0"},
			want:   true,
		},
	}
	RunFilterTests(t, tests)
}

func TestGreaterThanOrEquals(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "int_greater_than_or_equals_true_greater",
			entry: MockEntry{
				IntValue: 101,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "int_greater_than_or_equals_true_equals",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "int_greater_than_or_equals_false",
			entry: MockEntry{
				IntValue: 99,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want:   false,
		},
		{
			name: "float_greater_than_or_equals_true_greater",
			entry: MockEntry{
				FloatValue: 100.01,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "float_greater_than_or_equals_true_equals",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "float_greater_than_or_equals_false",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want:   false,
		},
		{
			name: "embedded_int_greater_than_or_equals_true_equals",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 100},
			},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "embedded_int_greater_than_or_equals_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 99},
			},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want:   false,
		},
		{
			name: "embedded_float_greater_than_or_equals_true_equals",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 100.0},
			},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "embedded_float_greater_than_or_equals_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 99.99},
			},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want:   false,
		},
	}
	RunFilterTests(t, tests)
}

func TestLessThanOrEquals(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "int_less_than_or_equals_true_less",
			entry: MockEntry{
				IntValue: 99,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "int_less_than_or_equals_true_equals",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "int_less_than_or_equals_false",
			entry: MockEntry{
				IntValue: 101,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want:   false,
		},
		{
			name: "float_less_than_or_equals_true_less",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "float_less_than_or_equals_true_equals",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "float_less_than_or_equals_false",
			entry: MockEntry{
				FloatValue: 100.01,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want:   false,
		},
		{
			name: "embedded_int_less_than_or_equals_true_equals",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 100},
			},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want:   true,
		},
		{
			name: "embedded_int_less_than_or_equals_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedInt: 101},
			},
			filter: Filter{ColumnName: "embedded_int", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want:   false,
		},
		{
			name: "embedded_float_less_than_or_equals_true_equals",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 100.0},
			},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want:   true,
		},
		{
			name: "embedded_float_less_than_or_equals_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedFloat: 100.01},
			},
			filter: Filter{ColumnName: "embedded_float", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want:   false,
		},
	}
	RunFilterTests(t, tests)
}

func TestLike(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "string_like_true_exact_match",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Mock Entry"},
			want:   true,
		},
		{
			name: "string_like_true_wildcard_prefix",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "%Entry"},
			want:   true,
		},
		{
			name: "string_like_true_wildcard_suffix",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Mock%"},
			want:   true,
		},
		{
			name: "string_like_true_wildcard_both",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "%ock Ent%"},
			want:   true,
		},
		{
			name: "string_like_false",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Test%"},
			want:   false,
		},
		{
			name: "string_like_true_single_char_wildcard",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Moc_ Entry"},
			want:   true,
		},
		{
			name: "embedded_string_like_true",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedString: "EmbeddedValue"},
			},
			filter: Filter{ColumnName: "embedded_string", Operator: table.OperatorLike, Expression: "Embedded%"},
			want:   true,
		},
		{
			name: "embedded_string_like_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedString: "EmbeddedValue"},
			},
			filter: Filter{ColumnName: "embedded_string", Operator: table.OperatorLike, Expression: "Other%"},
			want:   false,
		},
	}
	RunFilterTests(t, tests)
}

func TestGlob(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "string_glob_true_exact_match",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Mock Entry"},
			want:   true,
		},
		{
			name: "string_glob_true_wildcard",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Mock*"},
			want:   true,
		},
		{
			name: "string_glob_true_question_mark",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc? Entry"},
			want:   true,
		},
		{
			name: "string_glob_true_character_class",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc[ks] Entry"},
			want:   true,
		},
		{
			name: "string_glob_false_character_class",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc[s] Entry"},
			want:   false,
		},
		{
			name: "string_glob_false",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Test*"},
			want:   false,
		},
		{
			name: "string_glob_true_complex_pattern",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "M*k E*y"},
			want:   true,
		},
		{
			name: "embedded_string_glob_true",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedString: "EmbeddedValue"},
			},
			filter: Filter{ColumnName: "embedded_string", Operator: table.OperatorGlob, Expression: "Embedded*"},
			want:   true,
		},
		{
			name: "embedded_string_glob_false",
			entry: MockEntry{
				EmbeddedStruct: EmbeddedStruct{EmbeddedString: "EmbeddedValue"},
			},
			filter: Filter{ColumnName: "embedded_string", Operator: table.OperatorGlob, Expression: "Other*"},
			want:   false,
		},
	}
	RunFilterTests(t, tests)
}

func TestFilter_CustomTypes(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "int_less_than_true",
			entry: MockEntry{
				CustomIntValue: 100,
			},
			filter: Filter{ColumnName: "custom_int_value", Operator: table.OperatorLessThan, Expression: "101"},
			want:   true,
		},
		{
			name: "float_greater_than_true",
			entry: MockEntry{
				CustomFloatValue: 100.0,
			},
			filter: Filter{ColumnName: "custom_float_value", Operator: table.OperatorGreaterThan, Expression: "99.0"},
			want:   true,
		},
		{
			name: "string_equals_true",
			entry: MockEntry{
				CustomStringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "custom_string_value", Operator: table.OperatorEquals, Expression: "Mock Entry"},
			want:   true,
		},
		{
			name: "string_equals_false",
			entry: MockEntry{
				CustomStringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "custom_string_value", Operator: table.OperatorEquals, Expression: "Other Entry"},
			want:   false,
		},

		{
			name: "bool_equals_true",
			entry: MockEntry{
				CustomBoolValue: true,
			},
			filter: Filter{ColumnName: "custom_bool_value", Operator: table.OperatorEquals, Expression: "true"},
			want:   true,
		},
		{
			name: "bool_equals_false",
			entry: MockEntry{
				CustomBoolValue: false,
			},
			filter: Filter{ColumnName: "custom_bool_value", Operator: table.OperatorEquals, Expression: "false"},
			want:   true,
		},
		{
			name: "string_glob_true_wildcard",
			entry: MockEntry{
				CustomStringValue: "Value123",
			},
			filter: Filter{ColumnName: "custom_string_value", Operator: table.OperatorGlob, Expression: "Value*"},
			want:   true,
		},
		{
			name: "string_like_false",
			entry: MockEntry{
				CustomStringValue: "Value123",
			},
			filter: Filter{ColumnName: "custom_string_value", Operator: table.OperatorLike, Expression: "%Value4%"},
			want:   false,
		},
		{
			name: "string_like_true",
			entry: MockEntry{
				CustomStringValue: "Value123",
			},
			filter: Filter{ColumnName: "custom_string_value", Operator: table.OperatorLike, Expression: "%Value%"},
			want:   true,
		},
	}
	RunFilterTests(t, tests)
}
