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
				Affinity: table.ColumnTypeText,
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
	type args struct {
		queryContext table.QueryContext
		columnName   string
	}
	tests := []struct {
		name         string
		queryContext table.QueryContext
		want         []Filter
	}{
		{
			name:       "test_equals",
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
			name:         "test_like",
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
			name:         "test_match",
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
			name:         "test_multiple_filters",
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

type MockEntry struct {
	StringValue string `osquery:"string_value"`
	IntValue int `osquery:"int_value"`
	FloatValue float64 `osquery:"float_value"`
	BoolValue bool `osquery:"bool_value"`
}

type FilterTestCase struct {
	name string
	entry MockEntry
	filter Filter
	want bool
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
			name: "1 - StringEquals - True",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorEquals, Expression: "Mock Entry"},
			want: true,
		},
		{
			name: "2 - StringEquals - False",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorEquals, Expression: "NOtEquals"},
			want: false,
		},
		{
			name: "3 - IntEquals - True",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorEquals, Expression: "100"},
			want: true,
		},
		{
			name: "4 - IntEquals - False",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorEquals, Expression: "101"},
			want: false,
		},
		{
			name: "5 - FloatEquals - True",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorEquals, Expression: "100.0"},
			want: true,
		},
		{
			name: "6 - FloatEquals - False",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorEquals, Expression: "100.1"},
			want: false,
		},
		{
			name: "7 - BoolEquals - True",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorEquals, Expression: "true"},
			want: true,
		},
		{
			name: "8 - BoolEquals - False",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorEquals, Expression: "false"},
			want: false,
		},
	}
	RunFilterTests(t, tests)
}

func TestFilter_GreaterThan(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - IntGreaterThan - True",
			entry: MockEntry{
				IntValue: 100,
			},
		},
		{
			name: "2 - IntGreaterThan - False",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThan, Expression: "101"},
			want: false,
		},
		{
			name: "3 - FloatGreaterThan - True",
			entry: MockEntry{
				FloatValue: 100.12,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.01"},
			want: true,
		},
		{
			name: "4 - FloatGreaterThan - False",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.0"},
			want: false,
		},
		{
			name: "5 - FloatGreaterThan - False",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThan, Expression: "100.00"},
			want: false,
		},
		{
			name: "5 - BoolGreaterThan - True",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorGreaterThan, Expression: "false"},
			want: true,
		},
		{
			name: "6 - BoolGreaterThan - False",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorGreaterThan, Expression: "true"},
			want: false,
		},
		{
			name: "7 - StringGreaterThan - false when not a number",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGreaterThan, Expression: "Mock Entry"},
			want: false,
		},
		{
			name: "8 - StringGreaterThan - True",
			entry: MockEntry{
				StringValue: "101",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGreaterThan, Expression: "100"},
			want: true,
		},
	}
	RunFilterTests(t, tests)
}

func TestFilter_LessThan(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - IntLessThan - True",
			entry: MockEntry{
				IntValue: 100,
			},
		},
		{
			name: "2 - IntLessThan - False",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThan, Expression: "99"},
			want: false,
		},
		{
			name: "3 - FloatLessThan - True",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "100.01"},
			want: true,
		},
		{
			name: "4 - FloatLessThan - False",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "100.0"},
			want: false,
		},
		{
			name: "5 - FloatLessThan - False",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThan, Expression: "99.99"},
			want: false,
		},
		{
			name: "6 - BoolLessThan - True",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorLessThan, Expression: "false"},
			want: true,
		},
		{
			name: "7 - BoolLessThan - False",
			entry: MockEntry{
				BoolValue: true,
			},
			filter: Filter{ColumnName: "bool_value", Operator: table.OperatorLessThan, Expression: "true"},
			want: false,
		},
		{
			name: "8 - StringLessThan - false when not a number",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLessThan, Expression: "Mock Entry"},
			want: false,
		},
		{
			name: "9 - StringLessThan - True",
			entry: MockEntry{
				StringValue: "99",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLessThan, Expression: "100"},
			want: true,
		},
	}
	RunFilterTests(t, tests)
}

func TestGreaterThanOrEquals(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - IntGreaterThanOrEquals - True (greater)",
			entry: MockEntry{
				IntValue: 101,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want: true,
		},
		{
			name: "2 - IntGreaterThanOrEquals - True (equals)",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want: true,
		},
		{
			name: "3 - IntGreaterThanOrEquals - False",
			entry: MockEntry{
				IntValue: 99,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100"},
			want: false,
		},
		{
			name: "4 - FloatGreaterThanOrEquals - True (greater)",
			entry: MockEntry{
				FloatValue: 100.01,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want: true,
		},
		{
			name: "5 - FloatGreaterThanOrEquals - True (equals)",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want: true,
		},
		{
			name: "6 - FloatGreaterThanOrEquals - False",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorGreaterThanOrEquals, Expression: "100.0"},
			want: false,
		},
	}
	RunFilterTests(t, tests)
}

func TestLessThanOrEquals(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - IntLessThanOrEquals - True (less)",
			entry: MockEntry{
				IntValue: 99,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want: true,
		},
		{
			name: "2 - IntLessThanOrEquals - True (equals)",
			entry: MockEntry{
				IntValue: 100,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want: true,
		},
		{
			name: "3 - IntLessThanOrEquals - False",
			entry: MockEntry{
				IntValue: 101,
			},
			filter: Filter{ColumnName: "int_value", Operator: table.OperatorLessThanOrEquals, Expression: "100"},
			want: false,
		},
		{
			name: "4 - FloatLessThanOrEquals - True (less)",
			entry: MockEntry{
				FloatValue: 99.99,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want: true,
		},
		{
			name: "5 - FloatLessThanOrEquals - True (equals)",
			entry: MockEntry{
				FloatValue: 100.0,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want: true,
		},
		{
			name: "6 - FloatLessThanOrEquals - False",
			entry: MockEntry{
				FloatValue: 100.01,
			},
			filter: Filter{ColumnName: "float_value", Operator: table.OperatorLessThanOrEquals, Expression: "100.0"},
			want: false,
		},
	}
	RunFilterTests(t, tests)
}

func TestLike(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - StringLike - True (exact match)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Mock Entry"},
			want: true,
		},
		{
			name: "2 - StringLike - True (wildcard prefix)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "%Entry"},
			want: true,
		},
		{
			name: "3 - StringLike - True (wildcard suffix)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Mock%"},
			want: true,
		},
		{
			name: "4 - StringLike - True (wildcard both)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "%ock Ent%"},
			want: true,
		},
		{
			name: "5 - StringLike - False",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Test%"},
			want: false,
		},
		{
			name: "6 - StringLike - True (single char wildcard)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorLike, Expression: "Moc_ Entry"},
			want: true,
		},
	}
	RunFilterTests(t, tests)
}

func TestGlob(t *testing.T) {
	tests := []FilterTestCase{
		{
			name: "1 - StringGlob - True (exact match)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Mock Entry"},
			want: true,
		},
		{
			name: "2 - StringGlob - True (wildcard)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Mock*"},
			want: true,
		},
		{
			name: "3 - StringGlob - True (question mark)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc? Entry"},
			want: true,
		},
		{
			name: "4 - StringGlob - True (character class)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc[ks] Entry"},
			want: true,
		},
		{
			name: "4 - StringGlob - False (character class)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Moc[s] Entry"},
			want: false,
		},
		{
			name: "5 - StringGlob - False",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "Test*"},
			want: false,
		},
		{
			name: "6 - StringGlob - True (complex pattern)",
			entry: MockEntry{
				StringValue: "Mock Entry",
			},
			filter: Filter{ColumnName: "string_value", Operator: table.OperatorGlob, Expression: "M*k E*y"},
			want: true,
		},
	}
	RunFilterTests(t, tests)
}
